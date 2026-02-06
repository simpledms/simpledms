package inbox

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type FileContextMenuWidget struct {
	actions *Actions
}

func NewFileContextMenuWidget(actions *Actions) *FileContextMenuWidget {
	return &FileContextMenuWidget{
		actions: actions,
	}
}

func (qq *FileContextMenuWidget) Widget(ctx ctxx.Context, filex *enttenant.File) *wx.Menu {
	// filem := model.NewFile(filex)
	var menuItems []*wx.MenuItem

	menuItems = append(menuItems,
		&wx.MenuItem{
			TrailingIcon: "edit", // TODO
			Label:        wx.T("Rename"),
			HTMXAttrs: qq.actions.Browse.RenameFileCmd.ModalLinkAttrs(
				qq.actions.Browse.RenameFileCmd.Data(filex.PublicID.String(), filex.Name),
				"",
				// "#"+qq.actions.ListDir.WrapperID(),
			),
		},
	)

	menuItems = append(menuItems,
		&wx.MenuItem{
			IsDivider: true,
		},
		&wx.MenuItem{
			TrailingIcon: "delete",
			Label:        wx.T("Delete"),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.Browse.DeleteFileCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.Browse.DeleteFileCmd.Data(filex.PublicID.String())),
				// HxTarget:  "#" + qq.actions.ListDir.WrapperID(),
				HxConfirm: wx.T("Are you sure?").String(ctx),
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

	return &wx.Menu{
		Items: menuItems,
	}
}
