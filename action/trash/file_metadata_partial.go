package trash

import (
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
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

func (qq *FileMetadataPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

func (qq *FileMetadataPartial) Widget(ctx ctxx.Context, data *FileMetadataPartialData) *wx.ScrollableContent {
	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)

	items := []*wx.ListItem{
		{
			Headline:       wx.T("Name"),
			SupportingText: wx.Tu(filex.Data.Name),
		},
	}

	docTypeName := "-"
	if filex.Data.DocumentTypeID != 0 {
		docType, err := filex.Data.QueryDocumentType().Only(ctx)
		if err == nil && docType != nil {
			docTypeName = docType.Name
		}
	}
	items = append(items, &wx.ListItem{
		Headline:       wx.T("Document type"),
		SupportingText: wx.Tu(docTypeName),
	})

	if filex.Data.Notes != "" {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("Notes"),
			SupportingText: wx.Tu(filex.Data.Notes),
		})
	}

	appendTime := func(label string, timeValue time.Time) {
		if timeValue.IsZero() {
			return
		}
		items = append(items, &wx.ListItem{
			Headline:       wx.T(label),
			SupportingText: wx.Tu(timex.NewDateTime(timeValue).String(ctx.MainCtx().LanguageBCP47)),
		})
	}

	appendTime("Created at", filex.Data.CreatedAt)
	if filex.Data.ModifiedAt != nil {
		appendTime("Modified at", *filex.Data.ModifiedAt)
	}
	if !filex.Data.DeletedAt.IsZero() {
		appendTime("Deleted at", filex.Data.DeletedAt)
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: "trashFileMetadata",
		},
		GapY: true,
		Children: &wx.List{
			Children: items,
		},
		MarginY: true,
	}
}
