package managetenantusers

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/app/simpledms/renderable"
	"github.com/simpledms/simpledms/app/simpledms/ui/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type ManageUsersOfTenantPageData struct{}

type ManageUsersOfTenantPageState struct {
	UserListPartialState
}

type ManageUsersOfTenantPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewManageUsersOfTenantPage(infra *common.Infra, actions *Actions) *ManageUsersOfTenantPage {
	return &ManageUsersOfTenantPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *ManageUsersOfTenantPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[ManageUsersOfTenantPageState](rw, req)
	return qq.Render(rw, req, ctx, qq.infra, "Manage users of tenant", qq.Widget(ctx, state))
}

func (qq *ManageUsersOfTenantPage) Widget(ctx ctxx.Context, state *ManageUsersOfTenantPageState) renderable.Renderable {
	fabs := []*wx.FloatingActionButton{
		{
			Icon: "add",
			Child: []wx.IWidget{
				wx.NewIcon("add"),
				wx.T("Create user"),
			},
			HTMXAttrs: qq.actions.CreateUserCmd.ModalLinkAttrs(
				qq.actions.CreateUserCmd.Data(tenantrole.User, "", "", "", ctx.MainCtx().Account.Language),
				"",
			),
		},
	}

	return &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "manage-users", fabs),
		Content: &wx.DefaultLayout{
			AppBar:  qq.appBar(ctx),
			Content: qq.actions.UserListPartial.Widget(ctx, &state.UserListPartialState),
		},
	}
}

func (qq *ManageUsersOfTenantPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "person",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.Tf("Users «%s»", ctx.TenantCtx().Tenant.Name),
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
