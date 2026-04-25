package managetenantusers

import (
	"log"

	"github.com/simpledms/simpledms/core/db/entmain/tenantaccountassignment"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/common/tenantrole"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/user"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
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

func (qq *UserListPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.id(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
				events.UserCreated,
				events.UserUpdated,
				events.UserDeleted,
			),
			HxPost:   qq.Endpoint(),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Children: listItems,
	}
}

func (qq *UserListPartial) id() string {
	return "userListPartial"
}
