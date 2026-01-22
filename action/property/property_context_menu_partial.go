package property

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type PropertyContextMenuPartial struct {
	actions *Actions
}

func NewPropertyContextMenuPartial(actions *Actions) *PropertyContextMenuPartial {
	return &PropertyContextMenuPartial{
		actions: actions,
	}
}

func (qq *PropertyContextMenuPartial) Widget(ctx ctxx.Context, propertyx *enttenant.Property) *wx.Menu {
	renameItem := &wx.MenuItem{
		TrailingIcon: "edit",
		Label:        wx.T("Edit"),
		HTMXAttrs: qq.actions.EditPropertyCmd.ModalLinkAttrs(
			qq.actions.EditPropertyCmd.Data(
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
			HxPost:    qq.actions.DeletePropertyCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeletePropertyCmd.Data(propertyx.ID)),
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
