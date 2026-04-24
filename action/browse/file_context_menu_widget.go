package browse

import (
	"log"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type FileContextMenuWidget struct {
	infra   *common.Infra
	actions *Actions
}

func NewFileContextMenuWidget(infra *common.Infra, actions *Actions) *FileContextMenuWidget {
	return &FileContextMenuWidget{
		infra:   infra,
		actions: actions,
	}
}

func (qq *FileContextMenuWidget) Widget(
	ctx ctxx.Context,
	filePublicID string,
	fileName string,
	fileID int64,
	isDirectory bool,
) *wx.Menu {
	var menuItems []*wx.MenuItem

	// TODO `select` menu item for multiselection?

	menuItems = append(menuItems,
		&wx.MenuItem{
			TrailingIcon: "edit", // TODO
			Label:        wx.T("Rename"),
			HTMXAttrs: qq.actions.RenameFileCmd.ModalLinkAttrs(
				qq.actions.RenameFileCmd.Data(filePublicID, fileName),
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
					qq.actions.MoveFileCmd.Data(filePublicID, ""),
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
				HxPost:    qq.actions.DeleteFileCmd.Endpoint(),
				HxVals:    util.JSON(qq.actions.DeleteFileCmd.Data(filePublicID)),
				HxTarget:  "#" + qq.actions.ListDirPartial.WrapperID(),
				HxConfirm: wx.T("Are you sure?").String(ctx),
			},
		},
	)

	if !isDirectory {
		currentVersion, err := qq.infra.FileSystem().CurrentVersionByFileIDX(ctx, fileID)
		if err != nil {
			log.Println(err)
		} else if currentVersion.IsZIPArchive() {
			menuItems = append(menuItems, &wx.MenuItem{
				TrailingIcon: "Unarchive",
				Label:        wx.T("Unzip archive"),
				HTMXAttrs: qq.actions.UnzipArchiveCmd.ModalLinkAttrs(
					qq.actions.UnzipArchiveCmd.Data(filePublicID, false),
					"",
				),
			})
		}
	}

	return &wx.Menu{
		Items: menuItems,
	}
}
