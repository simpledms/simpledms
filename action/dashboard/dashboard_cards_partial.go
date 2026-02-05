package dashboard

import (
	"log"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/tenant"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DashboardCardsPartialData struct {
}

type DashboardCardsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDashboardCardsPartial(infra *common.Infra, actions *Actions) *DashboardCardsPartial {
	config := actionx.NewConfig(
		actions.Route("dashboard-cards-partial"),
		true,
	)
	return &DashboardCardsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DashboardCardsPartial) Data() *DashboardCardsPartialData {
	return &DashboardCardsPartialData{}
}

func (qq *DashboardCardsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx),
	)
}

// guidelines:
// most important card should always be in front, for example `set password` if not set yet
// or `manage spaces` if no space exists yet
func (qq *DashboardCardsPartial) Widget(
	ctx ctxx.Context,
) renderable.Renderable {
	var grids []*wx.Grid

	var openTaskCards []*wx.Card
	var accountCards []*wx.Card
	var systemCards []*wx.Card

	accountm := account.NewAccount(ctx.MainCtx().Account)
	if accountm.Data.Role == mainrole.Admin {
		systemCards = append(systemCards, qq.appStatusCard(ctx))
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
		grids = append(grids, &wx.Grid{
			Heading:  wx.H(wx.HeadingTypeTitleMd, wx.T("Open tasks")),
			Children: openTaskCards,
		})
	}

	spacesByTenant := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
	for tenantx, spaces := range spacesByTenant {
		var tenantCards []*wx.Card
		var tenantHeaderBtns []*wx.Button

		if tenantCard := qq.nilableTenantCard(ctx, tenantx); tenantCard != nil {
			tenantCards = append(tenantCards, tenantCard)
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
		for _, spacex := range spaces {
			tenantCards = append(tenantCards, qq.spaceCard(ctx, spacex, tenantx))
		}

		grids = append(grids, &wx.Grid{
			Heading: wx.Hf(wx.HeadingTypeTitleMd, "Organization «%s»", tenantx.Name),
			Footer: &wx.Row{
				Children: tenantHeaderBtns,
			},
			Children: tenantCards,
		})
	}

	var accountCardsBtns []*wx.Button

	if accountm.HasPassword() && !accountm.HasTemporaryPassword() {
		accountCardsBtns = append(accountCardsBtns, qq.changePasswordBtn(ctx))
	}

	accountCardsBtns = append(
		accountCardsBtns,
		qq.editAccountBtn(ctx),
		qq.deleteAccountBtn(ctx),
	)

	grids = append(grids, &wx.Grid{
		// TODO show Name and email?
		Heading:  wx.H(wx.HeadingTypeTitleMd, wx.Tf("Account «%s»", ctx.MainCtx().Account.Email.String())),
		Children: accountCards,
		Footer:   &wx.Row{Children: accountCardsBtns},
	})

	if len(systemCards) > 0 {
		grids = append(grids, &wx.Grid{
			Heading:  wx.H(wx.HeadingTypeTitleMd, wx.T("System")), // TODO admin, system or app?
			Children: systemCards,
		})
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.id(),
		},
		GapY: true,
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.InitialPasswordSet,
				event.TemporaryPasswordCleared,
				event.PasswordChanged, // necessary for "Active temporary password" card
				event.AppInitialized,
				event.AppUnlocked,
				event.AppPassphraseChanged,
				event.AccountUpdated, // refresh from when opening again and update language
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data()),
			HxTarget: "#" + qq.id(),
			HxSelect: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Child: grids,
		// HTMXAttrs: htmxAttrs,
	}

	/*return &wx.ScrollableContent{
		PaddingX: true,
		Children: grid,
	}*/
}

func (qq *DashboardCardsPartial) nilableTenantCard(ctx ctxx.Context, tenantx *entmain.Tenant) *wx.Card {
	var actions []*wx.Button

	tenantm := tenant.NewTenant(tenantx)

	var headline *wx.Heading
	var subhead *wx.Text
	var supportingText *wx.Text

	if tenantm.IsInitialized() {
		if !qq.infra.SystemConfig().IsSaaSModeEnabled() {
			return nil
		}

		// TODO add role info
		headline = wx.H(wx.HeadingTypeTitleLg, wx.T("Trial phase"))
		subhead = wx.T("Subscription")

		accountm := account.NewAccount(ctx.MainCtx().Account)
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
		headline = wx.H(wx.HeadingTypeTitleLg, wx.T("Not initialized"))
		subhead = wx.T("Please wait")
		supportingText = wx.T("The organization is not initialized yet, please wait until the initialization is complete.")

		actions = append(actions, &wx.Button{
			Label:     wx.T("Refresh"),
			StyleType: wx.ButtonStyleTypeOutlined,
			HTMXAttrs: wx.HTMXAttrs{
				// TODO show snackbar that user knows something has happened
				// 		maybe just add timestamp to description?
				HxGet: route2.Dashboard(),
			},
		})
	}

	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: headline,
		// Headline: wx.H(wx.HeadingTypeTitleLg, wx.Tu(tenantx.Name)),
		Subhead:        subhead,
		SupportingText: supportingText,
		Actions:        actions,
	}
}

func (qq *DashboardCardsPartial) spaceCard(ctx ctxx.Context, spacex *enttenant.Space, tenant *entmain.Tenant) *wx.Card {
	var contextMenu *wx.Menu
	// if ctx.TenantCtx().User.Role == tenantrole.Owner {
	contextMenu = NewSpaceContextMenuWidget(qq.actions).Widget(ctx, tenant.PublicID.String(), spacex.PublicID.String())
	// }

	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, wx.Tu(spacex.Name)),
		Subhead:  wx.T("Space"),
		// SupportingText: wx.Tf("Organization: %s", tenant.Name),
		ContextMenu: contextMenu,
		// SupportingText: wx.Tu(spacex.Description), // TODO tenant
		Actions: []*wx.Button{
			{
				Label:     wx.T("Select"), // TODO Browse, Open, Switch or activate? or Select?
				StyleType: wx.ButtonStyleTypeTonal,
				HTMXAttrs: wx.HTMXAttrs{
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

func (qq *DashboardCardsPartial) changePasswordBtn(ctx ctxx.Context) *wx.Button {
	return &wx.Button{
		Label:     wx.T("Change password"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AuthActions.ChangePasswordCmd.ModalLinkAttrs(
			qq.actions.AuthActions.ChangePasswordCmd.Data("", "", ""),
			"",
		),
	}
}

/*func (qq *DashboardCardsPartial) deleteAccountCard(ctx ctxx.Context) *wx.Card {
	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, wx.T("Delete account")),
		// Subhead:  wx.T("Account"),
		Actions: qq.deleteAccountBtn(ctx),
	}
}*/

func (qq *DashboardCardsPartial) deleteAccountBtn(ctx ctxx.Context) *wx.Button {
	return &wx.Button{
		Label:     wx.T("Delete account"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.AuthActions.DeleteAccountCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.AuthActions.DeleteAccountCmd.Data(ctx.MainCtx().Account.PublicID.String())),
			HxConfirm: wx.T("Are you sure?").String(ctx),
		},
	}
}

func (qq *DashboardCardsPartial) id() string {
	return "dashboardCards"
}

func (qq *DashboardCardsPartial) setPasswordCard(ctx ctxx.Context) *wx.Card {
	// TODO highlight important cards
	return &wx.Card{
		Style:          wx.CardStyleFilled,
		Headline:       wx.H(wx.HeadingTypeTitleLg, wx.T("No password set")),
		Subhead:        wx.T("Account"),
		SupportingText: wx.T("You've logged in with a temporary password. Please set a password to secure your account and use the app."),
		Actions: []*wx.Button{
			{
				Label:     wx.T("Set password now"),
				StyleType: wx.ButtonStyleTypeTonal,
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

func (qq *DashboardCardsPartial) editAccountBtn(ctx ctxx.Context) *wx.Button {
	return &wx.Button{
		Label:     wx.T("Edit account"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AuthActions.EditAccountCmd.ModalLinkAttrs(
			qq.actions.AuthActions.EditAccountCmd.Data(
				ctx.MainCtx().Account.PublicID.String(),
				ctx.MainCtx().Account.Language,
				ctx.MainCtx().Account.SubscribedToNewsletterAt != nil,
			),
			"",
		),
	}
}

func (qq *DashboardCardsPartial) clearTemporaryPasswordCard(ctx ctxx.Context) *wx.Card {
	return &wx.Card{
		Style:          wx.CardStyleFilled,
		Headline:       wx.H(wx.HeadingTypeTitleLg, wx.T("Active temporary password")),
		Subhead:        wx.T("Account"),
		SupportingText: wx.T("Your account has an active temporary password. Please change your password or clear the temporary password as soon as possible to secure your account."),
		Actions: []*wx.Button{
			{
				Label:     wx.T("Change password"),
				StyleType: wx.ButtonStyleTypeTonal,
				HTMXAttrs: qq.actions.AuthActions.ChangePasswordCmd.ModalLinkAttrs(
					qq.actions.AuthActions.ChangePasswordCmd.Data("", "", ""),
					"",
				),
			},
			{
				Label:     wx.T("Clear temporary password"),
				StyleType: wx.ButtonStyleTypeOutlined,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.actions.AuthActions.ClearTemporaryPasswordCmd.Endpoint(),
					HxVals: util.JSON(qq.actions.AuthActions.ClearTemporaryPasswordCmd.Data()),
				},
			},
		},
	}
}

func (qq *DashboardCardsPartial) manageSpacesCard(ctx ctxx.Context, tenantx *entmain.Tenant, spacesCount int) (*wx.Card, bool) {
	if spacesCount > 0 {
		return nil, false
	}

	tenantm := tenant.NewTenant(tenantx)
	if !tenantm.IsInitialized() {
		return nil, false
	}

	accountm := account.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return &wx.Card{
			Style:          wx.CardStyleFilled,
			Headline:       wx.H(wx.HeadingTypeTitleLg, wx.T("No space available yet")),
			Subhead:        wx.T("Space"),
			SupportingText: wx.Tf("You have no permission to access any space of this organization."),
		}, true
	}

	return &wx.Card{
		Style:          wx.CardStyleFilled,
		Headline:       wx.H(wx.HeadingTypeTitleLg, wx.T("No space available yet")),
		Subhead:        wx.T("Space"),
		SupportingText: wx.Tf("Please create one to get started."),
		Actions: []*wx.Button{{
			Label:     wx.T("Manage spaces"),
			StyleType: wx.ButtonStyleTypeTonal,
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route2.SpacesRoot(tenantx.PublicID.String()),
			},
		}},
	}, true
}

func (qq *DashboardCardsPartial) manageSpacesBtn(ctx ctxx.Context, tenantx *entmain.Tenant) (*wx.Button, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}
	return &wx.Button{
		Label:     wx.T("Manage spaces"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: wx.HTMXAttrs{
			HxGet: route2.SpacesRoot(tenantx.PublicID.String()),
		},
	}, true
}

func (qq *DashboardCardsPartial) manageUsersCard(ctx ctxx.Context, tenantDB *sqlx.TenantDB, tenantx *entmain.Tenant) (*wx.Card, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account.NewAccount(ctx.MainCtx().Account)
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
	supportingText := wx.Tf("%d users", usersCount)
	if usersCount == 1 {
		supportingText = wx.Tf("1 user")
	}

	btn, ok := qq.manageUsersBtn(ctx, tenantx)
	if !ok {
		return nil, false
	}

	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, supportingText),
		// Subhead:        wx.T("Organization"),
		// Subhead: supportingText,
		Actions: []*wx.Button{btn},
	}, true
}

func (qq *DashboardCardsPartial) manageUsersBtn(ctx ctxx.Context, tenantx *entmain.Tenant) (*wx.Button, bool) {
	tenantm := tenant.NewTenant(tenantx)
	accountm := account.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	return &wx.Button{
		Label:     wx.T("Manage users"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: wx.HTMXAttrs{
			HxGet: route2.ManageUsersOfTenant(tenantx.PublicID.String()),
		},
	}, true
}

func (qq *DashboardCardsPartial) appStatusCard(ctx ctxx.Context) *wx.Card {
	var actions []*wx.Button
	supportingText := wx.T("The app is unlocked and not protected by a passphrase.")

	if qq.infra.SystemConfig().IsAppLocked() {
		actions = append(actions, &wx.Button{
			Label:     wx.T("Unlock app"),
			StyleType: wx.ButtonStyleTypeTonal,
			HTMXAttrs: qq.actions.AdminActions.UnlockAppCmd.ModalLinkAttrs(
				qq.actions.AdminActions.UnlockAppCmd.Data(), ""),
		})
		supportingText = wx.T("The app is locked.")
	} else {
		label := wx.T("Set passphrase")
		styleType := wx.ButtonStyleTypeTonal

		if qq.infra.SystemConfig().IsIdentityEncryptedWithPassphrase() {
			supportingText = wx.T("The app is unlocked and protected by a passphrase.")
			label = wx.T("Change passphrase")
			styleType = wx.ButtonStyleTypeOutlined
		}

		actions = append(actions, &wx.Button{
			Label:     label,
			StyleType: styleType,
			HTMXAttrs: qq.actions.AdminActions.ChangePassphraseCmd.ModalLinkAttrs(
				qq.actions.AdminActions.ChangePassphraseCmd.Data(), ""),
		})

		if qq.infra.SystemConfig().IsIdentityEncryptedWithPassphrase() {
			actions = append(actions, &wx.Button{
				Label:     wx.T("Remove passphrase"),
				StyleType: wx.ButtonStyleTypeOutlined,
				HTMXAttrs: qq.actions.AdminActions.RemovePassphraseCmd.ModalLinkAttrs(
					qq.actions.AdminActions.RemovePassphraseCmd.Data(), ""),
			})
		}
	}

	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: wx.H(wx.HeadingTypeTitleLg, wx.T("App status")),
		// Subhead:        wx.T("Admin"),
		SupportingText: supportingText,
		Actions:        actions,
	}

}
