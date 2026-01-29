package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/event"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePreviewPartialData struct {
	CurrentDirID string
	FileID       string
}

type FilePreviewPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

type FilePreviewPartialState struct {
	ListDirPartialState
	ActiveTab string `url:"tab,omitempty"`
}

func NewFilePreviewPartial(infra *common.Infra, actions *Actions) *FilePreviewPartial {
	return &FilePreviewPartial{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("file-preview-partial"),
			true,
		),
	}
}

func (qq *FilePreviewPartial) Data(currentDirID, fileID string) *FilePreviewPartialData {
	return &FilePreviewPartialData{
		CurrentDirID: currentDirID,
		FileID:       fileID,
	}
}

func (qq *FilePreviewPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilePreviewPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[FilePreviewPartialState](rw, req)

	// filex := ctx.TenantCtx().TTx.File.GetX(ctx, data.FileID)
	dirx := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	viewx, err := qq.Widget(ctx, state, dirx, filex)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "rendering failed")
	}

	qq.infra.Renderer().RenderX(rw, ctx, viewx)
	return nil
}

func (qq *FilePreviewPartial) Widget(
	ctx ctxx.Context,
	state *FilePreviewPartialState,
	dirx *model.File,
	filex *model.File,
) (*wx.DetailsWithSheet, error) {
	// TODO action.ShowFileData or primitive types?
	//		is partial bound to action?
	//
	// TODO stream file (also necessary for download)

	// FIXME update on changes...
	// soft delete filter is not applied via TagAssignment
	// tagsCount := qq.infra.Client().File.GetX(ctx, filepathxx.FileID).QueryTags().CountX(ctx)

	title := wx.T("Preview") // TODO or `File preview`?
	if ctx.SpaceCtx().Space.IsFolderMode {
		title = wx.Tu(filex.Data.Name)
	}

	fileDetailsSideSheet := qq.actions.FileDetailsSideSheetPartial.Widget(
		ctx,
		qq.actions.FileDetailsSideSheetPartial.Data(
			dirx.Data.PublicID.String(),
			filex.Data.PublicID.String(),
		),
		state,
	)

	return &wx.DetailsWithSheet{
		AppBar: qq.appBar(ctx, dirx.Data.PublicID.String(), title, filex),
		Child: &wx.Column{
			Children: []wx.IWidget{
				&wx.FilePreview{
					FileURL:  route2.DownloadInline(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
					Filename: filex.Filename(ctx),
					MimeType: filex.CurrentVersion(ctx).Data.MimeType,
				},
				/*&wx.ScrollableContent{
					// TODO consider a custom type for FilePreviewx?
					Children:
				},*/
			},
		},
		SideSheet: fileDetailsSideSheet,
	}, nil
}

func (qq *FilePreviewPartial) appBar(ctx ctxx.Context, dirID string, title *wx.Text, filex *model.File) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.IconButton{
			Icon:    "close",
			Tooltip: wx.T("Close preview"),
			// TODO use link instead?
			HTMXAttrs: wx.HTMXAttrs{
				HxGet:     route2.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, dirID),
				HxOn:      event.DetailsClosed.HxOn("click"),
				HxHeaders: autil.CloseDetailsHeader(),
			},
		},
		Title: &wx.AppBarTitle{
			Text: title,
		},
		Actions: []wx.IWidget{
			&wx.IconButton{
				// TODO other icon if already open or hide...
				Icon:    "description", // right_panel_open, clarify, tune, description, info, ...?
				Tooltip: wx.T("Show details"),
				HTMXAttrs: wx.HTMXAttrs{
					DialogID: qq.actions.FileDetailsSideSheetPartial.ID(),
				},
			},
			&wx.Link{
				Href:      route2.Download(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
				IsNoColor: true,
				Filename:  filex.Filename(ctx),
				Child: &wx.IconButton{
					Icon:    "download",
					Tooltip: wx.T("Download"),
				},
			},
			/*
				&wx.IconButton{
					Icon: "more_vert",
					Children: &wx.Menu{
						Items: []*wx.MenuItem{
							{
								LeadingIcon:          "download",
								Label:                wx.T("Download"),
								DownloadLinkURL:      route.Download(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Data.PublicID.String()),
								DownloadLinkFilename: filex.Filename(ctx),
							},
						},
					},
				},
			*/
		},
	}
}
