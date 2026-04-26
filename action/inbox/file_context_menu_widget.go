package inbox

import (
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
)

type FileContextMenuWidget struct {
	actions *Actions
}

func NewFileContextMenuWidget(actions *Actions) *FileContextMenuWidget {
	return &FileContextMenuWidget{
		actions: actions,
	}
}

func (qq *FileContextMenuWidget) Widget(ctx ctxx.Context, filex *enttenant.File) *widget.Menu {
	// filem := filemodel.NewFile(filex)
	var menuItems []*widget.MenuItem

	menuItems = append(menuItems,
		&widget.MenuItem{
			TrailingIcon: "edit", // TODO
			Label:        widget.T("Rename"),
			HTMXAttrs: qq.actions.Browse.RenameFileCmd.ModalLinkAttrs(
				qq.actions.Browse.RenameFileCmd.Data(filex.PublicID.String(), filex.Name),
				"",
				// "#"+qq.actions.ListDir.WrapperID(),
			),
		},
	)

	menuItems = append(menuItems,
		&widget.MenuItem{
			IsDivider: true,
		},
		&widget.MenuItem{
			TrailingIcon: "delete",
			Label:        widget.T("Delete"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost: qq.actions.Browse.DeleteFileCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.Browse.DeleteFileCmd.Data(filex.PublicID.String())),
				// HxTarget:  "#" + qq.actions.ListDir.WrapperID(),
				HxConfirm: widget.T("Are you sure?").String(ctx),
			},
		},
	)

	/* TODO support for inbox must be implemented in ArchiveCmd
	if filem.IsZIPArchive(ctx) {
		menuItems = append(menuItems, &wx.MenuItem{
			TrailingIcon: "Unarchive",
			Label:        wx.T("Unzip archive"),
			HTMXAttrs: qq.actions.UnzipArchiveCmd.ModalLinkAttrs(
				qq.actions.UnzipArchiveCmd.Data(filem.Data.PublicID.String(), false),
				"",
			),
		})
	}
	*/

	return &widget.Menu{
		Items: menuItems,
	}
}
