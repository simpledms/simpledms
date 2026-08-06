package trash

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type TrashRootPage struct {
	infra   *common.Infra
	actions *Actions
}

func NewTrashRootPage(infra *common.Infra, actions *Actions) *TrashRootPage {
	return &TrashRootPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *TrashRootPage) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	viewx := qq.widget(ctx)

	if req.Header.Get("HX-Request") == "" {
		viewx = partial.NewBase(widget.T("Trash"), viewx)
	}

	return qq.infra.Renderer().Render(rw, ctx, viewx)
}

func (qq *TrashRootPage) widget(ctx ctxx.Context) renderable.Renderable {
	mainLayout := &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "trash", nil),
		Content: &widget.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List:   qq.actions.TrashListPartial.Widget(ctx, qq.actions.TrashListPartial.Data("")),
		},
	}
	return mainLayout
}

func (qq *TrashRootPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("delete"),
		LeadingAltMobile: partial.NewNavigationRailToggle(),
		Title:            widget.T("Trash"),
	}
}
