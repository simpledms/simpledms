package dashboard

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/systemstatus"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type SystemStatusPartialData struct{}

type SystemStatusPartial struct {
	infra   *common.Infra
	actions *Actions
	checker *systemstatus.SystemStatusChecker
	*actionx.Config
}

func NewSystemStatusPartial(
	infra *common.Infra,
	actions *Actions,
	checker *systemstatus.SystemStatusChecker,
) *SystemStatusPartial {
	config := actionx.NewConfig(
		actions.Route("system-status-partial"),
		true,
	)
	return &SystemStatusPartial{
		infra:   infra,
		actions: actions,
		checker: checker,
		Config:  config,
	}
}

func (qq *SystemStatusPartial) Data() *SystemStatusPartialData {
	return &SystemStatusPartialData{}
}

func (qq *SystemStatusPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	widget, err := qq.Widget(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return qq.infra.Renderer().Render(rw, ctx, widget)
}

// Widget checks the connections to all services on each request, thus it is not subscribed to
// any events; it is reloaded by the refresh button in the app bar.
func (qq *SystemStatusPartial) Widget(ctx ctxx.Context) (renderable.Renderable, error) {
	if ctx.MainCtx().Account.Role != mainrole.Admin {
		return nil, e.NewHTTPErrorf(
			http.StatusForbidden,
			"You must be an admin to access the system status.",
		)
	}

	status, err := qq.checker.Check(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.ID(),
		},
		Child: &partial.SystemStatusReport{
			Status:    status,
			CheckedAt: timex.NewDateTime(status.CheckedAt()).String(ctx.MainCtx().LanguageBCP47),
		},
	}, nil
}

// RefreshButton reloads the partial to run all checks again.
func (qq *SystemStatusPartial) RefreshButton() *widget.IconButton {
	return &widget.IconButton{
		Icon:    "refresh",
		Tooltip: widget.T("Check again"),
		Label:   widget.T("Check again"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data()),
			HxTarget: "#" + qq.ID(),
			HxSelect: "#" + qq.ID(),
			HxSwap:   "outerHTML",
		},
	}
}

func (qq *SystemStatusPartial) ID() string {
	return "systemStatus"
}
