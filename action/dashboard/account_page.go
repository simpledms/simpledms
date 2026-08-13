package dashboard

import (
	"log"

	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type AccountPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewAccountPage(infra *common.Infra, actions *Actions) *AccountPage {
	return &AccountPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *AccountPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	widget, err := qq.Widget(ctx)
	if err != nil {
		return err
	}

	return qq.Render(rw, req, ctx, qq.infra, "Account", widget)
}

func (qq *AccountPage) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	fabs := []*widget.FloatingActionButton{}
	accountCardsWidget, err := qq.actions.AccountCardsPartial.Widget(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx.MainCtx(), qq.infra, "account", fabs),
		Content: &widget.DefaultLayout{
			AppBar:        qq.appBar(ctx),
			Content:       accountCardsWidget,
			WithPoweredBy: false,
		},
	}, nil
}

func (qq *AccountPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "account_circle",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("Account"),
		},
	}
}
