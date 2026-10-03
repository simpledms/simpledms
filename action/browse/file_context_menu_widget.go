package browse

import (
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/util"
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
	filem := filemodel.NewFile(filex)
	var menuItems []*widget.MenuItem

	// TODO `select` menu item for multiselection?

	menuItems = append(menuItems,
		&widget.MenuItem{
			LeadingIcon: "edit",
			Label:       widget.T("Rename"),
			HTMXAttrs: qq.actions.RenameFileCmd.ModalLinkAttrs(
				qq.actions.RenameFileCmd.Data(filex.PublicID.String(), filex.Name),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
		},
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		menuItems = append(menuItems,
			&widget.MenuItem{
				LeadingIcon: "drive_file_move",
				Label:       widget.T("Move"),
				HTMXAttrs: qq.actions.MoveFileCmd.ModalLinkAttrs(
					qq.actions.MoveFileCmd.Data(filex.PublicID.String(), ""),
					"#"+qq.actions.ListDirPartial.WrapperID(),
				),
			},
		)
	}

	if filem.IsZIPArchive(ctx) {
		menuItems = append(menuItems, &widget.MenuItem{
			LeadingIcon: "unarchive",
			Label:       widget.T("Unzip archive"),
			HTMXAttrs: qq.actions.UnzipArchiveCmd.ModalLinkAttrs(
				qq.actions.UnzipArchiveCmd.Data(filem.Data.PublicID.String(), false),
				"",
			),
		})
	}

	deleteConfirm := widget.T("Delete this file? You can restore it from the Trash.")
	if filex.IsDirectory {
		deleteConfirm = widget.T("Delete this folder? Its files can be restored from the Trash.")
	}

	menuItems = append(menuItems,
		&widget.MenuItem{
			IsDivider: true,
		},
		&widget.MenuItem{
			LeadingIcon: "delete",
			Label:       widget.T("Delete"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.actions.DeleteFileCmd.Endpoint(),
				HxVals:    util.JSON(qq.actions.DeleteFileCmd.Data(filex.PublicID.String())),
				HxSwap:    "none",
				HxConfirm: deleteConfirm.String(ctx),
			},
		},
	)

	return &widget.Menu{
		Items: menuItems,
	}
}
