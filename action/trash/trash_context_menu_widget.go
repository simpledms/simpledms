package trash

import (
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
)

type TrashContextMenuWidget struct {
	actions *Actions
}

func NewTrashContextMenuWidget(actions *Actions) *TrashContextMenuWidget {
	return &TrashContextMenuWidget{
		actions: actions,
	}
}

func (qq *TrashContextMenuWidget) Widget(ctx ctxx.Context, filex *enttenant.File) *widget.Menu {
	var items []*widget.MenuItem

	items = append(items, &widget.MenuItem{
		TrailingIcon: "restore_from_trash",
		Label:        widget.T("Restore"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.actions.RestoreFileCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.RestoreFileCmd.DataWithOptions(filex.PublicID.String())),
			HxConfirm: widget.T("Are you sure?").String(ctx),
		},
	})

	return &widget.Menu{
		Items: items,
	}
}
