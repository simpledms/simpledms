package inbox

// package action

import (
	"github.com/simpledms/simpledms/action/browse"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
)

type FileMetadataPartialData struct {
	FileID         int64
	DocumentTypeID int64
}

type FileMetadataPartial struct {
	*browse.FileAttributesPartial
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileMetadataPartial(infra *common.Infra, actions *Actions) *FileMetadataPartial {
	config := actionx.NewConfig(
		actions.Route("file-metadata"),
		false,
	)
	return &FileMetadataPartial{
		FileAttributesPartial: actions.Browse.FileAttributesPartial,
		infra:                 infra,
		actions:               actions,
		Config:                config,
	}
}

func (qq *FileMetadataPartial) Widget(
	ctx ctxx.Context,
	data *browse.FileAttributesPartialData,
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

	// TODO above or below FileAttributes? Must remove MarginY
	// 		on scrollable content if below
	children = append(children,
		&wx.Button{
			Label:     wx.T("Mark as done"),
			StyleType: wx.ButtonStyleTypeElevated,
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.MarkAsDoneCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.MarkAsDoneCmd.Data(data.FileID)),
				HxHeaders: autil.QueryHeader(
					qq.actions.InboxPage.Endpoint(),
					qq.actions.InboxPage.Data(),
				),
			},
		},
	)

	children = append(children, qq.FileAttributesPartial.Widget(ctx, data))

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.MetadataTabContentID(),
		},
		// GapY:     true,
		Children: children,
		MarginY:  true,
		FlexCol:  true,
	}
}

func (qq *FileMetadataPartial) MetadataTabContentID() string {
	return "metadataTabContent"
}
