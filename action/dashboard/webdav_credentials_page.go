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
		Content: &widget.View{
			Children: []widget.IWidget{
				&widget.ListDetailLayout{
					AppBar: qq.appBar(data),
					List:   overview,
				},
				autil.DefaultSideSheetTrigger("webDAVCredentialsDefaultSideSheetTrigger", widget.HTMXAttrs{
					HxPost: qq.actions.WebDAVCredentialFilterDialog.Endpoint(),
				}),
			},
		},
	}, nil
}

func (qq *WebDAVCredentialsPage) appBar(data *WebDAVCredentialListPartialData) *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("vpn_key"),
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("WebDAV credentials"),
		},
		Actions: []widget.IWidget{qq.filterButton(data, false)},
	}
}

func (qq *WebDAVCredentialsPage) filterButton(
	data *WebDAVCredentialListPartialData,
	isOOB bool,
) *widget.Container {
	return newCredentialFilterButton(
		"webDAVCredentialFilterButton",
		widget.T("Filter WebDAV credentials"),
		qq.actions.WebDAVCredentialFilterDialog.Endpoint(),
		data.CredentialStatusValues,
		isOOB,
	)
}
