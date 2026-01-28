package trash

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	viewx renderable.Renderable,
) {
	if req.Header.Get("HX-Request") == "" {
		viewx = partial.NewBase(wx.T("Trash"), viewx)
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
}

func (qq *TrashWithSelectionPage) widget(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	state *FileTabsPartialState,
	filex *model.File,
) (renderable.Renderable, error) {
	filePreview, err := qq.filePreview(ctx, state, filex)
	if err != nil {
		return nil, err
	}

	if req.Header.Get("HX-Target") == "details" {
		return filePreview, nil
	}

	listDetailLayout := &wx.ListDetailLayout{
		AppBar: qq.appBar(ctx),
		List:   qq.actions.TrashListPartial.Widget(ctx, qq.actions.TrashListPartial.Data(filex.Data.PublicID.String())),
		Detail: filePreview,
	}

	mainLayout := &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "trash", nil),
		Content:    listDetailLayout,
	}
	return mainLayout, nil
}

func (qq *TrashWithSelectionPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading:          wx.NewIcon("delete"),
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title:            wx.T("Trash"),
	}
}

func (qq *TrashWithSelectionPage) filePreview(
	ctx ctxx.Context,
	state *FileTabsPartialState,
	filex *model.File,
) (*wx.DetailsWithSheet, error) {
	title := wx.T("Preview")
	if ctx.SpaceCtx().Space.IsFolderMode {
		title = wx.Tu(filex.Data.Name)
	}

	fileDetailsSideSheet := qq.actions.FileDetailsSideSheetPartial.Widget(
		ctx,
		qq.actions.FileDetailsSideSheetPartial.Data(filex.Data.PublicID.String()),
		state,
	)

	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	return &wx.DetailsWithSheet{
		AppBar: qq.previewAppBar(ctx, title, filex),
		Child: &wx.Column{
			Children: []wx.IWidget{
				&wx.FilePreview{
					FileURL:  route.TrashDownloadInline(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
					Filename: filex.Filename(ctx),
					MimeType: filex.CurrentVersion(ctxWithDeleted).Data.MimeType,
				},
			},
		},
		SideSheet: fileDetailsSideSheet,
	}, nil
}

func (qq *TrashWithSelectionPage) previewAppBar(ctx ctxx.Context, title *wx.Text, filex *model.File) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.IconButton{
			Icon: "close",
			HTMXAttrs: wx.HTMXAttrs{
				HxGet:     route.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				HxOn:      event.DetailsClosed.HxOn("click"),
				HxHeaders: autil.CloseDetailsHeader(),
			},
		},
		Title: &wx.AppBarTitle{
			Text: title,
		},
		Actions: []wx.IWidget{
			&wx.IconButton{
				Icon: "description",
				HTMXAttrs: wx.HTMXAttrs{
					DialogID: qq.actions.FileDetailsSideSheetPartial.ID(),
				},
			},
			&wx.IconButton{
				Icon:    "restore_from_trash",
				Tooltip: wx.T("Restore"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.actions.RestoreFileCmd.Endpoint(),
					HxVals:    util.JSON(qq.actions.RestoreFileCmd.DataWithOptions(filex.Data.PublicID.String())),
					HxConfirm: wx.T("Are you sure?").String(ctx),
				},
			},
			&wx.Link{
				Href:      route.TrashDownload(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
				IsNoColor: true,
				Filename:  filex.Filename(ctx),
				Child: &wx.IconButton{
					Icon: "download",
				},
			},
		},
	}
}
