package property

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type PropertyContextMenu struct {
	actions *Actions
}

func NewPropertyContextMenu(actions *Actions) *PropertyContextMenu {
	return &PropertyContextMenu{
		actions: actions,
	}
}

func (qq *PropertyContextMenu) Widget(ctx ctxx.Context, propertyx *enttenant.Property) *wx.Menu {
	renameItem := &wx.MenuItem{
		TrailingIcon: "edit",
		Label:        wx.T("Edit"),
		HTMXAttrs: qq.actions.EditProperty.ModalLinkAttrs(
			qq.actions.EditProperty.Data(
				propertyx.ID,
				propertyx.Name,
				propertyx.Unit,
			),
			"",
		),
	}

	deleteItem := &wx.MenuItem{
		TrailingIcon: "delete",
		Label:        wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.DeleteProperty.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteProperty.Data(propertyx.ID)),
			HxConfirm: wx.T("Are you sure?").String(ctx),
		},
	}

	menuItems := []*wx.MenuItem{
		renameItem,
		{
			IsDivider: true,
		},
		deleteItem,
	}

	return &wx.Menu{
		Items: menuItems,
	}

}
