package browse

import (
	"log"
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

// TODO rename to BrowseFile?
type BrowseWithSelectionPage struct {
	infra   *common.Infra
	actions *Actions
}

func NewBrowseWithSelectionPage(infra *common.Infra, actions *Actions) *BrowseWithSelectionPage {
	return &BrowseWithSelectionPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *BrowseWithSelectionPage) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	dirIDStr := req.PathValue("dir_id")
	fileIDStr := req.PathValue("file_id")

	if dirIDStr == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No dir id provided.")
	}
	if fileIDStr == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file id provided.")
	}

	filex := qq.infra.FileRepo.GetX(ctx, fileIDStr)
	dirx := qq.infra.FileRepo.GetX(ctx, dirIDStr)

	state := autil.StateX[FilePreviewPartialState](rw, req)

	browsePage, err := qq.widget(rw, req, ctx, state, dirx, filex)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not render widget")
	}

	qq.render(rw, req, ctx, browsePage)
	return nil
}

func (qq *BrowseWithSelectionPage) render(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	viewx renderable.Renderable,
) {
	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial.NewBase(widget.T("Files"), viewx)
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
}

func (qq *BrowseWithSelectionPage) widget(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	state *FilePreviewPartialState,
	dirx *filemodel.File,
	filex *filemodel.File,
) (renderable.Renderable, error) {
	filePreview, err := qq.actions.FilePreviewPartial.Widget(
		ctx,
		state,
		dirx,
		filex,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if req.Header.Get("HX-Target") == "details" {
		return filePreview, nil
	}

	listDetailsLayout := qq.actions.ListDirPartial.WidgetHandler(
		rw,
		req,
		ctx,
		dirx.Data.PublicID.String(),  // TODO pass dirx?
		filex.Data.PublicID.String(), // TODO pass filex?
	)
	listDetailsLayout.Detail = filePreview

	fabs := []*widget.FloatingActionButton{
		{
			Icon: "upload_file",
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:        qq.actions.FileUploadDialogPartial.Endpoint(),
				HxVals:        util.JSON(qq.actions.FileUploadDialogPartial.Data(dirx.Data.PublicID.String(), false)),
				LoadInPopover: true,
			},
			Child: []widget.IWidget{
				widget.NewIcon("upload_file"),
				widget.T("Upload file"),
			},
		},
	}

	if ctx.SpaceCtx().Space.IsFolderMode {
		fabs = append(fabs, &widget.FloatingActionButton{
			FABSize: widget.FABSizeSmall,
			Icon:    "create_new_folder",
			HTMXAttrs: qq.actions.MakeDirCmd.ModalLinkAttrs(
				qq.actions.MakeDirCmd.Data(dirx.Data.PublicID.String(), ""),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
			Child: []widget.IWidget{
				widget.NewIcon("create_new_folder"),
				widget.T("Create directory"),
			},
		})
	}

	/*
		fileDetailsSideSheet := qq.actions.FileDetailsSideSheetPartial.Widget(
			ctx,
			qq.actions.FileDetailsSideSheetPartial.Data(dirx.Data.PublicID.String(), filex.Data.PublicID.String()),
			state,
		)
	*/

	mainLayout := &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "browse", fabs),
		Content:    listDetailsLayout,
		// SideSheet:  fileDetailsSideSheet,
	}
	return mainLayout, nil
}
