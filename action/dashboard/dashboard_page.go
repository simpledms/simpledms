package dashboard

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type DashboardPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewDashboardPage(infra *common.Infra, actions *Actions) *DashboardPage {
	return &DashboardPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *DashboardPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.Render(rw, req, ctx, qq.infra, "Dashboard", qq.Widget(ctx))
}

func (qq *DashboardPage) Widget(ctx ctxx.Context) renderable.Renderable {
	fabs := []*wx.FloatingActionButton{}

	mainLayout := &wx.MainLayout{
		// MainCtx is necessary when navigating back from space, otherwise all menu items are rendered
		Navigation: partial2.NewNavigationRail(ctx.MainCtx(), "dashboard", fabs),
		Content: &wx.DefaultLayout{
			AppBar:        qq.appBar(ctx),
			Content:       qq.actions.DashboardCards.Widget(ctx),
			WithPoweredBy: true,
		},
	}

	return mainLayout
}

func (qq *DashboardPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "dashboard",
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.T("Dashboard"),
		},
		Actions: []wx.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},*/
		},
	}
}
