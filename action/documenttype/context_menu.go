package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type ContextMenuData struct {
	DocumentTypeID int64
}

// TODO name?
type ContextMenu struct {
	actions *Actions
}

func NewContextMenu(actions *Actions) *ContextMenu {
	return &ContextMenu{
		actions: actions,
	}
}

func (qq *ContextMenu) Widget(ctx ctxx.Context, documentType *enttenant.DocumentType) *wx.Menu {
	renameItem := &wx.MenuItem{
		TrailingIcon: "edit", // TODO
		Label:        wx.T("Rename"),
		HTMXAttrs: qq.actions.Rename.ModalLinkAttrs(
			qq.actions.Rename.Data(documentType.ID, documentType.Name),
			"",
			// "#"+qq.actions.ListDir.WrapperID(),
		),
	}

	deleteItem := &wx.MenuItem{
		TrailingIcon: "delete",
		Label:        wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.Delete.Endpoint(),
			HxVals:    util.JSON(qq.actions.Delete.Data(documentType.ID)),
			HxConfirm: wx.T("Are you sure?").String(ctx),
			HxSwap:    "none",
			// HxTarget:  "#" + qq.actions.ListDir.WrapperID(),
		},
	}

	return &wx.Menu{
		Items: []*wx.MenuItem{
			renameItem,
			{
				IsDivider: true,
			},
			deleteItem,
		},
	}
}
