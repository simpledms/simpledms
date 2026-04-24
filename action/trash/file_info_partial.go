package trash

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type FileInfoPartialData struct {
	FileID string
}

type FileInfoPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileInfoPartial(infra *common.Infra, actions *Actions) *FileInfoPartial {
	config := actionx.NewConfig(
		actions.Route("file-info-partial"),
		true,
	)
	return &FileInfoPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileInfoPartial) Data(fileID string) *FileInfoPartialData {
	return &FileInfoPartialData{
		FileID: fileID,
	}
}

func (qq *FileInfoPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileInfoPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileInfoPartial) Widget(ctx ctxx.Context, data *FileInfoPartialData) *wx.ScrollableContent {
	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	filem := repos.Read.FileByPublicIDWithDeletedX(ctx, data.FileID)
	currentVersion, err := qq.infra.FileSystem().CurrentVersionByFileIDX(ctx, filem.ID)
	if err != nil {
		log.Println(err)
		return &wx.ScrollableContent{}
	}

	ocrSucceededAt := wx.Tu("-")
	if filem.OcrSuccessAt != nil && !filem.OcrSuccessAt.IsZero() {
		ocrSucceededAt = wx.Tu(timex.NewDateTime(*filem.OcrSuccessAt).String(ctx.MainCtx().LanguageBCP47))
	}

	sha256 := wx.Tu("-")
	if currentVersion.Data.Sha256 != "" {
		sha256 = wx.Tu(currentVersion.Data.Sha256)
	}

	items := []*wx.ListItem{
		{
			Headline:       wx.T("File size"),
			SupportingText: wx.Tu(currentVersion.SizeString()),
		},
		{
			Headline:       wx.T("MIME type"),
			SupportingText: wx.Tu(currentVersion.Data.MimeType),
		},
		{
			Headline:       wx.T("SHA-256 hash"),
			SupportingText: sha256,
		},
		{
			Headline:       wx.T("Original filename"),
			SupportingText: wx.Tu(currentVersion.Data.Filename),
		},
	}

	if parentName := qq.parentName(ctx, filem.ParentID); parentName != "" {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("Parent folder"),
			SupportingText: wx.Tu(parentName),
		})
	}

	items = append(items, &wx.ListItem{
		Headline:       wx.T("Uploaded at"),
		SupportingText: wx.Tu(timex.NewDateTime(filem.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
	})
	items = append(items, &wx.ListItem{
		Headline:       wx.T("OCR succeeded at"),
		SupportingText: ocrSucceededAt,
	})

	if !filem.DeletedAt.IsZero() {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("Deleted at"),
			SupportingText: wx.T(timex.NewDateTime(filem.DeletedAt).String(ctx.MainCtx().LanguageBCP47)),
		})
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: "trashFileInfo",
		},
		GapY: true,
		Children: &wx.List{
			Widget: wx.Widget[wx.List]{
				ID: "trashFileInfoList",
			},
			Children: items,
		},
		MarginY: true,
	}
}

func (qq *FileInfoPartial) parentName(ctx ctxx.Context, parentID int64) string {
	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	return repos.Read.ParentDirectoryNameByIDX(ctx, parentID)
}
