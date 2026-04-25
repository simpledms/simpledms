package trash

import (
	"log"
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/events"
	partial2 "github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type TrashWithSelectionPage struct {
	infra   *common.Infra
	actions *Actions
}

func NewTrashWithSelectionPage(infra *common.Infra, actions *Actions) *TrashWithSelectionPage {
	return &TrashWithSelectionPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *TrashWithSelectionPage) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	fileIDStr := req.PathValue("file_id")
	if fileIDStr == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file id provided.")
	}

	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, fileIDStr)
	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File preview is not available for folders.")
	}

	state := autil.StateX[FileTabsPartialState](rw, req)
	// commented on 28.01.2026
	// rw.Header().Set("HX-Push-Url", route.TrashFileWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()))

	viewx, err := qq.widget(rw, req, ctx, state, filex)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not render widget")
	}

	qq.render(rw, req, ctx, viewx)
	return nil
}

func (qq *TrashWithSelectionPage) render(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	viewx renderable.Renderable,
) {
	if req.Header.Get("HX-Request") == "" {
		viewx = partial2.NewBase(widget.T("Trash"), viewx)
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
}

func (qq *TrashWithSelectionPage) widget(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	state *FileTabsPartialState,
	filex *filemodel.File,
) (renderable.Renderable, error) {
	filePreview, err := qq.filePreview(ctx, state, filex)
	if err != nil {
		return nil, err
	}

	if req.Header.Get("HX-Target") == "details" {
		return filePreview, nil
	}

	listDetailLayout := &widget.ListDetailLayout{
		AppBar: qq.appBar(ctx),
		List:   qq.actions.TrashListPartial.Widget(ctx, qq.actions.TrashListPartial.Data(filex.Data.PublicID.String())),
		Detail: filePreview,
	}

	mainLayout := &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "trash", nil),
		Content:    listDetailLayout,
	}
	return mainLayout, nil
}

func (qq *TrashWithSelectionPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("delete"),
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title:            widget.T("Trash"),
	}
}

func (qq *TrashWithSelectionPage) filePreview(
	ctx ctxx.Context,
	state *FileTabsPartialState,
	filex *filemodel.File,
) (*widget.DetailsWithSheet, error) {
	title := widget.T("Preview")
	if ctx.SpaceCtx().Space.IsFolderMode {
		title = widget.Tu(filex.Data.Name)
	}

	fileDetailsSideSheet := qq.actions.FileDetailsSideSheetPartial.Widget(
		ctx,
		qq.actions.FileDetailsSideSheetPartial.Data(filex.Data.PublicID.String()),
		state,
	)

	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	return &widget.DetailsWithSheet{
		AppBar: qq.previewAppBar(ctx, title, filex),
		Child: &widget.Column{
			Children: []widget.IWidget{
				&widget.FilePreview{
					FileURL:  route.TrashDownloadInline(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
					Filename: filex.Filename(ctx),
					MimeType: filex.CurrentVersion(ctxWithDeleted).Data.MimeType,
				},
			},
		},
		SideSheet: fileDetailsSideSheet,
	}, nil
}

func (qq *TrashWithSelectionPage) previewAppBar(ctx ctxx.Context, title *widget.Text, filex *filemodel.File) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.IconButton{
			Icon:    "close",
			Tooltip: widget.T("Close preview"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet:     route.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				HxOn:      events.DetailsClosed.HxOn("click"),
				HxHeaders: autil.CloseDetailsHeader(),
			},
		},
		Title: &widget.AppBarTitle{
			Text: title,
		},
		Actions: []widget.IWidget{
			&widget.IconButton{
				Icon:    "description",
				Tooltip: widget.T("Show details"),
				HTMXAttrs: widget.HTMXAttrs{
					DialogID: qq.actions.FileDetailsSideSheetPartial.ID(),
				},
			},
			&widget.IconButton{
				Icon:    "restore_from_trash",
				Tooltip: widget.T("Restore"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.actions.RestoreFileCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.RestoreFileCmd.DataWithOptions(filex.Data.PublicID.String())),
					HxConfirm: widget.T("Are you sure?").String(ctx),
				},
			},
			&widget.Link{
				Href:      route.TrashDownload(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
				IsNoColor: true,
				Filename:  filex.Filename(ctx),
				Child: &widget.IconButton{
					Icon:    "download",
					Tooltip: widget.T("Download"),
				},
			},
		},
	}
}
