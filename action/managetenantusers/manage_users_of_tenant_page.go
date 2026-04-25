package managetenantusers

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/uix/partial"

	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
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

func (qq *ManageUsersOfTenantPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "manage-users", fabs),
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
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
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
