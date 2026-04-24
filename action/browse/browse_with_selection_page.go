package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
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

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	filex := repos.Read.FileByPublicIDX(ctx, fileIDStr)
	dirx := repos.Read.FileByPublicIDX(ctx, dirIDStr)

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
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	viewx renderable.Renderable,
) {
	renderFullPage := false
	if req.Header.Get("HX-Request") == "" {
		renderFullPage = true
	}

	if renderFullPage {
		viewx = partial2.NewBase(wx.T("Files"), viewx)
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
}

func (qq *BrowseWithSelectionPage) widget(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	state *FilePreviewPartialState,
	dirx *filemodel.FileDTO,
	filex *filemodel.FileDTO,
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
		dirx.PublicID,  // TODO pass dirx?
		filex.PublicID, // TODO pass filex?
	)
	listDetailsLayout.Detail = filePreview

	fabs := []*wx.FloatingActionButton{
		{
			Icon: "upload_file",
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:        qq.actions.FileUploadDialogPartial.Endpoint(),
				HxVals:        util.JSON(qq.actions.FileUploadDialogPartial.Data(dirx.PublicID, false)),
				LoadInPopover: true,
			},
			Child: []wx.IWidget{
				wx.NewIcon("upload_file"),
				wx.T("Upload file"),
			},
		},
	}

	if ctx.SpaceCtx().Space.IsFolderMode {
		fabs = append(fabs, &wx.FloatingActionButton{
			FABSize: wx.FABSizeSmall,
			Icon:    "create_new_folder",
			HTMXAttrs: qq.actions.MakeDirCmd.ModalLinkAttrs(
				qq.actions.MakeDirCmd.Data(dirx.PublicID, ""),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
			Child: []wx.IWidget{
				wx.NewIcon("create_new_folder"),
				wx.T("Create directory"),
			},
		})
	}

	/*
		fileDetailsSideSheet := qq.actions.FileDetailsSideSheetPartial.Widget(
			ctx,
			qq.actions.FileDetailsSideSheetPartial.Data(dirx.PublicID, filex.PublicID),
			state,
		)
	*/

	mainLayout := &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "browse", fabs),
		Content:    listDetailsLayout,
		// SideSheet:  fileDetailsSideSheet,
	}
	return mainLayout, nil
}
