package inbox

// package action

import (
	"github.com/simpledms/simpledms/action/browse"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/core/util/ocrutil"
	"github.com/simpledms/simpledms/ctxx"
)

type FileMetadataPartialData struct {
	*browse.FileAttributesPartialData
}

type FileMetadataPartial struct {
	// *browse.FileAttributesPartial // to error prone to embed this
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileMetadataPartial(infra *common.Infra, actions *Actions) *FileMetadataPartial {
	config := actionx.NewConfig(
		actions.Route("file-metadata-partial"),
		true,
	)
	return &FileMetadataPartial{
		// FileAttributesPartial: actions.Browse.FileAttributesPartial,
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileMetadataPartial) Data(fileID string) *FileMetadataPartialData {
	return &FileMetadataPartialData{
		FileAttributesPartialData: qq.actions.Browse.FileAttributesPartial.Data(fileID),
	}
}

func (qq *FileMetadataPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileMetadataPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	// TODO is there a way to implement this conditional, only when reload
	//  	button is used? May not be relevant in all cases
	rw.AddRenderables(widget.NewSnackbarf("Reloaded metadata"))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileMetadataPartial) Widget(
	ctx ctxx.Context,
	data *FileMetadataPartialData,
) *widget.ScrollableContent {
	// TODO datum; as special field or value tag?
	// 		value tag allows the user to define multiple date types (Eingangsdatum, Erstellungsdatum, etc.)

	// TODO just Title instead of filename? autofilename based on attributes?
	//		value Tag or special attribute?

	// which name to show?
	// virtual filename composed of value tags?
	// !! no filename at all? tags should describe what normally is in filename
	// can be created on demand from tags (user defines pattern) can also use title tag...

	// how to sort files in browse if no primary filename? value tag?

	var children []widget.IWidget

	// TODO above or below FileAttributes? Must remove MarginY
	// 		on scrollable content if below
	children = append(children,
		&widget.Button{
			Label:     widget.T("Mark as done"),
			StyleType: widget.ButtonStyleTypeElevated,
			HTMXAttrs: widget.HTMXAttrs{
				HxPost: qq.actions.MarkAsDoneCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.MarkAsDoneCmd.Data(data.FileID)),
				HxHeaders: autil.QueryHeader(
					qq.actions.InboxPage.Endpoint(),
					qq.actions.InboxPage.Data(),
				),
			},
		},
	)

	children = append(children, qq.actions.Browse.FileAttributesPartial.Widget(
		ctx,
		data.FileAttributesPartialData,
	))

	// TODO also loaded in qq.actions.Browse.FileAttributesPartial.Widget
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	var nilableBottomAppBar *widget.BottomAppBar

	if message := qq.nilableOCRStatusMessage(filex.HasOCRSuccess(ctx), filex.Size(ctx)); message != nil {
		nilableBottomAppBar = &widget.BottomAppBar{
			Actions: []widget.IWidget{
				&widget.IconButton{
					Icon:    "refresh",
					Tooltip: widget.T("Reload metadata"),
					HTMXAttrs: widget.HTMXAttrs{
						HxPost:   qq.Endpoint(),
						HxVals:   util.JSON(data),
						HxTarget: "#" + qq.MetadataTabContentID(),
						HxSwap:   "outerHTML",
					},
				},
			},
			Children: widget.NewBody(
				widget.BodyTypeSm,
				message,
			),
		}
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.MetadataTabContentID(),
		},
		// GapY:     true,
		Children:     children,
		MarginY:      true,
		FlexCol:      true,
		BottomAppBar: nilableBottomAppBar,
	}
}

func (qq *FileMetadataPartial) MetadataTabContentID() string {
	return "metadataTabContent"
}

func (qq *FileMetadataPartial) nilableOCRStatusMessage(hasOCRSuccess bool, fileSize int64) *widget.Text {
	if hasOCRSuccess {
		return nil
	}

	if ocrutil.IsFileTooLarge(fileSize) {
		return widget.T("Text recognition (OCR) cannot be applied because the file is too large, suggestions are based on the filename only.")
	}

	return widget.T("Text recognition (OCR) is not ready yet, suggestions are based on the filename only.")
}
