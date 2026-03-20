package dashboard

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
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
	widget, err := qq.Widget(ctx)
	if err != nil {
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "Dashboard", widget)
}

func (qq *DashboardPage) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	fabs := []*wx.FloatingActionButton{}
	dashboardCardsWidget, err := qq.actions.DashboardCardsPartial.Widget(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	dashboardCardsContent := &wx.Container{
		HTMXAttrs: wx.HTMXAttrs{
			HxGet:     "/",
			HxTrigger: event.HxTrigger(event.AccountDeleted),
			HxTarget:  "#content",
		},
		Child: dashboardCardsWidget,
	}

	mainLayout := &wx.MainLayout{
		// MainCtx is necessary when navigating back from space, otherwise all menu items are rendered
		Navigation: partial2.NewNavigationRail(ctx.MainCtx(), qq.infra, "dashboard", fabs),
		Content: &wx.DefaultLayout{
			AppBar:        qq.appBar(ctx),
			Content:       dashboardCardsContent,
			WithPoweredBy: false,
		},
	}

	return mainLayout, nil
}

func (qq *DashboardPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "dashboard",
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
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
