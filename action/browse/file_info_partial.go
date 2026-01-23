package browse

import (
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
	filem := qq.infra.FileRepo.GetX(ctx, data.FileID) // TODO inject?
	currentVersion := filem.CurrentVersion(ctx)

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
		{
			Headline:       wx.T("Uploaded at"),
			SupportingText: wx.Tu(timex.NewDateTime(filem.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)), // TODO file or version?
		},
		/*
			{
				Headline:       wx.T("Uploaded by"),
				SupportingText: wx.T(filem.Data.UpdatedBy),
			},
		*/
		{
			Headline:       wx.T("OCR succeeded at"), // TODO naming
			SupportingText: ocrSucceededAt,
		},
		// TODO collapsable OCR content
		// TODO last editied at/by (based on version)
		// TODO copied to final location?
	}

	if !filem.Data.DeletedAt.IsZero() {
		// order is good, because oriented on lifecycle
		items = append(items, []*wx.ListItem{
			{
				Headline:       wx.T("Deleted at"),
				SupportingText: wx.T(timex.NewDateTime(filem.Data.DeletedAt).String(ctx.MainCtx().LanguageBCP47)),
			},
			/* TODO
			{
				Headline:       wx.T("Deleted by"),
				SupportingText: wx.T(""),
			},
			*/
		}...)

	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.ID(),
		},
		GapY: true,
		Children: &wx.List{
			Widget: wx.Widget[wx.List]{
				ID: "fileInfoList",
			},
			Children: items,
		},
		MarginY: true,
	}
}

func (qq *FileInfoPartial) ID() string {
	return "fileInfo"
}
