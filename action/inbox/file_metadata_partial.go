package inbox

// package action

import (
	"log"

	"github.com/simpledms/simpledms/action/browse"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/ocrutil"
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

func (qq *FileMetadataPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
	_, duplicateStatusMessage, hasDuplicates, err := qq.actions.Browse.DuplicateMatchesPartial.WidgetWithStatus(
		ctx,
		qq.actions.Browse.DuplicateMatchesPartial.Data(data.FileID),
	)
	if err != nil {
		log.Println(err)
	}
	// TODO above or below FileAttributes? Must remove MarginY
	// 		on scrollable content if below
	children = append(children, qq.markAsDoneButton(data.FileID))

	children = append(children, qq.actions.Browse.FileAttributesPartial.Widget(
		ctx,
		data.FileAttributesPartialData,
	))

	// TODO also loaded in qq.actions.Browse.FileAttributesPartial.Widget
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	statusMessages := qq.statusMessages(
		ctx,
		data.FileID,
		filex.HasOCRSuccess(ctx),
		filex.Size(ctx),
		duplicateStatusMessage,
		hasDuplicates,
	)

	var nilableToolbar *widget.Toolbar
	if len(statusMessages) > 0 {
		nilableToolbar = widget.NewToolbar(
			widget.T("Reload metadata").String(ctx),
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
			qq.statusMessageWidget(statusMessages),
		)
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.MetadataTabContentID(),
		},
		// GapY:     true,
		Children: children,
		MarginY:  true,
		FlexCol:  true,
		Toolbar:  nilableToolbar,
	}
}

func (qq *FileMetadataPartial) MetadataTabContentID() string {
	return "metadataTabContent"
}

func (qq *FileMetadataPartial) deleteFromInboxButton(ctx ctxx.Context, fileID string) *widget.Button {
	return &widget.Button{
		Label:     widget.T("Delete from inbox"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.Browse.DeleteFileCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.Browse.DeleteFileCmd.Data(fileID)),
			HxConfirm: widget.T("Are you sure?").String(ctx),
		},
	}
}

func (qq *FileMetadataPartial) markAsDoneButton(fileID string) *widget.Button {
	return &widget.Button{
		Label:     widget.T("Mark as done"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.MarkAsDoneCmd.Endpoint(),
			HxVals: util.JSON(qq.actions.MarkAsDoneCmd.Data(fileID)),
			HxHeaders: autil.QueryHeader(
				qq.actions.InboxPage.Endpoint(),
				qq.actions.InboxPage.Data(),
			),
		},
	}
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

func (qq *FileMetadataPartial) statusMessages(
	ctx ctxx.Context,
	fileID string,
	hasOCRSuccess bool,
	fileSize int64,
	duplicateStatusMessage *widget.Text,
	hasDuplicates bool,
) []widget.IWidget {
	var messages []widget.IWidget
	if ocrStatusMessage := qq.nilableOCRStatusMessage(hasOCRSuccess, fileSize); ocrStatusMessage != nil {
		messages = append(messages, widget.NewBody(widget.BodyTypeSm, ocrStatusMessage))
	}
	if duplicateStatusMessage != nil {
		messages = append(messages, widget.NewBody(widget.BodyTypeSm, duplicateStatusMessage))
	}
	if hasDuplicates {
		messages = append(messages, qq.duplicatesFoundLink(ctx, fileID))
	}

	return messages
}

func (qq *FileMetadataPartial) statusMessageWidget(messages []widget.IWidget) widget.IWidget {
	if len(messages) == 1 {
		return messages[0]
	}

	return &widget.Column{
		GapYSize:   widget.Gap1,
		AutoHeight: true,
		Children:   messages,
	}
}

func (qq *FileMetadataPartial) duplicatesFoundLink(ctx ctxx.Context, fileID string) *widget.Link {
	linkText := widget.T("Duplicates found").SetWrap()
	linkText.IsSmall = true
	state := &InboxPageState{
		FilesListPartialState: FilesListPartialState{
			ActiveSideSheet: qq.actions.FilePartial.SideSheetID(),
		},
		FilePartialState: FilePartialState{
			ActiveTab: "duplicates",
		},
	}
	targetURL := route.InboxWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, fileID)

	return &widget.Link{
		Href:  targetURL,
		Child: linkText,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.FileTabsPartial.Endpoint(),
			HxVals: util.JSON(qq.actions.FileTabsPartial.Data(
				fileID,
				"duplicates",
			)),
			HxTarget: "#" + qq.actions.FileTabsPartial.ID(),
			HxSwap:   "outerHTML",
		},
	}
}
