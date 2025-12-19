package managespaceusers

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
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
	var listItems []*wx.ListItem

	if ctx.SpaceCtx().UserRoleInSpace() == spacerole.Owner {
		listItems = append(listItems, &wx.ListItem{
			Headline: wx.T("Assign a user"), // TODO Create or add? system or real world perspective?
			Leading:  wx.NewIcon("add"),
			Type:     wx.ListItemTypeHelper,
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
		leading := wx.NewIcon("person")
		if assignment.Role == spacerole.Owner {
			// TODO add tooltip...
			leading = wx.NewIcon("manage_accounts")
		}
		userm := model.NewUser(assignment.Edges.User)
		listItems = append(listItems, &wx.ListItem{
			Leading:        leading,
			Headline:       wx.Tu(userm.Name()),
			SupportingText: wx.Tu(userm.NameSecondLine()),
			ContextMenu:    NewUserAssignmentContextMenu(qq.actions).Widget(ctx, assignment),
		})
	}

	return &wx.List{
		Widget: wx.Widget[wx.List]{
			ID: qq.id(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
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
