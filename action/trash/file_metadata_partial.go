package trash

import (
	"time"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/timex"
)

type FileMetadataPartialData struct {
	FileID string
}

type FileMetadataPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileMetadataPartial(infra *common.Infra, actions *Actions) *FileMetadataPartial {
	return &FileMetadataPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-metadata-partial"),
			true,
		),
	}
}

func (qq *FileMetadataPartial) Data(fileID string) *FileMetadataPartialData {
	return &FileMetadataPartialData{FileID: fileID}
}

func (qq *FileMetadataPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileMetadataPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileMetadataPartial) Widget(ctx ctxx.Context, data *FileMetadataPartialData) *widget.ScrollableContent {
	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)

	items := []*widget.ListItem{
		{
			Headline:       widget.T("Name"),
			SupportingText: widget.Tu(filex.Data.Name),
		},
	}

	docTypeName := "-"
	if filex.Data.DocumentTypeID != 0 {
		docType, err := filex.Data.QueryDocumentType().Only(ctx)
		if err == nil && docType != nil {
			docTypeName = docType.Name
		}
	}
	items = append(items, &widget.ListItem{
		Headline:       widget.T("Document type"),
		SupportingText: widget.Tu(docTypeName),
	})

	if filex.Data.Notes != "" {
		items = append(items, &widget.ListItem{
			Headline:       widget.T("Notes"),
			SupportingText: widget.Tu(filex.Data.Notes),
		})
	}

	appendTime := func(label string, timeValue time.Time) {
		if timeValue.IsZero() {
			return
		}
		items = append(items, &widget.ListItem{
			Headline:       widget.T(label),
			SupportingText: widget.Tu(timex.NewDateTime(timeValue).String(ctx.MainCtx().LanguageBCP47)),
		})
	}

	appendTime("Created at", filex.Data.CreatedAt)
	if filex.Data.ModifiedAt != nil {
		appendTime("Modified at", *filex.Data.ModifiedAt)
	}
	if !filex.Data.DeletedAt.IsZero() {
		appendTime("Deleted at", filex.Data.DeletedAt)
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: "trashFileMetadata",
		},
		GapY: true,
		Children: &widget.List{
			Children: items,
		},
		MarginY: true,
	}
}
