package common

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/renderable"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
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
