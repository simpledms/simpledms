package page

import (
	"net/http"
	"strconv"

	"github.com/simpledms/simpledms/action/documenttype"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

// TODO via settings or prefix with manage
type ManageDocumentTypes struct {
	infra   *common.Infra
	actions *documenttype.Actions
}

func NewManageDocumentTypes(infra *common.Infra, actions *documenttype.Actions) *ManageDocumentTypes {
	return &ManageDocumentTypes{
		infra:   infra,
		actions: actions,
	}
}

func (qq *ManageDocumentTypes) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	var viewx renderable.Renderable

	fabs := []*wx.FloatingActionButton{
		{
			Icon:    "add",
			Tooltip: wx.T("Add document type"),
			HTMXAttrs: qq.actions.Create.ModalLinkAttrs(
				qq.actions.Create.Data(""),
				"",
			),
			Child: []wx.IWidget{
				wx.NewIcon("add"),
				wx.T("Add document type"),
			},
		},
	}

	idStr := req.PathValue("id")
	id := 0
	if idStr != "" {
		var err error
		id, err = strconv.Atoi(idStr)
		if err != nil {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Could not convert id to integer.")
		}
	}
	// TODO is this safe? should be on 64 bit system
	id64 := int64(id)

	viewx = &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, "", fabs),
		Content:    qq.actions.Page.WidgetHandler(rw, req, ctx, id64),
	}

	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial2.NewBase(wx.T("Manage document types"), viewx)
	}

	return qq.infra.Renderer().Render(rw, ctx, viewx)
}
