package trash

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
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
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filem := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)
	currentVersion := filem.CurrentVersion(ctxWithDeleted)

	ocrSucceededAt := wx.Tu("-")
	if filem.Data.OcrSuccessAt != nil && !filem.Data.OcrSuccessAt.IsZero() {
		ocrSucceededAt = wx.Tu(timex.NewDateTime(*filem.Data.OcrSuccessAt).String(ctx.MainCtx().LanguageBCP47))
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

	if parentName := qq.parentName(ctx, filem.Data.ParentID); parentName != "" {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("Parent folder"),
			SupportingText: wx.Tu(parentName),
		})
	}

	items = append(items, &wx.ListItem{
		Headline:       wx.T("Uploaded at"),
		SupportingText: wx.Tu(timex.NewDateTime(filem.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
	})
	items = append(items, &wx.ListItem{
		Headline:       wx.T("OCR succeeded at"),
		SupportingText: ocrSucceededAt,
	})

	if !filem.Data.DeletedAt.IsZero() {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("Deleted at"),
			SupportingText: wx.T(timex.NewDateTime(filem.Data.DeletedAt).String(ctx.MainCtx().LanguageBCP47)),
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
	if parentID == 0 {
		return ""
	}
	parent, err := ctx.TenantCtx().TTx.File.Query().
		Where(file.ID(parentID), file.IsDirectory(true)).
		Only(ctx)
	if err != nil || parent == nil {
		return ""
	}
	return parent.Name
}
