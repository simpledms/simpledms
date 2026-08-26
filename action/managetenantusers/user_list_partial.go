package managetenantusers

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	mainwebdavcredential "github.com/simpledms/simpledms/db/entmain/webdavcredential"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type UserListPartialState struct{}

type UserListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUserListPartial(infra *common.Infra, actions *Actions) *UserListPartial {
	return &UserListPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig("user-list-partial", true),
	}
}

func (qq *UserListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[UserListPartialState](rw, req)
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, state))
}

func (qq *UserListPartial) Widget(ctx ctxx.Context, state *UserListPartialState) *widget.List {
	var listItems []*widget.ListItem

	listItems = append(listItems, &widget.ListItem{
		Headline: widget.T("Add a new user"), // TODO Create or add? system or real world perspective?
		Leading:  widget.NewIcon("add"),
		Type:     widget.ListItemTypeHelper,
		HTMXAttrs: qq.actions.CreateUserCmd.ModalLinkAttrs(
			qq.actions.CreateUserCmd.Data(
				tenantrole.User,
				"",
				"",
				"",
				ctx.MainCtx().Account.Language, // TODO okay?
			), ""),
	})

	// TODO filtered by tenant?
	users := ctx.TenantCtx().TTx.User.Query().Order(user.ByLastName(), user.ByFirstName()).AllX(ctx)

	accountIDs := make([]int64, 0, len(users))
	for _, userx := range users {
		accountIDs = append(accountIDs, userx.AccountID)
	}

	isOwningTenantByAccountID := make(map[int64]bool, len(accountIDs))
	if len(accountIDs) > 0 {
		assignments, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
			Where(
				tenantaccountassignment.TenantID(ctx.TenantCtx().Tenant.ID),
				tenantaccountassignment.AccountIDIn(accountIDs...),
			).
			All(ctx)
		if err != nil {
			log.Println(err)
		} else {
			for _, assignment := range assignments {
				isOwningTenantByAccountID[assignment.AccountID] = assignment.IsOwningTenant
			}
		}
	}
	credentialsByAccountID := qq.webDAVCredentialsByAccountID(ctx, accountIDs)
	spacesByPublicID := qq.spacesByPublicID(ctx, credentialsByAccountID)

	for _, userx := range users {
		userm := usermodel.NewUser(userx)
		isOwningTenantAssignment := isOwningTenantByAccountID[userx.AccountID]

		leading := widget.NewIcon("person")
		if userx.Role == tenantrole.Owner {
			// TODO add tooltip...
			leading = widget.NewIcon("manage_accounts")
		}

		ownershipText := widget.T("Member account")
		if isOwningTenantAssignment {
			ownershipText = widget.T("Owned account")
		}

		listItems = append(listItems, &widget.ListItem{
			Leading:        leading,
			Headline:       widget.Tu(userm.Name()),
			SupportingText: widget.Tf("%s - %s", widget.Tu(userm.NameSecondLine()), ownershipText),
			ContextMenu: NewUserContextMenuWidget(qq.actions).Widget(
				ctx,
				userx,
				isOwningTenantAssignment,
			),
		})
		listItems = append(listItems, qq.webDAVCredentialItems(
			ctx,
			credentialsByAccountID[userx.AccountID],
			spacesByPublicID,
		)...)
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.id(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.UserCreated,
				event.UserUpdated,
				event.UserDeleted,
			),
			HxPost:   qq.Endpoint(),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Children: listItems,
	}
}

func (qq *UserListPartial) webDAVCredentialsByAccountID(
	ctx ctxx.Context,
	accountIDs []int64,
) map[int64][]*entmain.WebDAVCredential {
	result := map[int64][]*entmain.WebDAVCredential{}
	if len(accountIDs) == 0 {
		return result
	}
	credentials := ctx.MainCtx().MainTx.WebDAVCredential.Query().
		Where(
			mainwebdavcredential.TenantID(ctx.TenantCtx().Tenant.ID),
			mainwebdavcredential.AccountIDIn(accountIDs...),
		).
		Order(mainwebdavcredential.ByCreatedAt()).
		AllX(ctx)
	for _, credentialx := range credentials {
		result[credentialx.AccountID] = append(result[credentialx.AccountID], credentialx)
	}
	return result
}

func (qq *UserListPartial) spacesByPublicID(
	ctx ctxx.Context,
	credentialsByAccountID map[int64][]*entmain.WebDAVCredential,
) map[string]*enttenant.Space {
	spaceIDs := []entx.CIText{}
	for _, credentials := range credentialsByAccountID {
		for _, credentialx := range credentials {
			spaceIDs = append(spaceIDs, credentialx.SpacePublicID)
		}
	}
	if len(spaceIDs) == 0 {
		return map[string]*enttenant.Space{}
	}
	spaces := ctx.TenantCtx().TTx.Space.Query().Where(space.PublicIDIn(spaceIDs...)).AllX(ctx)
	result := map[string]*enttenant.Space{}
	for _, spacex := range spaces {
		result[spacex.PublicID.String()] = spacex
	}
	return result
}

func (qq *UserListPartial) webDAVCredentialItems(
	ctx ctxx.Context,
	credentials []*entmain.WebDAVCredential,
	spacesByPublicID map[string]*enttenant.Space,
) []*widget.ListItem {
	items := make([]*widget.ListItem, 0, len(credentials))
	for _, credentialx := range credentials {
		spaceLabel := widget.T("Unavailable destination").String(ctx)
		if spacex := spacesByPublicID[credentialx.SpacePublicID.String()]; spacex != nil {
			spaceLabel = spacex.Name
		}
		supportingText := widget.Tf(
			"Space: %s · Username: %s · Created: %s",
			spaceLabel,
			credentialx.Username,
			timex.NewDateTime(credentialx.CreatedAt).String(ctx.MainCtx().LanguageBCP47),
		)
		if credentialx.RevokedAt != nil {
			supportingText = widget.Tf(
				"Space: %s · Username: %s · Revoked: %s",
				spaceLabel,
				credentialx.Username,
				timex.NewDateTime(*credentialx.RevokedAt).String(ctx.MainCtx().LanguageBCP47),
			)
		}
		var menu *widget.Menu
		if credentialx.RevokedAt == nil && ctx.TenantCtx().User.Role == tenantrole.Owner {
			menu = &widget.Menu{Items: []*widget.MenuItem{{
				LeadingIcon: "block",
				Label:       widget.T("Revoke"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost: qq.actions.RevokeWebDAVCredentialCmd.Endpoint(),
					HxVals: util.JSON(qq.actions.RevokeWebDAVCredentialCmd.Data(
						credentialx.PublicID.String(),
					)),
					HxConfirm: widget.T("Revoke this WebDAV credential?").String(ctx),
				},
			}}}
		}
		items = append(items, &widget.ListItem{
			Leading:        widget.NewIcon("vpn_key"),
			Headline:       widget.Tu(credentialx.Label),
			SupportingText: supportingText,
			ContextMenu:    menu,
		})
	}
	return items
}

func (qq *UserListPartial) id() string {
	return "userListPartial"
}
