package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
)

type ContextMenuWidgetData struct {
	DocumentTypeID int64
}

// TODO name?
type ContextMenuWidget struct {
	actions *Actions
}

func NewContextMenuWidget(actions *Actions) *ContextMenuWidget {
	return &ContextMenuWidget{
		actions: actions,
	}
}

func (qq *ContextMenuWidget) Widget(ctx ctxx.Context, documentType *enttenant.DocumentType) *widget.Menu {
	renameItem := &widget.MenuItem{
		TrailingIcon: "edit", // TODO
		Label:        widget.T("Rename"),
		HTMXAttrs: qq.actions.RenameCmd.ModalLinkAttrs(
			qq.actions.RenameCmd.Data(documentType.ID, documentType.Name),
			"",
			// "#"+qq.actions.ListDir.WrapperID(),
		),
	}

	deleteItem := &widget.MenuItem{
		TrailingIcon: "delete",
		Label:        widget.T("Delete"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.DeleteCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteCmd.Data(documentType.ID)),
			HxConfirm: widget.T("Are you sure?").String(ctx),
			HxSwap:    "none",
			// HxTarget:  "#" + qq.actions.ListDir.WrapperID(),
		},
	}

	return &widget.Menu{
		Items: []*widget.MenuItem{
			renameItem,
			{
				IsDivider: true,
			},
			deleteItem,
		},
	}
}
