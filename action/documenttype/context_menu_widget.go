package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
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

func (qq *ContextMenuWidget) Widget(ctx ctxx.Context, documentType *enttenant.DocumentType) *wx.Menu {
	renameItem := &wx.MenuItem{
		TrailingIcon: "edit", // TODO
		Label:        wx.T("Rename"),
		HTMXAttrs: qq.actions.RenameCmd.ModalLinkAttrs(
			qq.actions.RenameCmd.Data(documentType.ID, documentType.Name),
			"",
			// "#"+qq.actions.ListDir.WrapperID(),
		),
	}

	deleteItem := &wx.MenuItem{
		TrailingIcon: "delete",
		Label:        wx.T("Delete"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.DeleteCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.DeleteCmd.Data(documentType.ID)),
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
