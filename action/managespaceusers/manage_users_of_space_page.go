package managespaceusers

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type ManageUsersOfSpacePageState struct {
	UsersOfSpaceListPartialState
}

type ManageUsersOfSpacePage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewManageUsersOfSpace(infra *common.Infra, actions *Actions) *ManageUsersOfSpacePage {
	return &ManageUsersOfSpacePage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *ManageUsersOfSpacePage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[ManageUsersOfSpacePageState](rw, req)
	return qq.Render(rw, req, ctx, qq.infra, "Users", qq.Widget(ctx, state))
}

func (qq *ManageUsersOfSpacePage) Widget(
	ctx ctxx.Context,
	state *ManageUsersOfSpacePageState,
) renderable.Renderable {
	var fabs []*widget.FloatingActionButton
	if ctx.SpaceCtx().UserRoleInSpace() == spacerole.Owner {
		fabs = append(fabs, &widget.FloatingActionButton{
			Icon: "person_add",
			Child: []widget.IWidget{
				widget.NewIcon("person_add"),
				widget.T("Assign user"),
			},
			HTMXAttrs: qq.actions.AssignUserToSpaceCmd.ModalLinkAttrs(
				qq.actions.AssignUserToSpaceCmd.Data(),
				"",
			),
		})
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "manage-users", fabs),
		Content: &widget.DefaultLayout{
			AppBar:  qq.appBar(ctx),
			Content: qq.actions.UsersOfSpaceListPartial.Widget(ctx, &state.UsersOfSpaceListPartialState),
		},
	}
}

func (qq *ManageUsersOfSpacePage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "person",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("Users"),
		},
		Actions: []widget.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},
			*/
		},
	}
}
