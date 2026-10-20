package dashboard

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SystemStatusPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewSystemStatusPage(infra *common.Infra, actions *Actions) *SystemStatusPage {
	return &SystemStatusPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *SystemStatusPage) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(
			http.StatusForbidden,
			"You must be an admin to access the system status.",
		)
	}

	widget, err := qq.Widget(ctx)
	if err != nil {
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "System status", widget)
}

func (qq *SystemStatusPage) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	fabs := []*widget.FloatingActionButton{}
	systemStatusWidget, err := qq.actions.SystemStatusPartial.Widget(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx.MainCtx(), qq.infra, "system-status", fabs),
		Content: &widget.DefaultLayout{
			AppBar:        qq.appBar(),
			Content:       systemStatusWidget,
			WithPoweredBy: false,
		},
	}, nil
}

func (qq *SystemStatusPage) appBar() *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("monitor_heart"),
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("System status"),
		},
		Actions: []widget.IWidget{
			qq.actions.SystemStatusPartial.RefreshButton(),
		},
	}
}
