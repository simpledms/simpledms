package trash

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type TrashContextMenuWidget struct {
	actions *Actions
}

func NewTrashContextMenuWidget(actions *Actions) *TrashContextMenuWidget {
	return &TrashContextMenuWidget{
		actions: actions,
	}
}

func (qq *TrashContextMenuWidget) Widget(ctx ctxx.Context, filex *enttenant.File) *wx.Menu {
	var items []*wx.MenuItem

	items = append(items, &wx.MenuItem{
		TrailingIcon: "restore_from_trash",
		Label:        wx.T("Restore"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.actions.RestoreFileCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.RestoreFileCmd.DataWithOptions(filex.PublicID.String())),
			HxConfirm: wx.T("Are you sure?").String(ctx),
		},
	})

	return &wx.Menu{
		Items: items,
	}
}
