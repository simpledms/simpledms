package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type AttributeContextMenuWidgetData struct {
	AttributeID int64
}

type AttributeContextMenuWidget struct {
	actions *Actions
}

func NewAttributeContextMenuWidget(actions *Actions) *AttributeContextMenuWidget {
	return &AttributeContextMenuWidget{
		actions: actions,
	}
}

func (qq *AttributeContextMenuWidget) Widget(ctx ctxx.Context, attributex *enttenant.Attribute) *wx.Menu {
	// TODO change tag group

	var items []*wx.MenuItem

	if attributex.Type == attributetype.Tag {
		// properties have no name...
		items = append(items, &wx.MenuItem{
			TrailingIcon: "edit",
			Label:        wx.T("Edit"),
			HTMXAttrs: qq.actions.EditTagAttributeCmd.ModalLinkAttrs(
				qq.actions.EditTagAttributeCmd.Data(attributex.ID, attributex.Name, attributex.IsNameGiving),
				"",
				// "#"+qq.actions.ListDir.WrapperID(),
			),
		}, &wx.MenuItem{
			IsDivider: true,
		})
	} else if attributex.Type == attributetype.Field {
		items = append(items, &wx.MenuItem{
			TrailingIcon: "edit",
			Label:        wx.T("Edit"),
			HTMXAttrs: qq.actions.EditPropertyAttributeCmd.ModalLinkAttrs(
				qq.actions.EditPropertyAttributeCmd.Data(attributex.ID, attributex.IsNameGiving),
				"",
				// "#"+qq.actions.ListDir.WrapperID(),
			),
		}, &wx.MenuItem{
			IsDivider: true,
		})

	}

	items = append(items, &wx.MenuItem{
		TrailingIcon: "delete",
		Label:        wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.DeleteAttributeCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteAttributeCmd.Data(attributex.ID)),
			HxConfirm: wx.T("Are you sure?").String(ctx),
			HxSwap:    "none",
			// HxTarget:  "#" + qq.actions.ListDir.WrapperID(),
		},
	})

	return &wx.Menu{
		Items: items,
	}
}
