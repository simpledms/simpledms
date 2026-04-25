package trash

import (
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	partial2 "github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	partial "github.com/simpledms/simpledms/ui/uix/partial"
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
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	viewx := qq.widget(ctx)

	if req.Header.Get("HX-Request") == "" {
		viewx = partial2.NewBase(widget.T("Trash"), viewx)
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
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title:            widget.T("Trash"),
	}
}
