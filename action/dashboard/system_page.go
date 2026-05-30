package dashboard

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SystemPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewSystemPage(infra *common.Infra, actions *Actions) *SystemPage {
	return &SystemPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *SystemPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to access system settings.")
	}

	widget, err := qq.Widget(ctx)
	if err != nil {
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "System", widget)
}

func (qq *SystemPage) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	fabs := []*wx.FloatingActionButton{}
	systemCardsWidget, err := qq.actions.SystemCardsPartial.Widget(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx.MainCtx(), qq.infra, "system", fabs),
		Content: &wx.DefaultLayout{
			AppBar:        qq.appBar(ctx),
			Content:       systemCardsWidget,
			WithPoweredBy: false,
		},
	}, nil
}

func (qq *SystemPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "settings",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &wx.AppBarTitle{
			Text: wx.T("System"),
		},
	}
}
