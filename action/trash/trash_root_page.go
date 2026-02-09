package trash

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
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
		viewx = partial.NewBase(wx.T("Trash"), viewx)
	}

	return qq.infra.Renderer().Render(rw, ctx, viewx)
}

func (qq *TrashRootPage) widget(ctx ctxx.Context) renderable.Renderable {
	mainLayout := &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "trash", nil),
		Content: &wx.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List:   qq.actions.TrashListPartial.Widget(ctx, qq.actions.TrashListPartial.Data("")),
		},
	}
	return mainLayout
}

func (qq *TrashRootPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading:          wx.NewIcon("delete"),
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title:            wx.T("Trash"),
	}
}
