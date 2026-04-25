package dashboard

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/marcobeierer/go-core/db/entmain"
	mainaccount "github.com/marcobeierer/go-core/db/entmain/account"
	"github.com/marcobeierer/go-core/db/entmain/passkeycredential"
	maintenant "github.com/marcobeierer/go-core/db/entmain/tenant"
	"github.com/marcobeierer/go-core/db/entmain/tenantaccountassignment"

	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	ctxx2 "github.com/marcobeierer/go-core/ctxx"
	account2 "github.com/marcobeierer/go-core/model/account"
	"github.com/marcobeierer/go-core/model/common/mainrole"
	"github.com/marcobeierer/go-core/model/tenant"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/uix/route"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/fileutil"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
)

type DashboardCardsPartialData struct {
}

type DashboardCardsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewDashboardCardsPartial(infra *common.Infra, actions *Actions) *DashboardCardsPartial {
	config := actionx2.NewConfig(
		actions.Route("dashboard-cards-partial"),
		true,
	).EnableSetupSessionAccess()
	return &DashboardCardsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DashboardCardsPartial) Data() *DashboardCardsPartialData {
	return &DashboardCardsPartialData{}
}

func (qq *DashboardCardsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	widget, err := qq.Widget(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		widget,
	)
}

// guidelines:
// most important card should always be in front, for example `set password` if not set yet
// or `manage spaces` if no space exists yet
func (qq *DashboardCardsPartial) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	var grids []*widget.Grid

	var openTaskCards []*widget.Card
	var accountCards []*widget.Card
	var systemCards []*widget.Card
	var systemFooterBtns []*widget.Button

	accountm := account2.NewAccount(ctx.MainCtx().Account)
	passkeyPolicy, err := accountm.PasskeyPolicy(ctx)
	if err != nil {
		log.Println(err)
		passkeyPolicy = account2.NewPasskeyPolicy(false, false, false)
	}
	isTenantPasskeyEnrollmentRequired := passkeyPolicy.IsTenantPasskeyEnrollmentRequired()

	if isTenantPasskeyEnrollmentRequired {
		return qq.setupRequiredWidget(ctx), nil
	}

	passkeyCredentials := ctx.MainCtx().MainTx.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(ctx.MainCtx().Account.ID)).
		AllX(ctx)

	if accountm.Data.Role == mainrole.Admin {
		systemCards = append(systemCards, qq.appStatusCard(ctx))
		systemFooterBtns = append(systemFooterBtns, qq.manageUploadLimitBtn(ctx))
	}
	if accountm.HasPassword() {
		// only if main password is already set
		if accountm.HasTemporaryPassword() {
			openTaskCards = append(openTaskCards, qq.clearTemporaryPasswordCard(ctx))
		}
	} else {
		openTaskCards = append(openTaskCards, qq.setPasswordCard(ctx))
	}

	if len(openTaskCards) > 0 {
		grids = append(grids, &widget.Grid{
			Heading:  widget.H(widget.HeadingTypeTitleMd, widget.T("Open tasks")),
			Children: openTaskCards,
		})
	}

	spacesByTenant, err := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for tenantx, spaces := range spacesByTenant {
		var tenantCards []*widget.Card
		var tenantHeaderBtns []widget.IWidget

		if tenantCard := qq.nilableTenantCard(ctx, tenantx); tenantCard != nil {
			tenantCards = append(tenantCards, tenantCard)
		}
		if quotaUsageCard := qq.nilableQuotaUsageCard(ctx, tenantx); quotaUsageCard != nil {
			tenantCards = append(tenantCards, quotaUsageCard)
		}

		if btn, ok := qq.passkeyEnforcementBtn(ctx, tenantx); ok {
			tenantHeaderBtns = append(tenantHeaderBtns, btn)
		}

		if btn, ok := qq.manageUsersBtn(ctx, tenantx); ok {
			tenantHeaderBtns = append(tenantHeaderBtns, btn)
		}

		if manageSpacesCard, ok := qq.manageSpacesCard(ctx, tenantx, len(spaces)); ok {
			tenantCards = append(tenantCards, manageSpacesCard)
		}
		if btn, ok := qq.manageSpacesBtn(ctx, tenantx); ok {
			tenantHeaderBtns = append(tenantHeaderBtns, btn)
		}
		if btn, ok := qq.deleteTenantBtn(ctx, tenantx); ok {
			tenantHeaderBtns = append(tenantHeaderBtns, btn)
		}
		if link, ok := qq.downloadTenantBackupLink(ctx, tenantx); ok {
			tenantHeaderBtns = append(tenantHeaderBtns, link)
		}
		for _, spacex := range spaces {
			tenantCards = append(tenantCards, qq.spaceCard(ctx, spacex, tenantx))
		}

		grids = append(grids, &widget.Grid{
			Heading: widget.Hf(widget.HeadingTypeTitleMd, "Organization «%s»", tenantx.Name),
			Footer: &widget.Row{
				Wrap:     true,
				Children: tenantHeaderBtns,
			},
			Children: tenantCards,
		})
	}

	var accountCardsBtns []*widget.Button

	if accountm.HasPassword() && !accountm.HasTemporaryPassword() && len(passkeyCredentials) == 0 {
		accountCardsBtns = append(accountCardsBtns, qq.changePasswordBtn(ctx))
	}
	if btn, ok := qq.editAccountBtn(ctx); ok {
		accountCardsBtns = append(accountCardsBtns, btn)
	}

	if len(passkeyCredentials) == 0 {
		accountCards = append(accountCards, &widget.Card{
			Style:          widget.CardStyleFilled,
			Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("No passkeys registered")),
			Subhead:        widget.T("Passkeys"),
			SupportingText: widget.T("Register a passkey to enable passwordless sign in."),
			Actions: []*widget.Button{
				qq.registerPasskeyBtn(ctx, widget.ButtonStyleTypeTonal),
			},
		})
	} else {
		recoveryCodesCount := len(accountm.Data.PasskeyRecoveryCodeHashes)
		accountCards = append(accountCards, qq.recoveryCodesLeftCard(ctx, recoveryCodesCount))

		for _, credentialx := range passkeyCredentials {
			accountCards = append(accountCards, qq.passkeyCredentialCard(ctx, credentialx))
		}

		if len(passkeyCredentials) == 1 {
			accountCards = append(accountCards, &widget.Card{
				Style:          widget.CardStyleFilled,
				Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("Add a backup passkey")),
				Subhead:        widget.T("Passkey recommendation"),
				SupportingText: widget.T("Set up a second passkey on another device as backup in case one device is lost."),
				HTMXAttrs:      qq.registerPasskeyHTMXAttrs(),
			})
		}

		accountCardsBtns = append(accountCardsBtns, qq.registerPasskeyBtn(ctx, widget.ButtonStyleTypeElevated))
	}

	if len(passkeyCredentials) > 0 {
		accountCardsBtns = append(
			accountCardsBtns,
			&widget.Button{
				Label:     widget.T("Regenerate backup codes"),
				StyleType: widget.ButtonStyleTypeElevated,
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.actions.AuthActions.RegeneratePasskeyCodesCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.AuthActions.RegeneratePasskeyCodesCmd.Data()),
					HxConfirm: widget.T("Regenerate backup codes? Existing codes will stop working.").String(ctx),
					HxSwap:    "none",
				},
			},
		)
	}

	accountHeading := widget.Tf("Account «%s»", ctx.MainCtx().Account.Email.String())
	if owningTenantName, ok := qq.owningTenantName(ctx); ok {
		accountHeading = widget.Tf(
			"Account «%s», owned by «%s»",
			ctx.MainCtx().Account.Email.String(),
			owningTenantName,
		)
	}

	grids = append(grids, &widget.Grid{
		// TODO show Name and email?
		Heading:  widget.H(widget.HeadingTypeTitleMd, accountHeading),
		Children: accountCards,
		Footer: &widget.Row{
			Wrap:     true,
			Children: accountCardsBtns,
		},
	})

	if len(systemCards) > 0 {
		grids = append(grids, &widget.Grid{
			Heading:  widget.H(widget.HeadingTypeTitleMd, widget.T("System")), // TODO admin, system or app?
			Children: systemCards,
			Footer: &widget.Row{
				Wrap:     true,
				Children: systemFooterBtns,
			},
		})
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.id(),
		},
		GapY: true,
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
				events.InitialPasswordSet,
				events.TemporaryPasswordCleared,
				events.PasswordChanged, // necessary for "Active temporary password" card
				events.AppInitialized,
				events.AppUnlocked,
				events.AppPassphraseChanged,
				events.UploadLimitUpdated,
				events.AccountUpdated, // refresh from when opening again and update language
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data()),
			HxTarget: "#" + qq.id(),
			HxSelect: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Child: grids,
		// HTMXAttrs: htmxAttrs,
	}, nil

	/*return &wx.ScrollableContent{
		PaddingX: true,
		Children: grid,
	}*/
}

func (qq *DashboardCardsPartial) setupRequiredWidget(ctx ctxx.Context) *widget.Container {
	grids := []*widget.Grid{{
		Heading: widget.H(widget.HeadingTypeTitleMd, widget.Tf("Account «%s»", ctx.MainCtx().Account.Email.String())),
		Children: []*widget.Card{
			{
				Style:          widget.CardStyleFilled,
				Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("Passkey setup required")),
				Subhead:        widget.T("Passkeys"),
				SupportingText: widget.T("Your organization requires passkey sign-in. Register a passkey to continue."),
				Actions: []*widget.Button{
					qq.registerPasskeyBtn(ctx, widget.ButtonStyleTypeTonal),
				},
			},
		},
	}}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.id(),
		},
		GapY: true,
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
				events.AccountUpdated,
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data()),
			HxTarget: "#" + qq.id(),
			HxSelect: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Child: grids,
	}
}

func (qq *DashboardCardsPartial) recoveryCodesLeftCard(ctx ctxx.Context, recoveryCodesCount int) *widget.Card {
	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: widget.H(widget.HeadingTypeTitleLg, widget.Tf("%d backup codes left", recoveryCodesCount)),
		Subhead:  widget.T("Passkeys"),
	}
}

func (qq *DashboardCardsPartial) registerPasskeyBtn(ctx ctxx.Context, styleType widget.ButtonStyleType) *widget.Button {
	return &widget.Button{
		Label:     widget.T("Register passkey"),
		StyleType: styleType,
		HTMXAttrs: qq.registerPasskeyHTMXAttrs(),
	}
}

func (qq *DashboardCardsPartial) registerPasskeyHTMXAttrs() widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost: qq.actions.AuthActions.PasskeyRegisterDialog.EndpointWithParams(
			actionx2.ResponseWrapperDialog,
			"",
		),
		HxVals:        util.JSON(qq.actions.AuthActions.PasskeyRegisterDialog.Data()),
		LoadInPopover: true,
	}
}

func (qq *DashboardCardsPartial) nilableTenantCard(ctx ctxx.Context, tenantx *entmain.Tenant) *widget.Card {
	var actions []*widget.Button

	tenantm := tenant.NewTenant(tenantx)

	var headline *widget.Heading
	var subhead *widget.Text
	var supportingText *widget.Text

	if tenantm.IsInitialized() {
		if !qq.infra.SystemConfig().IsSaaSModeEnabled() {
			return nil
		}

		// TODO add role info
		headline = widget.H(widget.HeadingTypeTitleLg, widget.Tu(tenantx.Plan.String()))
		subhead = widget.T("Subscription")

		accountm := account2.NewAccount(ctx.MainCtx().Account)
		if tenantm.IsOwner(accountm) {
			// TODO
			/*actions = append(actions, &wx.Button{
				Label:     wx.T("Change plan"),
				StyleType: wx.ButtonStyleTypeOutlined,
			})*/
		}

		/*actions = append(actions, &wx.Button{
			Label:     wx.T("Spaces"),
			StyleType: wx.ButtonStyleTypeTonal,
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.SpacesRoot(tenantx.PublicID.String()),
			},
		})*/
	} else {
		headline = widget.H(widget.HeadingTypeTitleLg, widget.T("Not initialized"))
		subhead = widget.T("Please wait")
		supportingText = widget.T("The organization is not initialized yet, please wait until the initialization is complete.")

		actions = append(actions, &widget.Button{
			Label:     widget.T("Refresh"),
			StyleType: widget.ButtonStyleTypeOutlined,
			HTMXAttrs: widget.HTMXAttrs{
				// TODO show snackbar that user knows something has happened
				// 		maybe just add timestamp to description?
				HxGet: route.Dashboard(),
			},
		})
	}

	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: headline,
		// Headline: wx.H(wx.HeadingTypeTitleLg, wx.Tu(tenantx.Name)),
		Subhead:        subhead,
		SupportingText: supportingText,
		Actions:        actions,
	}
}

func (qq *DashboardCardsPartial) nilableQuotaUsageCard(ctx ctxx.Context, tenantx *entmain.Tenant) *widget.Card {
	tenantm := tenant.NewTenant(tenantx)
	if !tenantm.IsInitialized() {
		return nil
	}

	if !qq.infra.SystemConfig().IsSaaSModeEnabled() {
		return nil
	}

	quotaUsageLabel := qq.tenantStorageUsageLabel(ctx, tenantx)

	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: widget.H(widget.HeadingTypeTitleLg, widget.Tu(quotaUsageLabel)),
		Subhead:  widget.T("Quota usage"),
	}
}

func (qq *DashboardCardsPartial) tenantStorageUsageLabel(ctx ctxx.Context, tenantx *entmain.Tenant) string {
	tenantDB, ok := ctx.MainCtx().UnsafeTenantDB(tenantx.ID)
	if !ok {
		log.Println("tenant db not found, tenant id was", tenantx.ID)
		return widget.T("Unavailable").String(ctx)
	}

	tenantTx, err := tenantDB.ReadOnlyConn.Tx(ctx)
	if err != nil {
		log.Println("failed to start transaction for tenant", tenantx.ID, err)
		return widget.T("Unavailable").String(ctx)
	}

	tenantCtx := ctxx2.NewTenantContext(ctx.MainCtx(), tenantTx, tenantx, true)
	usedBytes, limitBytes, err := qq.infra.FileSystem().TenantUsageBytes(tenantCtx)
	if err != nil {
		log.Println("failed to query storage usage for tenant", tenantx.ID, err)
		if rollbackErr := tenantTx.Rollback(); rollbackErr != nil {
			log.Println("failed to rollback transaction for tenant", tenantx.ID, rollbackErr)
		}
		return widget.T("Unavailable").String(ctx)
	}

	if err := tenantTx.Commit(); err != nil {
		log.Println("failed to commit transaction for tenant", tenantx.ID, err)
		if rollbackErr := tenantTx.Rollback(); rollbackErr != nil {
			log.Println("failed to rollback transaction for tenant", tenantx.ID, rollbackErr)
		}
		return widget.T("Unavailable").String(ctx)
	}

	return fmt.Sprintf("%s of %s", fileutil.FormatSize(usedBytes), fileutil.FormatSize(limitBytes))
}

func (qq *DashboardCardsPartial) spaceCard(ctx ctxx.Context, spacex *enttenant.Space, tenant *entmain.Tenant) *widget.Card {
	var contextMenu *widget.Menu
	// if ctx.TenantCtx().User.Role == tenantrole.Owner {
	contextMenu = NewSpaceContextMenuWidget(qq.actions).Widget(ctx, tenant.PublicID.String(), spacex.PublicID.String())
	// }

	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: widget.H(widget.HeadingTypeTitleLg, widget.Tu(spacex.Name)),
		Subhead:  widget.T("Space"),
		// SupportingText: wx.Tf("Organization: %s", tenant.Name),
		ContextMenu: contextMenu,
		// SupportingText: wx.Tu(spacex.Description), // TODO tenant
		Actions: []*widget.Button{
			{
				Label:     widget.T("Select"), // TODO Browse, Open, Switch or activate? or Select?
				StyleType: widget.ButtonStyleTypeTonal,
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.BrowseRoot(tenant.PublicID.String(), spacex.PublicID.String()),
				},
			},
		},
	}
}

/*func (qq *DashboardCardsPartial) changePasswordCard(ctx ctxx.Context) *wx.Card {
	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, wx.T("Change password")),
		// Subhead:  wx.T("Account"), // TODO or Account settings / Security / Account security?
		Actions: qq.changePasswordBtn(),
	}
}*/

func (qq *DashboardCardsPartial) changePasswordBtn(ctx ctxx.Context) *widget.Button {
	return &widget.Button{
		Label:     widget.T("Change password"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AuthActions.ChangePasswordCmd.ModalLinkAttrs(
			qq.actions.AuthActions.ChangePasswordCmd.Data("", "", ""),
			"",
		),
	}
}

func (qq *DashboardCardsPartial) owningTenantName(ctx ctxx.Context) (string, bool) {
	now := time.Now()

	owningAssignment, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.AccountID(ctx.MainCtx().Account.ID),
			tenantaccountassignment.IsOwningTenant(true),
			tenantaccountassignment.Or(
				tenantaccountassignment.ExpiresAtIsNil(),
				tenantaccountassignment.ExpiresAtGT(now),
			),
			tenantaccountassignment.HasAccountWith(mainaccount.DeletedAtIsNil()),
			tenantaccountassignment.HasTenantWith(maintenant.DeletedAtIsNil()),
		).
		Only(ctx)
	if err != nil {
		log.Println("failed to query owning tenant assignment", ctx.MainCtx().Account.ID, err)
		return "", false
	}

	owningTenantx, err := ctx.MainCtx().MainTx.Tenant.Query().
		Where(
			maintenant.ID(owningAssignment.TenantID),
			maintenant.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		log.Println("failed to query owning tenant", owningAssignment.TenantID, err)
		return "", false
	}

	return owningTenantx.Name, true
}

/*func (qq *DashboardCardsPartial) deleteAccountCard(ctx ctxx.Context) *wx.Card {
	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, wx.T("Delete account")),
		// Subhead:  wx.T("Account"),
		Actions: qq.deleteAccountBtn(ctx),
	}
}*/

func (qq *DashboardCardsPartial) id() string {
	return "dashboardCards"
}

func (qq *DashboardCardsPartial) setPasswordCard(ctx ctxx.Context) *widget.Card {
	// TODO highlight important cards
	return &widget.Card{
		Style:          widget.CardStyleFilled,
		Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("No password set")),
		Subhead:        widget.T("Account"),
		SupportingText: widget.T("You've logged in with a temporary password. Please set a password to secure your account and use the app."),
		Actions: []*widget.Button{
			{
				Label:     widget.T("Set password now"),
				StyleType: widget.ButtonStyleTypeTonal,
				HTMXAttrs: qq.actions.AuthActions.SetInitialPasswordCmd.ModalLinkAttrs(
					qq.actions.AuthActions.SetInitialPasswordCmd.Data("", ""),
					"",
				),
			},
		},
	}
}

/*
func (qq *DashboardCardsPartial) editAccountCard(ctx ctxx.Context) *wx.Card {
	// TODO highlight important cards
	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, wx.T("Edit account")),
		// Subhead:  wx.T("Account"),
		Actions: qq.editAccountBtn(ctx),
	}
}
*/

func (qq *DashboardCardsPartial) editAccountBtn(ctx ctxx.Context) (*widget.Button, bool) {
	return &widget.Button{
		Label:     widget.T("Edit account"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AuthActions.EditAccountCmd.ModalLinkAttrs(
			qq.actions.AuthActions.EditAccountCmd.Data(
				ctx.MainCtx().Account.PublicID.String(),
				ctx.MainCtx().Account.Language,
				ctx.MainCtx().Account.SubscribedToNewsletterAt != nil,
			),
			"",
		),
	}, true
}

func (qq *DashboardCardsPartial) clearTemporaryPasswordCard(ctx ctxx.Context) *widget.Card {
	return &widget.Card{
		Style:          widget.CardStyleFilled,
		Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("Active temporary password")),
		Subhead:        widget.T("Account"),
		SupportingText: widget.T("Your account has an active temporary password. Please change your password or clear the temporary password as soon as possible to secure your account."),
		Actions: []*widget.Button{
			{
				Label:     widget.T("Change password"),
				StyleType: widget.ButtonStyleTypeTonal,
				HTMXAttrs: qq.actions.AuthActions.ChangePasswordCmd.ModalLinkAttrs(
					qq.actions.AuthActions.ChangePasswordCmd.Data("", "", ""),
					"",
				),
			},
			{
				Label:     widget.T("Clear temporary password"),
				StyleType: widget.ButtonStyleTypeOutlined,
				HTMXAttrs: widget.HTMXAttrs{
					HxPost: qq.actions.AuthActions.ClearTemporaryPasswordCmd.Endpoint(),
					HxVals: util.JSON(qq.actions.AuthActions.ClearTemporaryPasswordCmd.Data()),
				},
			},
		},
	}
}

func (qq *DashboardCardsPartial) manageSpacesCard(ctx ctxx.Context, tenantx *entmain.Tenant, spacesCount int) (*widget.Card, bool) {
	if spacesCount > 0 {
		return nil, false
	}

	tenantm := tenant.NewTenant(tenantx)
	if !tenantm.IsInitialized() {
		return nil, false
	}

	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return &widget.Card{
			Style:          widget.CardStyleFilled,
			Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("No space available yet")),
			Subhead:        widget.T("Space"),
			SupportingText: widget.Tf("You have no permission to access any space of this organization."),
		}, true
	}

	return &widget.Card{
		Style:          widget.CardStyleFilled,
		Headline:       widget.H(widget.HeadingTypeTitleLg, widget.T("No space available yet")),
		Subhead:        widget.T("Space"),
		SupportingText: widget.Tf("Please create one to get started."),
		Actions: []*widget.Button{{
			Label:     widget.T("Manage spaces"),
			StyleType: widget.ButtonStyleTypeTonal,
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route2.SpacesRoot(tenantx.PublicID.String()),
			},
		}},
	}, true
}

func (qq *DashboardCardsPartial) passkeyEnforcementBtn(ctx ctxx.Context, tenantx *entmain.Tenant) (*widget.Button, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	buttonLabel := widget.T("Enable passkey enforcement")
	confirmText := widget.T("Enable passkey enforcement for this organization? Members will need passkeys to sign in.")
	if tenantx.PasskeyAuthEnforced {
		buttonLabel = widget.T("Disable passkey enforcement")
		confirmText = widget.T("Disable passkey enforcement for this organization? Members can use passwords again if allowed.")
	}

	return &widget.Button{
		Label:     buttonLabel,
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.ToggleTenantPasskeyEnforcementCmd.Endpoint(),
			HxVals: util.JSON(
				qq.actions.ToggleTenantPasskeyEnforcementCmd.Data(
					tenantx.PublicID.String(),
					!tenantx.PasskeyAuthEnforced,
				),
			),
			HxConfirm: confirmText.String(ctx),
			HxSwap:    "none",
		},
	}, true
}

func (qq *DashboardCardsPartial) manageSpacesBtn(ctx ctxx.Context, tenantx *entmain.Tenant) (*widget.Button, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}
	return &widget.Button{
		Label:     widget.T("Manage spaces"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
			HxGet: route2.SpacesRoot(tenantx.PublicID.String()),
		},
	}, true
}

func (qq *DashboardCardsPartial) manageUsersCard(ctx ctxx.Context, tenantDB *sqlx.TenantDB, tenantx *entmain.Tenant) (*widget.Card, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	usersCount, err := tenantDB.ReadOnlyConn.User.Query().Count(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		log.Println("failed to query users of tenant", tenantx.ID, err)
		return nil, false
	}

	// headline := wx.H(wx.HeadingTypeTitleLg, wx.T("Manage users"))
	supportingText := widget.Tf("%d users", usersCount)
	if usersCount == 1 {
		supportingText = widget.Tf("1 user")
	}

	btn, ok := qq.manageUsersBtn(ctx, tenantx)
	if !ok {
		return nil, false
	}

	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: widget.H(widget.HeadingTypeTitleLg, supportingText),
		// Subhead:        wx.T("Organization"),
		// Subhead: supportingText,
		Actions: []*widget.Button{btn},
	}, true
}

func (qq *DashboardCardsPartial) manageUsersBtn(ctx ctxx.Context, tenantx *entmain.Tenant) (*widget.Button, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	return &widget.Button{
		Label:     widget.T("Manage users"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
			HxGet: route.ManageUsersOfTenant(tenantx.PublicID.String()),
		},
	}, true
}

func (qq *DashboardCardsPartial) deleteTenantBtn(ctx ctxx.Context, tenantx *entmain.Tenant) (*widget.Button, bool) {
	if !qq.infra.SystemConfig().IsSaaSModeEnabled() {
		return nil, false
	}

	deleteTenantCmdEndpoint := qq.infra.ManageTenantsDeleteTenantCmdEndpoint()
	if deleteTenantCmdEndpoint == "" {
		return nil, false
	}

	tenantm := tenant.NewTenant(tenantx)
	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	return &widget.Button{
		Label:     widget.T("Delete organization"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    deleteTenantCmdEndpoint,
			HxVals:    util.JSON(map[string]any{"TenantID": tenantx.PublicID.String()}),
			HxConfirm: widget.T("Are you sure? This organization will be deleted. All accounts owned by this organization will be deleted globally.").String(ctx),
		},
	}, true
}

func (qq *DashboardCardsPartial) downloadTenantBackupLink(
	ctx ctxx.Context,
	tenantx *entmain.Tenant,
) (*widget.Link, bool) {
	if !qq.infra.SystemConfig().IsSaaSModeEnabled() {
		return nil, false
	}

	endpoint := qq.infra.ManageTenantsDownloadBackupEndpoint()
	if endpoint == "" {
		return nil, false
	}

	tenantm := tenant.NewTenant(tenantx)
	accountm := account2.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	filename := "tenant-backup-" + tenantx.PublicID.String() + ".zip"

	return &widget.Link{
		Href:     endpoint + "?tenant_id=" + url.QueryEscape(tenantx.PublicID.String()),
		Filename: filename,
		Child: &widget.Button{
			Label:     widget.T("Download backup"),
			StyleType: widget.ButtonStyleTypeElevated,
		},
	}, true
}

func (qq *DashboardCardsPartial) appStatusCard(ctx ctxx.Context) *widget.Card {
	var actions []*widget.Button
	supportingText := widget.T("The app is unlocked and not protected by a passphrase.")

	if qq.infra.SystemConfig().IsAppLocked() {
		actions = append(actions, &widget.Button{
			Label:     widget.T("Unlock app"),
			StyleType: widget.ButtonStyleTypeTonal,
			HTMXAttrs: qq.actions.AdminActions.UnlockAppCmd.ModalLinkAttrs(
				qq.actions.AdminActions.UnlockAppCmd.Data(), ""),
		})
		supportingText = widget.T("The app is locked.")
	} else {
		label := widget.T("Set passphrase")
		styleType := widget.ButtonStyleTypeTonal

		if qq.infra.SystemConfig().IsIdentityEncryptedWithPassphrase() {
			supportingText = widget.T("The app is unlocked and protected by a passphrase.")
			label = widget.T("Change passphrase")
			styleType = widget.ButtonStyleTypeOutlined
		}

		actions = append(actions, &widget.Button{
			Label:     label,
			StyleType: styleType,
			HTMXAttrs: qq.actions.AdminActions.ChangePassphraseCmd.ModalLinkAttrs(
				qq.actions.AdminActions.ChangePassphraseCmd.Data(), ""),
		})

		if qq.infra.SystemConfig().IsIdentityEncryptedWithPassphrase() {
			actions = append(actions, &widget.Button{
				Label:     widget.T("Remove passphrase"),
				StyleType: widget.ButtonStyleTypeOutlined,
				HTMXAttrs: qq.actions.AdminActions.RemovePassphraseCmd.ModalLinkAttrs(
					qq.actions.AdminActions.RemovePassphraseCmd.Data(), ""),
			})
		}
	}

	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: widget.H(widget.HeadingTypeTitleLg, widget.T("App status")),
		// Subhead:        wx.T("Admin"),
		SupportingText: supportingText,
		Actions:        actions,
	}

}

func (qq *DashboardCardsPartial) passkeyCredentialCard(
	ctx ctxx.Context,
	credentialx *entmain.PasskeyCredential,
) *widget.Card {
	supportingText := widget.Tf("Created on %s", credentialx.CreatedAt.Format("2006-01-02 15:04"))
	if credentialx.LastUsedAt != nil {
		supportingText = widget.Tf("Last used on %s", credentialx.LastUsedAt.Format("2006-01-02 15:04"))
	}

	credentialName := strings.TrimSpace(credentialx.Name)
	if credentialName == "" {
		credentialName = widget.T("Passkey").String(ctx)
	}

	return &widget.Card{
		Style:          widget.CardStyleFilled,
		Headline:       widget.H(widget.HeadingTypeTitleLg, widget.Tu(credentialName)),
		Subhead:        widget.T("Passkey"),
		SupportingText: supportingText,
		ContextMenu:    NewPasskeyContextMenuWidget(qq.actions).Widget(ctx, credentialx.PublicID.String(), credentialName),
	}
}

func (qq *DashboardCardsPartial) manageUploadLimitBtn(ctx ctxx.Context) *widget.Button {
	return &widget.Button{
		Label:     widget.T("Manage upload limit"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AdminActions.SetGlobalUploadLimitForm.ModalLinkAttrs(
			qq.actions.AdminActions.SetGlobalUploadLimitForm.Data(),
			"",
		),
	}
}
