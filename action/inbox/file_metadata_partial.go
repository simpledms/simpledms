package inbox

// package action

import (
	"log"

	"github.com/simpledms/simpledms/action/browse"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
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
	rw.AddRenderables(wx.NewSnackbarf("Reloaded metadata"))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileMetadataPartial) Widget(
	ctx ctxx.Context,
	data *FileMetadataPartialData,
) *wx.ScrollableContent {
	// TODO datum; as special field or value tag?
	// 		value tag allows the user to define multiple date types (Eingangsdatum, Erstellungsdatum, etc.)

	// TODO just Title instead of filename? autofilename based on attributes?
	//		value Tag or special attribute?

	// which name to show?
	// virtual filename composed of value tags?
	// !! no filename at all? tags should describe what normally is in filename
	// can be created on demand from tags (user defines pattern) can also use title tag...

	// how to sort files in browse if no primary filename? value tag?

	var children []wx.IWidget
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

	var nilableBottomAppBar *wx.BottomAppBar
	if len(statusMessages) > 0 {
		nilableBottomAppBar = &wx.BottomAppBar{
			Actions: []wx.IWidget{
				&wx.IconButton{
					Icon:    "refresh",
					Tooltip: wx.T("Reload metadata"),
					HTMXAttrs: wx.HTMXAttrs{
						HxPost:   qq.Endpoint(),
						HxVals:   util.JSON(data),
						HxTarget: "#" + qq.MetadataTabContentID(),
						HxSwap:   "outerHTML",
					},
				},
			},
			Children: qq.statusMessageWidget(statusMessages),
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
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

func (qq *FileMetadataPartial) deleteFromInboxButton(ctx ctxx.Context, fileID string) *wx.Button {
	return &wx.Button{
		Label:     wx.T("Delete from inbox"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.Browse.DeleteFileCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.Browse.DeleteFileCmd.Data(fileID)),
			HxConfirm: wx.T("Are you sure?").String(ctx),
		},
	}
}

func (qq *FileMetadataPartial) markAsDoneButton(fileID string) *wx.Button {
	return &wx.Button{
		Label:     wx.T("Mark as done"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.actions.MarkAsDoneCmd.Endpoint(),
			HxVals: util.JSON(qq.actions.MarkAsDoneCmd.Data(fileID)),
			HxHeaders: autil.QueryHeader(
				qq.actions.InboxPage.Endpoint(),
				qq.actions.InboxPage.Data(),
			),
		},
	}
}

func (qq *FileMetadataPartial) nilableOCRStatusMessage(hasOCRSuccess bool, fileSize int64) *wx.Text {
	if hasOCRSuccess {
		return nil
	}

	if ocrutil.IsFileTooLarge(fileSize) {
		return wx.T("Text recognition (OCR) cannot be applied because the file is too large, suggestions are based on the filename only.")
	}

	return wx.T("Text recognition (OCR) is not ready yet, suggestions are based on the filename only.")
}

func (qq *FileMetadataPartial) statusMessages(
	ctx ctxx.Context,
	fileID string,
	hasOCRSuccess bool,
	fileSize int64,
	duplicateStatusMessage *wx.Text,
	hasDuplicates bool,
) []wx.IWidget {
	var messages []wx.IWidget
	if ocrStatusMessage := qq.nilableOCRStatusMessage(hasOCRSuccess, fileSize); ocrStatusMessage != nil {
		messages = append(messages, wx.NewBody(wx.BodyTypeSm, ocrStatusMessage))
	}
	if duplicateStatusMessage != nil {
		messages = append(messages, wx.NewBody(wx.BodyTypeSm, duplicateStatusMessage))
	}
	if hasDuplicates {
		messages = append(messages, qq.duplicatesFoundLink(ctx, fileID))
	}

	return messages
}

func (qq *FileMetadataPartial) statusMessageWidget(messages []wx.IWidget) wx.IWidget {
	if len(messages) == 1 {
		return messages[0]
	}

	return &wx.Column{
		GapYSize:   wx.Gap1,
		AutoHeight: true,
		Children:   messages,
	}
}

func (qq *FileMetadataPartial) duplicatesFoundLink(ctx ctxx.Context, fileID string) *wx.Link {
	linkText := wx.T("Duplicates found").SetWrap()
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

	return &wx.Link{
		Href:  targetURL,
		Child: linkText,
		HTMXAttrs: wx.HTMXAttrs{
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
