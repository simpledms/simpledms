package trash

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/timex"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
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
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filem := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)
	currentVersion := filem.CurrentVersion(ctxWithDeleted)

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
	}

	if parentName := qq.parentName(ctx, filem.Data.ParentID); parentName != "" {
		items = append(items, &widget.ListItem{
			Headline:       widget.T("Parent folder"),
			SupportingText: widget.Tu(parentName),
		})
	}

	items = append(items, &widget.ListItem{
		Headline:       widget.T("Uploaded at"),
		SupportingText: widget.Tu(timex.NewDateTime(filem.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
	})
	items = append(items, &widget.ListItem{
		Headline:       widget.T("OCR succeeded at"),
		SupportingText: ocrSucceededAt,
	})

	if !filem.Data.DeletedAt.IsZero() {
		items = append(items, &widget.ListItem{
			Headline:       widget.T("Deleted at"),
			SupportingText: widget.T(timex.NewDateTime(filem.Data.DeletedAt).String(ctx.MainCtx().LanguageBCP47)),
		})
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: "trashFileInfo",
		},
		GapY: true,
		Children: &widget.List{
			Widget: widget.Widget[widget.List]{
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
