package managespaceusers

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
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
	return qq.Render(rw, req, ctx, qq.infra, "Manage users", qq.Widget(ctx, state))
}

func (qq *ManageUsersOfSpacePage) Widget(
	ctx ctxx.Context,
	state *ManageUsersOfSpacePageState,
) renderable.Renderable {
	/*
		fabs := []*wx.FloatingActionButton{
			{
				Icon: "add",
				Child: []wx.IWidget{
					wx.NewIcon("add"),
					wx.T("Create property"),
				},
				HTMXAttrs: qq.actions.CreateProperty.ModalLinkAttrs(
					qq.actions.CreateProperty.Data(""),
					"",
				),
			},
		}

	*/

	return &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, "manage-users", nil),
		Content: &wx.DefaultLayout{
			AppBar:  qq.appBar(ctx),
			Content: qq.actions.UsersOfSpaceListPartial.Widget(ctx, &state.UsersOfSpaceListPartialState),
		},
	}
}

func (qq *ManageUsersOfSpacePage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "person",
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.Tf("Users «%s»", ctx.SpaceCtx().Space.Name),
		},
		Actions: []wx.IWidget{
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
