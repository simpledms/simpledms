package browse

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type FileContextMenuPartial struct {
	actions *Actions
}

func NewFileContextMenuPartial(actions *Actions) *FileContextMenuPartial {
	return &FileContextMenuPartial{
		actions: actions,
	}
}

func (qq *FileContextMenuPartial) Widget(ctx ctxx.Context, filex *enttenant.File) *wx.Menu {
	filem := model.NewFile(filex)
	var menuItems []*wx.MenuItem

	// TODO `select` menu item for multiselection?

	menuItems = append(menuItems,
		&wx.MenuItem{
			TrailingIcon: "edit", // TODO
			Label:        wx.T("Rename"),
			HTMXAttrs: qq.actions.RenameFileCmd.ModalLinkAttrs(
				qq.actions.RenameFileCmd.Data(filex.PublicID.String(), filex.Name),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
		},
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		menuItems = append(menuItems,
			&wx.MenuItem{
				TrailingIcon: "drive_file_move",
				Label:        wx.T("Move"),
				HTMXAttrs: qq.actions.MoveFileCmd.ModalLinkAttrs(
					qq.actions.MoveFileCmd.Data(filex.PublicID.String(), ""),
					"#"+qq.actions.ListDirPartial.WrapperID(),
				),
			},
		)
	}

	menuItems = append(menuItems,
		&wx.MenuItem{
			IsDivider: true,
		},
		&wx.MenuItem{
			TrailingIcon: "delete",
			Label:        wx.T("Delete"),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:    qq.actions.DeleteFile.Endpoint(),
				HxVals:    util.JSON(qq.actions.DeleteFile.Data(filex.PublicID.String())),
				HxTarget:  "#" + qq.actions.ListDirPartial.WrapperID(),
				HxConfirm: wx.T("Are you sure?").String(ctx),
			},
		},
	)

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

	return &wx.Menu{
		Items: menuItems,
	}
}
