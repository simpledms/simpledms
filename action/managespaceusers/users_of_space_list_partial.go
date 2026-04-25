package managespaceusers

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/tenant/common/spacerole"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type UsersOfSpaceListPartialState struct {
}

type UsersOfSpaceListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUsersOfSpaceListPartial(infra *common.Infra, actions *Actions) *UsersOfSpaceListPartial {
	return &UsersOfSpaceListPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig("users-of-space-list-partial", true),
	}
}

func (qq *UsersOfSpaceListPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	state := autil.StateX[UsersOfSpaceListPartialState](rw, req)
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, state))
}

func (qq *UsersOfSpaceListPartial) Widget(ctx ctxx.Context, state *UsersOfSpaceListPartialState) renderable.Renderable {
	var listItems []*widget.ListItem

	if ctx.SpaceCtx().UserRoleInSpace() == spacerole.Owner {
		listItems = append(listItems, &widget.ListItem{
			Headline: widget.T("Assign a user"), // TODO Create or add? system or real world perspective?
			Leading:  widget.NewIcon("add"),
			Type:     widget.ListItemTypeHelper,
			HTMXAttrs: qq.actions.AssignUserToSpaceCmd.ModalLinkAttrs(
				qq.actions.AssignUserToSpaceCmd.Data(), ""),
		})
	}

	spaceAssignments := ctx.SpaceCtx().TTx.SpaceUserAssignment.Query().
		WithUser().
		Order(
			spaceuserassignment.ByUserField(user.FieldLastName),
			spaceuserassignment.ByUserField(user.FieldFirstName),
			spaceuserassignment.ByUserField(user.FieldEmail),
		).
		AllX(ctx)

	for _, assignment := range spaceAssignments {
		leading := widget.NewIcon("person")
		if assignment.Role == spacerole.Owner {
			// TODO add tooltip...
			leading = widget.NewIcon("manage_accounts")
		}
		userm := usermodel.NewUser(assignment.Edges.User)
		listItems = append(listItems, &widget.ListItem{
			Leading:        leading,
			Headline:       widget.Tu(userm.Name()),
			SupportingText: widget.Tu(userm.NameSecondLine()),
			ContextMenu:    NewUserAssignmentContextMenuWidget(qq.actions).Widget(ctx, assignment),
		})
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.id(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
				event.UserAssignedToSpace,
				event.UserUnassignedFromSpace,
			),
			HxPost:   qq.Endpoint(),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Children: listItems,
	}
}

func (qq *UsersOfSpaceListPartial) id() string {
	return "usersOfSpaceListPartial"
}
