package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/tenant/common/attributetype"
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

func (qq *AttributeContextMenuWidget) Widget(ctx ctxx.Context, attributex *enttenant.Attribute) *widget.Menu {
	// TODO change tag group

	var items []*widget.MenuItem

	if attributex.Type == attributetype.Tag {
		// properties have no name...
		items = append(items, &widget.MenuItem{
			TrailingIcon: "edit",
			Label:        widget.T("Edit"),
			HTMXAttrs: qq.actions.EditTagAttributeCmd.ModalLinkAttrs(
				qq.actions.EditTagAttributeCmd.Data(attributex.ID, attributex.Name, attributex.IsNameGiving),
				"",
				// "#"+qq.actions.ListDir.WrapperID(),
			),
		}, &widget.MenuItem{
			IsDivider: true,
		})
	} else if attributex.Type == attributetype.Field {
		items = append(items, &widget.MenuItem{
			TrailingIcon: "edit",
			Label:        widget.T("Edit"),
			HTMXAttrs: qq.actions.EditPropertyAttributeCmd.ModalLinkAttrs(
				qq.actions.EditPropertyAttributeCmd.Data(attributex.ID, attributex.IsNameGiving),
				"",
				// "#"+qq.actions.ListDir.WrapperID(),
			),
		}, &widget.MenuItem{
			IsDivider: true,
		})

	}

	items = append(items, &widget.MenuItem{
		TrailingIcon: "delete",
		Label:        widget.T("Delete"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.DeleteAttributeCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteAttributeCmd.Data(attributex.ID)),
			HxConfirm: widget.T("Are you sure?").String(ctx),
			HxSwap:    "none",
			// HxTarget:  "#" + qq.actions.ListDir.WrapperID(),
		},
	})

	return &widget.Menu{
		Items: items,
	}
}
