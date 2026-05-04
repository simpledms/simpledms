package dashboard

import (
	"log"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AccountCardsPartialData struct{}

type AccountCardsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAccountCardsPartial(infra *common.Infra, actions *Actions) *AccountCardsPartial {
	config := actionx.NewConfig(
		actions.Route("account-cards-partial"),
		true,
	).EnableSetupSessionAccess()
	return &AccountCardsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *AccountCardsPartial) Data() *AccountCardsPartialData {
	return &AccountCardsPartialData{}
}

func (qq *AccountCardsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	widget, err := qq.Widget(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return qq.infra.Renderer().Render(rw, ctx, widget)
}

func (qq *AccountCardsPartial) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	grids, err := qq.actions.DashboardCardsPartial.AccountGrids(ctx)
	if err != nil {
		return nil, err
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.id(),
		},
		GapY: true,
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.InitialPasswordSet,
				event.TemporaryPasswordCleared,
				event.PasswordChanged,
				event.AccountUpdated,
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data()),
			HxTarget: "#" + qq.id(),
			HxSelect: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Child: grids,
	}, nil
}

func (qq *AccountCardsPartial) id() string {
	return "accountCards"
}
