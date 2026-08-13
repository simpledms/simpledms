package managetenantusers

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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
	fabs := []*widget.FloatingActionButton{
		{
			Icon: "add",
			Child: []widget.IWidget{
				widget.NewIcon("add"),
				widget.T("Create user"),
			},
			HTMXAttrs: qq.actions.CreateUserCmd.ModalLinkAttrs(
				qq.actions.CreateUserCmd.Data(tenantrole.User, "", "", "", ctx.MainCtx().Account.Language),
				"",
			),
		},
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(
			ctx,
			qq.infra,
			partial2.TenantUsersNavigationRailValue(ctx.TenantCtx().TenantID),
			fabs,
		),
		Content: &widget.DefaultLayout{
			AppBar:  qq.appBar(ctx),
			Content: qq.actions.UserListPartial.Widget(ctx, &state.UserListPartialState),
		},
	}
}

func (qq *ManageUsersOfTenantPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "person",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.Tf("Users «%s»", ctx.TenantCtx().Tenant.Name),
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
