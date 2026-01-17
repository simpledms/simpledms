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

type FileMetadataData struct {
	FileID         int64
	DocumentTypeID int64
}

type FileMetadata struct {
	*browse.FileAttributes
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileMetadata(infra *common.Infra, actions *Actions) *FileMetadata {
	config := actionx.NewConfig(
		actions.Route("file-metadata"),
		false,
	)
	return &FileMetadata{
		FileAttributes: actions.Browse.FileAttributes,
		infra:          infra,
		actions:        actions,
		Config:         config,
	}
}

func (qq *FileMetadata) Widget(
	ctx ctxx.Context,
	data *browse.FileAttributesData,
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
				HxPost: qq.actions.MarkAsDone.Endpoint(),
				HxVals: util.JSON(qq.actions.MarkAsDone.Data(data.FileID)),
				HxHeaders: autil.QueryHeader(
					qq.actions.Page.Endpoint(),
					qq.actions.Page.Data(),
				),
			},
		},
	)

	children = append(children, qq.FileAttributes.Widget(ctx, data))

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

func (qq *FileMetadata) MetadataTabContentID() string {
	return "metadataTabContent"
}
