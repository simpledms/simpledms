package dashboard

import (
	"log"

	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
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

func (qq *DashboardPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	widget, err := qq.Widget(ctx)
	if err != nil {
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "Dashboard", widget)
}

func (qq *DashboardPage) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	fabs := []*widget.FloatingActionButton{}
	dashboardCardsWidget, err := qq.actions.DashboardCardsPartial.Widget(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	dashboardCardsContent := &widget.Container{
		HTMXAttrs: widget.HTMXAttrs{
			HxGet:     "/",
			HxTrigger: events.HxTrigger(events.AccountDeleted),
			HxTarget:  "#content",
		},
		Child: dashboardCardsWidget,
	}

	mainLayout := &widget.MainLayout{
		// MainCtx is necessary when navigating back from space, otherwise all menu items are rendered
		Navigation: partial.NewNavigationRail(ctx.MainCtx(), qq.infra, "dashboard", fabs),
		Content: &widget.DefaultLayout{
			AppBar:        qq.appBar(ctx),
			Content:       dashboardCardsContent,
			WithPoweredBy: false,
		},
	}

	return mainLayout, nil
}

func (qq *DashboardPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "dashboard",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T("Dashboard"),
		},
		Actions: []widget.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},*/
		},
	}
}
