package dashboard

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SystemCardsPartialData struct{}

type SystemCardsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSystemCardsPartial(infra *common.Infra, actions *Actions) *SystemCardsPartial {
	config := actionx.NewConfig(
		actions.Route("system-cards-partial"),
		true,
	)
	return &SystemCardsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SystemCardsPartial) Data() *SystemCardsPartialData {
	return &SystemCardsPartialData{}
}

func (qq *SystemCardsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	widget, err := qq.Widget(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return qq.infra.Renderer().Render(rw, ctx, widget)
}

func (qq *SystemCardsPartial) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return nil, e.NewHTTPErrorf(http.StatusForbidden, "You must be an admin to access system settings.")
	}

	grids := qq.actions.DashboardCardsPartial.SystemGrids(ctx)

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.id(),
		},
		GapY: true,
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.AppInitialized,
				event.AppUnlocked,
				event.AppPassphraseChanged,
				event.UploadLimitUpdated,
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

func (qq *SystemCardsPartial) id() string {
	return "systemCards"
}
