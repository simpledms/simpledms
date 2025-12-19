package managespaceusers

import (
	"fmt"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type AssignUserToSpaceCmdData struct {
}

type AssignUserToSpaceCmdFormData struct {
	AssignUserToSpaceCmdData `structs:",flatten"`
	Role                     spacerole.SpaceRole `validate:"required"`
	UserID                   string              `validate:"required" structs:"-"`
	// SpaceID string              `validate:"required"` in ctx
}

type AssignUserToSpaceCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[AssignUserToSpaceCmdData]
}

func NewAssignUserToSpaceCmd(infra *common.Infra, actions *Actions) *AssignUserToSpaceCmd {
	config := actionx.NewConfig(actions.Route("assign-user-to-space-cmd"), false)
	return &AssignUserToSpaceCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[AssignUserToSpaceCmdData](infra, config, wx.T("Assign user to space")),
	}
}

func (qq *AssignUserToSpaceCmd) Data() *AssignUserToSpaceCmdData {
	return &AssignUserToSpaceCmdData{}
}

func (qq *AssignUserToSpaceCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignUserToSpaceCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	if !ctx.IsSpaceCtx() {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No space selected. Please select a space first.")
	}
	if ctx.SpaceCtx().UserRoleInSpace() != spacerole.Owner {
		return e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to assign users to spaces because you aren't the owner.")
	}

	userx := ctx.SpaceCtx().TTx.User.Query().Where(user.PublicID(entx.NewCIText(data.UserID))).OnlyX(ctx)
	if ctx.SpaceCtx().Space.QueryUserAssignment().Where(
		spaceuserassignment.UserID(userx.ID),
	).ExistX(ctx) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "User is already assigned to this space.")
	}

	ctx.SpaceCtx().TTx.SpaceUserAssignment.Create().
		SetSpace(ctx.SpaceCtx().Space).
		SetUserID(userx.ID).
		SetRole(data.Role).
		SaveX(ctx)

	// TODO send message to user via Chat?
	rw.AddRenderables(wx.NewSnackbarf("User assigned to space successfully."))
	rw.Header().Set("HX-Trigger", event.UserAssignedToSpace.String())

	return nil
}

func (qq *AssignUserToSpaceCmd) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormDataX[AssignUserToSpaceCmdFormData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	hxTarget := req.URL.Query().Get("hx-target")
	wrapper := req.URL.Query().Get("wrapper")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(
			ctx,
			data,
			actionx.ResponseWrapper(wrapper),
			hxTarget,
		),
	)
}

func (qq *AssignUserToSpaceCmd) Form(
	ctx ctxx.Context,
	data *AssignUserToSpaceCmdFormData,
	wrapper actionx.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	if data.Role == spacerole.SpaceRole(0) {
		data.Role = spacerole.User
	}

	form := &wx.Form{
		Widget: wx.Widget[wx.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					wx.NewFormFields(ctx, data),
					qq.userList(ctx),
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		wx.T("Assign user to space"),
		wx.T("Save"),
		form,
		wrapper,
		wx.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)
}

func (qq *AssignUserToSpaceCmd) popoverID() string {
	return "assignUserToSpacePopover"
}

func (qq *AssignUserToSpaceCmd) formID() string {
	return "assignUserToSpaceForm"
}

func (qq *AssignUserToSpaceCmd) userList(ctx ctxx.Context) wx.IWidget {
	listItems := qq.userListItems(ctx)

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.userListID(),
		},
		Children: &wx.List{
			Children: listItems,
		},
	}
}

func (qq *AssignUserToSpaceCmd) userListID() string {
	return "userList"
}

func (qq *AssignUserToSpaceCmd) userListItems(ctx ctxx.Context) interface{} {
	// TODO implement pagination?

	var items []*wx.ListItem

	unassignedUsers := ctx.TenantCtx().TTx.User.Query().WithSpaceAssignment().
		Where(user.Not(user.HasSpaceAssignmentWith(spaceuserassignment.SpaceID(ctx.SpaceCtx().Space.ID)))).
		AllX(ctx)

	if len(unassignedUsers) == 0 {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("No unassigned users available."),
			SupportingText: wx.T("Please create a user in the organization user management first."), // TODO link
		})
		return items
	}

	for _, unassignedUser := range unassignedUsers {
		userm := model.NewUser(unassignedUser)
		items = append(items, &wx.ListItem{
			RadioGroupName: "UserID",
			RadioValue:     fmt.Sprintf("%s", unassignedUser.PublicID),
			Headline:       wx.Tu(userm.Name()),
			SupportingText: wx.Tu(userm.NameSecondLine()),
			Leading:        wx.NewIcon("person"),
		})
	}

	return items
}
