package managespaceusers

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *UsersOfSpaceListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[UsersOfSpaceListPartialState](rw, req)
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, state))
}

func (qq *UsersOfSpaceListPartial) Widget(ctx ctxx.Context, state *UsersOfSpaceListPartialState) renderable.Renderable {
	var listItems []*widget.ListItem

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

	htmxAttrs := widget.HTMXAttrs{
		HxTrigger: event.HxTrigger(
			event.UserAssignedToSpace,
			event.UserUnassignedFromSpace,
		),
		HxPost:   qq.Endpoint(),
		HxTarget: "#" + qq.id(),
		HxSwap:   "outerHTML",
	}

	if len(listItems) == 0 {
		var actions []widget.IWidget
		if ctx.SpaceCtx().UserRoleInSpace() == spacerole.Owner {
			actions = append(actions, &widget.Button{
				Icon:  widget.NewIcon("person_add"),
				Label: widget.T("Assign user"),
				HTMXAttrs: qq.actions.AssignUserToSpaceCmd.ModalLinkAttrs(
					qq.actions.AssignUserToSpaceCmd.Data(),
					"",
				),
			})
		}

		return &widget.Container{
			Widget: widget.Widget[widget.Container]{
				ID: qq.id(),
			},
			Child: &widget.EmptyState{
				Icon:     widget.NewIcon("person"),
				Headline: widget.T("No users assigned yet."),
				Actions:  actions,
			},
			HTMXAttrs: htmxAttrs,
		}
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.id(),
		},
		HTMXAttrs: htmxAttrs,
		Children:  listItems,
	}
}

func (qq *UsersOfSpaceListPartial) id() string {
	return "usersOfSpaceListPartial"
}
