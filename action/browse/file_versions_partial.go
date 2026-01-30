package browse

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type FileVersionsPartialData struct {
	FileID string
}

type FileVersionsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersionsPartial(infra *common.Infra, actions *Actions) *FileVersionsPartial {
	config := actionx.NewConfig(
		actions.Route("file-versions-partial"),
		true,
	)
	return &FileVersionsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionsPartial) Data(fileID string) *FileVersionsPartialData {
	return &FileVersionsPartialData{
		FileID: fileID,
	}
}

func (qq *FileVersionsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileVersionsPartial) Widget(ctx ctxx.Context, data *FileVersionsPartialData) *wx.ScrollableContent {
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	versions := filex.Data.QueryVersions().Order(storedfile.ByCreatedAt(sql.OrderDesc())).AllX(ctx)

	var listItems []*wx.ListItem
	if len(versions) == 0 {
		// should never happen in current system where a file always has one underlying
		// stored file
		listItems = append(listItems, &wx.ListItem{
			Headline:       wx.T("No versions available yet."),
			SupportingText: wx.T("Upload a new version to get started."),
			Type:           wx.ListItemTypeHelper,
		})
	} else {
		for i, versionx := range versions {
			versionm := model.NewStoredFile(versionx)
			versionLabel := fmt.Sprintf("Version %d", len(versions)-i)

			var supportingParts []string
			supportingParts = append(supportingParts, versionLabel)
			supportingParts = append(supportingParts, versionm.SizeString())
			if versionx.MimeType != "" {
				supportingParts = append(supportingParts, versionx.MimeType)
			}

			listItem := &wx.ListItem{
				Headline:       wx.Tu(timex.NewDateTime(versionx.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
				SupportingText: wx.Tu(strings.Join(supportingParts, " - ")),
			}
			listItem.HTMXAttrs = wx.HTMXAttrs{
				HxPost:        qq.actions.FileVersionPreviewDialogPartial.Endpoint(),
				HxVals:        util.JSON(qq.actions.FileVersionPreviewDialogPartial.Data(data.FileID, fmt.Sprintf("%d", versionx.ID))),
				LoadInPopover: true,
			}

			listItems = append(listItems, listItem)
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.ID(),
		},
		GapY: true,
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.FileUploaded.Handler(),
			HxPost:    qq.Endpoint(),
			HxVals:    util.JSON(qq.Data(data.FileID)),
			HxTarget:  "#" + qq.ID(),
			HxSwap:    "outerHTML",
		},
		Children: []wx.IWidget{
			&wx.Column{
				AutoHeight: true,
				Children: &wx.Button{
					Icon:      wx.NewIcon("upload_file"),
					Label:     wx.T("Add new version"),
					StyleType: wx.ButtonStyleTypeElevated,
					HTMXAttrs: wx.HTMXAttrs{
						HxPost:        qq.actions.FileVersionUploadDialogPartial.Endpoint(),
						HxVals:        util.JSON(qq.actions.FileVersionUploadDialogPartial.Data(data.FileID)),
						LoadInPopover: true,
					},
				},
			},
			&wx.List{
				Children: listItems,
			},
		},
		MarginY: true,
	}
}

func (qq *FileVersionsPartial) ID() string {
	return "fileVersions"
}
