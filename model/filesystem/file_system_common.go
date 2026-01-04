package filesystem

import (
	"log"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileinfo"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
)

type fileSystemCommon struct {
	// storageDirPath string
	metaPath string
}

func NewFileSystemCommon(metaPath string) *fileSystemCommon {
	return &fileSystemCommon{
		metaPath: metaPath,
		// storageDirPath: storageDirPath,
	}
}

// TODO public ID or private ID?
// returns last created dir
func (qq *fileSystemCommon) MakeDirAllIfNotExists(ctx ctxx.Context, currentParentDir *enttenant.File, pathToCreate string) (*enttenant.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	pathToCreate = filepath.Clean(pathToCreate)
	// filepath.SplitList uses filepath.ListSeparator, thus not the same
	// TODO correct that os specific? for unzip it seems so because paths are handled on server
	pathElems := strings.Split(pathToCreate, string(filepath.Separator))

	// currentParentDir := ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicID(entx.NewCIText(parentDirID))).OnlyX(ctx)
	currentParentDirFileInfo := ctx.SpaceCtx().TTx.FileInfo.Query().Where(fileinfo.FileID(currentParentDir.ID)).OnlyX(ctx)

	for _, pathElem := range pathElems {
		fullPath, err := securejoin.SecureJoin(currentParentDirFileInfo.FullPath, pathElem)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if strings.HasPrefix(fullPath, "/") {
			// TODO is this always true and okay?
			fullPath = fullPath[1:]
		}

		var newCurrentDir *model.File
		newCurrentDirFileInfo, err := ctx.SpaceCtx().TTx.FileInfo.Query().Where(fileinfo.FullPath(fullPath)).Only(ctx)
		if err != nil && !enttenant.IsNotFound(err) {
			log.Println(err)
			return nil, err
		}
		if enttenant.IsNotFound(err) {
			newCurrentDir, err = qq.MakeDir(ctx, currentParentDir.PublicID.String(), pathElem)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			newCurrentDirFileInfo = ctx.SpaceCtx().TTx.FileInfo.Query().
				Where(fileinfo.FileID(newCurrentDir.Data.ID)).
				OnlyX(ctx)
		} else {
			// newCurrentDirFileInfo already set
			newCurrentDirx := ctx.SpaceCtx().Space.QueryFiles().Where(file.ID(newCurrentDirFileInfo.FileID)).OnlyX(ctx)
			newCurrentDir = model.NewFile(newCurrentDirx)

			// TODO good? correct location
			if !newCurrentDir.Data.IsDirectory {
				return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Path element is file, not a directory.")
			}
		}

		currentParentDir = newCurrentDir.Data
		currentParentDirFileInfo = newCurrentDirFileInfo
	}

	return currentParentDir, nil
}

// TODO public ID or private ID?
func (qq *fileSystemCommon) MakeDir(ctx ctxx.Context, parentDirID string, newDirName string) (*model.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	newDirName = filepath.Clean(newDirName)

	if !filenamex.IsAllowed(newDirName) {
		log.Println("filename is not allowed", newDirName)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "The provided filename is not allowed.")
	}

	// parentDir := ctx.TenantCtx().TTx.File.GetX(ctx, parentDirID)
	parentDir := ctx.TenantCtx().TTx.File.Query().Where(file.PublicID(entx.NewCIText(parentDirID))).OnlyX(ctx)

	// FIXME case sensitivy
	if parentDir.QueryChildren().Where(file.Name(newDirName)).ExistX(ctx) {
		log.Println("duplicate file", newDirName)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "A folder with this name already exists.")
	}

	// FIXME handle transaction or let indexer handle such situations?
	filex := ctx.TenantCtx().TTx.File.Create().
		SetName(newDirName).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()).
		// TODO take mode time from uploadedFile if possible at all
		SetModifiedAt(time.Now()).
		SetParentID(parentDir.ID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SaveX(ctx)

	return model.NewFile(filex), nil
}

// TODO name Move or Rename to be more consistent with FS interface?
func (qq *fileSystemCommon) Move(
	ctx ctxx.Context,
	destDir, filex *model.File,
	newFilename string,
	dirNameToCreate string, // TODO name?
) (*model.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	if !destDir.Data.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Destination is not a directory.")
	}
	if filex.Data.ID == destDir.Data.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot move directory to itself.")
	}
	if filex.Data.ParentID == destDir.Data.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Destination is current location.")
	}

	if dirNameToCreate != "" {
		newDir, err := qq.MakeDir(ctx, destDir.Data.PublicID.String(), dirNameToCreate)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		destDir = newDir
	}

	if !destDir.Data.IsDirectory {
		log.Println("not a directory")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "destination is not a directory")
	}

	destDirInfo := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileID(destDir.Data.ID)).OnlyX(ctx)
	if slices.Contains(destDirInfo.Path, filex.Data.ID) {
		log.Println("cannot move file into child directory")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "cannot move file into child directory")
	}

	fileUpdate := filex.Data.Update()
	if destDir.Data.ID != filex.Data.ParentID {
		fileUpdate.SetParent(destDir.Data)
	}
	if newFilename != "" {
		fileUpdate.SetName(newFilename)
	}

	// returns new pointer, thus must be returned to caller
	// TODO not very nice solution
	filexx := fileUpdate.SaveX(ctx)
	filex = model.NewFile(filexx)

	// FIXME is overwrite automatically prevented by unique constraint? impl test
	return filex, nil
}

func (qq *fileSystemCommon) Rename(ctx ctxx.Context, filex *model.File, newFilename string) (*model.File, error) {
	// TODO block in non folder mode?

	if newFilename == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "New filename is empty.")
	}
	if filex.Data.Name == newFilename {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "New filename is the same as old.")
	}

	// returns new pointer, thus must be returned to caller
	filexx := filex.Data.Update().SetName(newFilename).SaveX(ctx)
	filex = model.NewFile(filexx)

	// FIXME is overwrite automatically prevented by unique constraint? impl test
	return filex, nil
}
