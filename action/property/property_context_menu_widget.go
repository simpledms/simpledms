package property

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
)

type PropertyContextMenuWidget struct {
	actions *Actions
}

func NewPropertyContextMenuWidget(actions *Actions) *PropertyContextMenuWidget {
	return &PropertyContextMenuWidget{
		actions: actions,
	}
}

func (qq *PropertyContextMenuWidget) Widget(ctx ctxx.Context, propertyx *enttenant.Property) *widget.Menu {
	renameItem := &widget.MenuItem{
		TrailingIcon: "edit",
		Label:        widget.T("Edit"),
		HTMXAttrs: qq.actions.EditPropertyCmd.ModalLinkAttrs(
			qq.actions.EditPropertyCmd.Data(
				propertyx.ID,
				propertyx.Name,
				propertyx.Unit,
			),
			"",
		),
	}

	deleteItem := &widget.MenuItem{
		TrailingIcon: "delete",
		Label:        widget.T("Delete"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.DeletePropertyCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeletePropertyCmd.Data(propertyx.ID)),
			HxConfirm: widget.T("Are you sure?").String(ctx),
		},
	}

	menuItems := []*widget.MenuItem{
		renameItem,
		{
			IsDivider: true,
		},
		deleteItem,
	}

	return &widget.Menu{
		Items: menuItems,
	}

}
