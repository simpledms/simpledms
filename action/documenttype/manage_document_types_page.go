package documenttype

import (
	"net/http"
	"strconv"

	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/model/tenant/library"
)

// TODO via settings or prefix with manage
type ManageDocumentTypesPage struct {
	infra   *common.Infra
	actions *Actions
}

func NewManageDocumentTypesPage(infra *common.Infra, actions *Actions) *ManageDocumentTypesPage {
	return &ManageDocumentTypesPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *ManageDocumentTypesPage) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	var viewx renderable.Renderable

	fabs := []*widget.FloatingActionButton{
		{
			Icon:    "add",
			Tooltip: widget.T("Add document type"),
			HTMXAttrs: qq.actions.CreateCmd.ModalLinkAttrs(
				qq.actions.CreateCmd.Data(""),
				"",
			),
			Child: []widget.IWidget{
				widget.NewIcon("add"),
				widget.T("Add document type"),
			},
		},
	}

	service := library.NewService()
	if !service.SpaceHasMetadata(ctx) {
		fabs = append(fabs, &widget.FloatingActionButton{
			Icon:    "download",
			Tooltip: widget.T("Import from library"),
			FABSize: widget.FABSizeSmall,
			HTMXAttrs: qq.actions.ImportFromLibraryDialog.ModalLinkAttrs(
				qq.actions.ImportFromLibraryDialog.Data(),
				"",
			),
			Child: []widget.IWidget{
				widget.NewIcon("download"),
				widget.T("Import from library"),
			},
		})
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

	viewx = &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "document-types", fabs),
		Content:    qq.actions.DocumentTypePage.WidgetHandler(rw, req, ctx, id64),
	}

	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial.NewBase(widget.T("Manage document types"), viewx)
	}

	return qq.infra.Renderer().Render(rw, ctx, viewx)
}
