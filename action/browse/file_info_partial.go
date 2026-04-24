package browse

import (
	"fmt"
	"log"

	"entgo.io/ent/dialect/sql"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
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
	filem := repos.Read.FileByPublicIDX(ctx, data.FileID)
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
		{
			Headline:       wx.T("Uploaded at"),
			SupportingText: wx.Tu(timex.NewDateTime(filem.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
		},
		{
			Headline:       wx.T("Version"),
			SupportingText: qq.versionLabel(ctx, filem.ID),
		},
		{
			Headline:       wx.T("Current version uploaded at"),
			SupportingText: wx.Tu(timex.NewDateTime(currentVersion.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
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

	if !filem.DeletedAt.IsZero() {
		// order is good, because oriented on lifecycle
		items = append(items, []*wx.ListItem{
			{
				Headline:       wx.T("Deleted at"),
				SupportingText: wx.T(timex.NewDateTime(filem.DeletedAt).String(ctx.MainCtx().LanguageBCP47)),
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

func (qq *FileInfoPartial) versionLabel(ctx ctxx.Context, fileID int64) *wx.Text {
	versionData := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileID(fileID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		FirstX(ctx)
	return wx.Tu(fmt.Sprintf("%d", versionData.VersionNumber))
}

func (qq *FileInfoPartial) ID() string {
	return "fileInfo"
}
