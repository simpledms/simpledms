package managespaceusers

import (
	autil "github.com/marcobeierer/go-core/action/util"

	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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

func (qq *ManageUsersOfSpacePage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "manage-users", nil),
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
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.Tf("Users «%s»", ctx.SpaceCtx().Space.Name),
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
