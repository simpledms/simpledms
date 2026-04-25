package browse

import (
	"fmt"

	"entgo.io/ent/dialect/sql"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/timex"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
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

func (qq *FileInfoPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

func (qq *FileInfoPartial) Widget(ctx ctxx.Context, data *FileInfoPartialData) *widget.ScrollableContent {
	filem := qq.infra.FileRepo.GetX(ctx, data.FileID) // TODO inject?
	currentVersion := filem.CurrentVersion(ctx)

	ocrSucceededAt := widget.Tu("-")
	if filem.Data.OcrSuccessAt != nil && !filem.Data.OcrSuccessAt.IsZero() {
		ocrSucceededAt = widget.Tu(timex.NewDateTime(*filem.Data.OcrSuccessAt).String(ctx.MainCtx().LanguageBCP47))
	}

	sha256 := widget.Tu("-")
	if currentVersion.Data.Sha256 != "" {
		sha256 = widget.Tu(currentVersion.Data.Sha256)
	}

	items := []*widget.ListItem{
		{
			Headline:       widget.T("File size"),
			SupportingText: widget.Tu(currentVersion.SizeString()),
		},
		{
			Headline:       widget.T("MIME type"),
			SupportingText: widget.Tu(currentVersion.Data.MimeType),
		},
		{
			Headline:       widget.T("SHA-256 hash"),
			SupportingText: sha256,
		},
		{
			Headline:       widget.T("Original filename"),
			SupportingText: widget.Tu(currentVersion.Data.Filename),
		},
		{
			Headline:       widget.T("Uploaded at"),
			SupportingText: widget.Tu(timex.NewDateTime(filem.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
		},
		{
			Headline:       widget.T("Version"),
			SupportingText: qq.versionLabel(ctx, filem),
		},
		{
			Headline:       widget.T("Current version uploaded at"),
			SupportingText: widget.Tu(timex.NewDateTime(currentVersion.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
		},
		/*
			{
				Headline:       wx.T("Uploaded by"),
				SupportingText: wx.T(filem.Data.UpdatedBy),
			},
		*/
		{
			Headline:       widget.T("OCR succeeded at"), // TODO naming
			SupportingText: ocrSucceededAt,
		},
		// TODO collapsable OCR content
		// TODO last editied at/by (based on version)
		// TODO copied to final location?
	}

	if !filem.Data.DeletedAt.IsZero() {
		// order is good, because oriented on lifecycle
		items = append(items, []*widget.ListItem{
			{
				Headline:       widget.T("Deleted at"),
				SupportingText: widget.T(timex.NewDateTime(filem.Data.DeletedAt).String(ctx.MainCtx().LanguageBCP47)),
			},
			/* TODO
			{
				Headline:       wx.T("Deleted by"),
				SupportingText: wx.T(""),
			},
			*/
		}...)

	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.ID(),
		},
		GapY: true,
		Children: &widget.List{
			Widget: widget.Widget[widget.List]{
				ID: "fileInfoList",
			},
			Children: items,
		},
		MarginY: true,
	}
}

func (qq *FileInfoPartial) versionLabel(ctx ctxx.Context, filem *filemodel.File) *widget.Text {
	versionData := filem.Data.QueryFileVersions().Order(fileversion.ByVersionNumber(sql.OrderDesc())).FirstX(ctx)
	return widget.Tu(fmt.Sprintf("%d", versionData.VersionNumber))
}

func (qq *FileInfoPartial) ID() string {
	return "fileInfo"
}
