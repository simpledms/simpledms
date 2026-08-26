package dashboard

import (
	"log"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type WebDAVCredentialsPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewWebDAVCredentialsPage(infra *common.Infra, actions *Actions) *WebDAVCredentialsPage {
	return &WebDAVCredentialsPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *WebDAVCredentialsPage) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	state := autil.StateX[WebDAVCredentialListPartialData](rw, req)
	page, err := qq.Widget(ctx, req, state)
	if err != nil {
		log.Println(err)
		return err
	}
	return qq.Render(rw, req, ctx, qq.infra, "WebDAV credentials", page)
}

func (qq *WebDAVCredentialsPage) Widget(
	ctx ctxx.Context,
	req *httpx.Request,
	data *WebDAVCredentialListPartialData,
) (renderable.Renderable, error) {
	overview, err := qq.actions.WebDAVCredentialListPartial.Widget(
		ctx,
		req,
		qq.actions.WebDAVCredentialListPartial.Data("", data.CredentialStatusValues...),
	)
	if err != nil {
		return nil, err
	}
	createAttrs := qq.actions.CreateWebDAVCredentialCmd.ModalLinkAttrs(
		qq.actions.CreateWebDAVCredentialCmd.Data("", ""),
	)
	fabs := []*widget.FloatingActionButton{{
		Icon: "add",
		Child: []widget.IWidget{
			widget.NewIcon("add"),
			widget.T("Create WebDAV credential"),
		},
		HTMXAttrs: createAttrs,
	}}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(
			ctx.MainCtx(),
			qq.infra,
			"webdav-credentials",
			fabs,
		),
		Content: &widget.ListDetailLayout{
			AppBar: qq.appBar(),
			List:   overview,
		},
	}, nil
}

func (qq *WebDAVCredentialsPage) appBar() *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("vpn_key"),
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("WebDAV credentials"),
		},
		Actions: []widget.IWidget{qq.filterButton()},
	}
}

func (qq *WebDAVCredentialsPage) filterButton() *widget.IconButton {
	return &widget.IconButton{
		Icon:    "filter_alt",
		Tooltip: widget.T("Filter WebDAV credentials"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:        qq.actions.WebDAVCredentialFilterDialog.Endpoint(),
			LoadInPopover: true,
		},
	}
}
