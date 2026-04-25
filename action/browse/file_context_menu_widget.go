package browse

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
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
			TrailingIcon: "edit", // TODO
			Label:        widget.T("Rename"),
			HTMXAttrs: qq.actions.RenameFileCmd.ModalLinkAttrs(
				qq.actions.RenameFileCmd.Data(filex.PublicID.String(), filex.Name),
				"#"+qq.actions.ListDirPartial.WrapperID(),
			),
		},
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		menuItems = append(menuItems,
			&widget.MenuItem{
				TrailingIcon: "drive_file_move",
				Label:        widget.T("Move"),
				HTMXAttrs: qq.actions.MoveFileCmd.ModalLinkAttrs(
					qq.actions.MoveFileCmd.Data(filex.PublicID.String(), ""),
					"#"+qq.actions.ListDirPartial.WrapperID(),
				),
			},
		)
	}

	menuItems = append(menuItems,
		&widget.MenuItem{
			IsDivider: true,
		},
		&widget.MenuItem{
			TrailingIcon: "delete",
			Label:        widget.T("Delete"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.actions.DeleteFileCmd.Endpoint(),
				HxVals:    util.JSON(qq.actions.DeleteFileCmd.Data(filex.PublicID.String())),
				HxTarget:  "#" + qq.actions.ListDirPartial.WrapperID(),
				HxConfirm: widget.T("Are you sure?").String(ctx),
			},
		},
	)

	if filem.IsZIPArchive(ctx) {
		menuItems = append(menuItems, &widget.MenuItem{
			TrailingIcon: "Unarchive",
			Label:        widget.T("Unzip archive"),
			HTMXAttrs: qq.actions.UnzipArchiveCmd.ModalLinkAttrs(
				qq.actions.UnzipArchiveCmd.Data(filem.Data.PublicID.String(), false),
				"",
			),
		})
	}

	return &widget.Menu{
		Items: menuItems,
	}
}
