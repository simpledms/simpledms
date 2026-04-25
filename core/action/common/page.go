package common

import (
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/uix/partial"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

// TODO correct place?
type Page struct {
	// infra *common.Infra
	// title string // TODO or page title?
}

/*
func NewPage(infra *common.Infra, title string) *Page {
	return &Page{
		infra: infra,
		title: title,
	}
}
*/

func (qq *Page) Render(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	infra *common.Infra,
	title string,
	viewx renderable.Renderable,
) error {
	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		titlex := wx.Tuf("%s | SimpleDMS", wx.T(title).String(ctx))
		viewx = partial.NewBase(titlex, viewx)
	}

	return infra.Renderer().Render(rw, ctx, viewx)
}
