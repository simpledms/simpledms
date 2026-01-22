package documenttype

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type ContextMenuPartialData struct {
	DocumentTypeID int64
}

// TODO name?
type ContextMenuPartial struct {
	actions *Actions
}

func NewContextMenuPartial(actions *Actions) *ContextMenuPartial {
	return &ContextMenuPartial{
		actions: actions,
	}
}

func (qq *ContextMenuPartial) Widget(ctx ctxx.Context, documentType *enttenant.DocumentType) *wx.Menu {
	renameItem := &wx.MenuItem{
		TrailingIcon: "edit", // TODO
		Label:        wx.T("RenameCmd"),
		HTMXAttrs: qq.actions.RenameCmd.ModalLinkAttrs(
			qq.actions.RenameCmd.Data(documentType.ID, documentType.Name),
			"",
			// "#"+qq.actions.ListDir.WrapperID(),
		),
	}

	deleteItem := &wx.MenuItem{
		TrailingIcon: "delete",
		Label:        wx.T("DeleteCmd"),
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
