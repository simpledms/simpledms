package browse

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/core/util/timex"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/ui/uix/event"
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

func (qq *FileVersionsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

func (qq *FileVersionsPartial) Widget(ctx ctxx.Context, data *FileVersionsPartialData) renderable.Renderable {
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	versions := filex.Data.QueryFileVersions().
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		WithStoredFile().
		AllX(ctx)

	var listItems []*widget.ListItem
	if len(versions) == 0 {
		// should never happen in current system where a file always has one underlying
		// stored file
		listItems = append(listItems, &widget.ListItem{
			Headline:       widget.T("No versions available yet."),
			SupportingText: widget.T("Upload a new version to get started."),
			Type:           widget.ListItemTypeHelper,
		})
	} else {
		for _, versionx := range versions {
			storedFile := versionx.Edges.StoredFile
			versionm := storedfilemodel.NewStoredFile(storedFile)
			versionLabel := fmt.Sprintf("Version %d", versionx.VersionNumber)

			var supportingParts []string
			supportingParts = append(supportingParts, versionLabel)
			supportingParts = append(supportingParts, versionm.SizeString())
			if versionm.Data.MimeType != "" {
				supportingParts = append(supportingParts, versionm.Data.MimeType)
			}

			listItem := &widget.ListItem{
				Headline:       widget.Tu(timex.NewDateTime(versionm.Data.CreatedAt).String(ctx.MainCtx().LanguageBCP47)),
				SupportingText: widget.Tu(strings.Join(supportingParts, " - ")),
			}
			listItem.HTMXAttrs = widget.HTMXAttrs{
				HxPost:        qq.actions.FileVersionPreviewDialogPartial.Endpoint(),
				HxVals:        util.JSON(qq.actions.FileVersionPreviewDialogPartial.Data(data.FileID, fmt.Sprintf("%d", versionx.VersionNumber))),
				LoadInPopover: true,
			}

			listItems = append(listItems, listItem)
		}
	}

	return &widget.Column{
		Widget: widget.Widget[widget.Column]{
			ID: qq.ID(),
		},
		GapYSize: widget.Gap4,
		MarginY:  widget.Margin4,
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: event.FileUploaded.Handler(),
			HxPost:    qq.Endpoint(),
			HxVals:    util.JSON(qq.Data(data.FileID)),
			HxTarget:  "#" + qq.ID(),
			HxSwap:    "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.Column{
				AutoHeight: true,
				GapYSize:   widget.Gap2,
				// necessary that column doesn't get shrunk when available space is tight
				// (version lists grows)
				NoOverflowHidden: true,
				Children: []widget.IWidget{
					&widget.Button{
						Icon:      widget.NewIcon("upload_file"),
						Label:     widget.T("Add new version"),
						StyleType: widget.ButtonStyleTypeElevated,
						HTMXAttrs: widget.HTMXAttrs{
							HxPost:        qq.actions.FileVersionUploadDialogPartial.Endpoint(),
							HxVals:        util.JSON(qq.actions.FileVersionUploadDialogPartial.Data(data.FileID)),
							LoadInPopover: true,
						},
					},
					&widget.Button{
						Icon:      widget.NewIcon("merge"),
						Label:     widget.T("Add new version from inbox"),
						StyleType: widget.ButtonStyleTypeElevated,
						HTMXAttrs: widget.HTMXAttrs{
							HxPost:        qq.actions.FileVersionFromInboxDialog.Endpoint(),
							HxVals:        util.JSON(qq.actions.FileVersionFromInboxDialog.Data(data.FileID, "", "")),
							LoadInPopover: true,
						},
					},
				},
			},
			&widget.ScrollableContent{
				Children: &widget.List{
					Children: listItems,
				},
			},
		},
	}
}

func (qq *FileVersionsPartial) ID() string {
	return "fileVersions"
}
