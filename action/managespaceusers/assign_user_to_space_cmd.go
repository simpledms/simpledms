package managespaceusers

import (
	"fmt"
	"log"
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/model/tenant/common/spacerole"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
	"github.com/simpledms/simpledms/ui/uix/event"
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
	infra           *common.Infra
	actions         *Actions
	spaceRepository spacemodel.SpaceRepository
	*actionx2.Config
	*autil.FormHelper[AssignUserToSpaceCmdData]
}

func NewAssignUserToSpaceCmd(infra *common.Infra, actions *Actions) *AssignUserToSpaceCmd {
	config := actionx2.NewConfig(actions.Route("assign-user-to-space-cmd"), false)
	return &AssignUserToSpaceCmd{
		infra:           infra,
		actions:         actions,
		spaceRepository: spacemodel.NewEntSpaceRepository(),
		Config:          config,
		FormHelper:      autil.NewFormHelper[AssignUserToSpaceCmdData](infra, config, widget.T("Assign user to space")),
	}
}

func (qq *AssignUserToSpaceCmd) Data() *AssignUserToSpaceCmdData {
	return &AssignUserToSpaceCmdData{}
}

func (qq *AssignUserToSpaceCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	err = spacemodel.NewSpaceWithRepository(ctx.SpaceCtx().Space, qq.spaceRepository).
		AssignUser(ctx, data.UserID, data.Role)
	if err != nil {
		return mapSpaceError(err)
	}

	// TODO send message to user via Chat?
	rw.AddRenderables(widget.NewSnackbarf("User assigned to space successfully."))
	rw.Header().Set("HX-Trigger", event.UserAssignedToSpace.String())

	return nil
}

func (qq *AssignUserToSpaceCmd) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
			actionx2.ResponseWrapper(wrapper),
			hxTarget,
		),
	)
}

func (qq *AssignUserToSpaceCmd) Form(
	ctx ctxx.Context,
	data *AssignUserToSpaceCmdFormData,
	wrapper actionx2.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	if data.Role == spacerole.SpaceRole(0) {
		data.Role = spacerole.User
	}

	form := &widget.Form{
		Widget: widget.Widget[widget.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					widget.NewFormFields(ctx, data),
					qq.userList(ctx),
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		widget.T("Assign user to space"),
		widget.T("Save"),
		form,
		wrapper,
		widget.DialogLayoutStable,
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

func (qq *AssignUserToSpaceCmd) userList(ctx ctxx.Context) widget.IWidget {
	listItems := qq.userListItems(ctx)

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.userListID(),
		},
		Children: &widget.List{
			Children: listItems,
		},
	}
}

func (qq *AssignUserToSpaceCmd) userListID() string {
	return "userList"
}

func (qq *AssignUserToSpaceCmd) userListItems(ctx ctxx.Context) interface{} {
	// TODO implement pagination?

	var items []*widget.ListItem

	unassignedUsers, err := qq.spaceRepository.UnassignedUsers(ctx, ctx.SpaceCtx().Space.ID)
	if err != nil {
		log.Println(err)
		return []*widget.ListItem{
			{
				Headline:       widget.T("Could not load users."),
				SupportingText: widget.T("Please reload the page and try again."),
			},
		}
	}

	if len(unassignedUsers) == 0 {
		items = append(items, &widget.ListItem{
			Headline:       widget.T("No unassigned users available."),
			SupportingText: widget.T("Please create a user in the organization user management first."), // TODO link
		})
		return items
	}

	for _, unassignedUser := range unassignedUsers {
		userm := usermodel.NewUser(unassignedUser)
		items = append(items, &widget.ListItem{
			RadioGroupName: "UserID",
			RadioValue:     fmt.Sprintf("%s", unassignedUser.PublicID),
			Headline:       widget.Tu(userm.Name()),
			SupportingText: widget.Tu(userm.NameSecondLine()),
			Leading:        widget.NewIcon("person"),
		})
	}

	return items
}
