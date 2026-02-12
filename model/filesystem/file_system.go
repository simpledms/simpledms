package filesystem

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
)

// Deprecated: use S3FileSystem instead
type FileSystem struct {
	// storageDirPath string
	metaPath string
	fileTree *FileTree
}

// Deprecated: use S3FileSystem instead
func NewFileSystem(metaPath string) *FileSystem {
	return &FileSystem{
		metaPath: metaPath,
		fileTree: NewFileTree(),
		// storageDirPath: storageDirPath,
	}
}

func (qq *FileSystem) FileTree() *FileTree {
	if qq.fileTree == nil {
		qq.fileTree = NewFileTree()
	}
	return qq.fileTree
}

// TODO public ID or private ID?
// returns last created dir
func (qq *FileSystem) MakeDirAllIfNotExists(ctx ctxx.Context, currentParentDir *enttenant.File, pathToCreate string) (*enttenant.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	pathToCreate = filepath.Clean(pathToCreate)
	// filepath.SplitList uses filepath.ListSeparator, thus not the same
	// TODO correct that os specific? for unzip it seems so because paths are handled on server
	pathElems := strings.Split(pathToCreate, string(filepath.Separator))

	for _, pathElem := range pathElems {
		if pathElem == "" || pathElem == "." {
			continue
		}

		newCurrentDirx, err := ctx.TenantCtx().TTx.File.Query().
			Where(
				file.SpaceID(ctx.SpaceCtx().Space.ID),
				file.ParentID(currentParentDir.ID),
				file.Name(pathElem),
			).
			Only(ctx)
		if err != nil && !enttenant.IsNotFound(err) {
			log.Println(err)
			return nil, err
		}

		if enttenant.IsNotFound(err) {
			newCurrentDir, err := qq.MakeDir(ctx, currentParentDir.PublicID.String(), pathElem)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			currentParentDir = newCurrentDir.Data
			continue
		} else {
			// TODO good? correct location
			if !newCurrentDirx.IsDirectory {
				return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Path element is file, not a directory.")
			}
		}

		currentParentDir = newCurrentDirx
	}

	return currentParentDir, nil
}

// TODO public ID or private ID?
func (qq *FileSystem) MakeDir(ctx ctxx.Context, parentDirID string, newDirName string) (*model.File, error) {
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
func (qq *FileSystem) Move(
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

	isDescendant, err := qq.FileTree().IsDescendantOf(ctx, destDir.Data.ID, filex.Data.ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if isDescendant {
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

func (qq *FileSystem) Rename(ctx ctxx.Context, filex *model.File, newFilename string) (*model.File, error) {
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

/*
// IMPORTANT
// very similar code in S3FileSystem
func (qq *FileSystem) AddFile(
	ctx ctxx.Context,
	fileToSave multipart.File,
	filename string,
	isInInbox bool,
	parentDirInfo *enttenant.FileInfo,
) (*enttenant.File, error) {
	filename = filepath.Clean(filename)

	if !filenamex.IsAllowed(filename) {
		log.Println("invalid filename")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "invalid filename")
	}

	fileExtension := filepath.Ext(filename)
	if fileExtension == "" {
		log.Println("invalid filename")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "file has no extension")
	}

	// TODO use nanoid? or would folders get to large (1296 vs 256 sub dirs)
	pathToStorageDir := func() string {
		bytes := make([]byte, 3) // 3 bytes = 6 hex characters
		if _, err := rand.Read(bytes); err != nil {
			log.Println("Error generating random bytes:", err)
			return ""
		}
		return hex.EncodeToString(bytes)
	}()

	if ctx.TenantCtx().StoragePath == "" {
		log.Println("storage path is empty")
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "storage path is empty")
	}

	// 3 indirections with 2 hex chars each allow for 256 x 256 x 256 directories, with 256 files in each dir
	// we get a safe structure for over 4 billion files
	storageLocation, err := securejoin.SecureJoin(ctx.TenantCtx().StoragePath, pathToStorageDir[0:2])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	storageLocation, err = securejoin.SecureJoin(storageLocation, pathToStorageDir[2:4])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	storageLocation, err = securejoin.SecureJoin(storageLocation, pathToStorageDir[4:6])
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = os.MkdirAll(storageLocation, 0777)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "could not create storage directory")
	}

	// FIXME handle transaction or let indexer handle such situations?
	filex := ctx.TenantCtx().TTx.File.Create().
		SetName(filename).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		// TODO take mode time from uploadedFile if possible at all
		// SetModifiedAt(fileInfo.ModTime()). // TODO necessary?
		SetParentID(parentDirInfo.FileID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SetIsInInbox(isInInbox).
		// AddSpaceIDs(ctx.SpaceCtx().Space.ID).
		// AddVersions(fileVersionx).
		SaveX(ctx)

	// FIXME PublicID or private ID? does anybody see filenames?
	destFilename := filex.PublicID.String() + fileExtension

	// two steps are necessary for security reasons, otherwise user can create outside current dir
	destPath, err := securejoin.SecureJoin(storageLocation, destFilename)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "could not join paths")
	}

	if _, err = os.Stat(destPath); !errors.Is(err, os.ErrNotExist) {
		log.Println("file already exists")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "file already exists")
	}

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "could not open file")
	}

	_, err = io.Copy(destFile, fileToSave)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "could not copy file")
	}

	fileInfo, err := destFile.Stat()
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "could not stat file")
	}

	storageLocationRel, err := filepath.Rel(ctx.TenantCtx().StoragePath, destPath) // destPath including filename
	if err != nil {
		log.Println(err)
		return nil, err
	}

	storedFilex := ctx.TenantCtx().TTx.StoredFile.Create().
		SetFilename(filename).
		SetSize(fileInfo.Size()).          // only okay as long as it doesn't get gzipped
		SetSizeInStorage(fileInfo.Size()). // no gzipped used
		SetStorageType(storagetype.Local).
		SetStoragePath(filepath.Dir(storageLocationRel)).
		SetStorageFilename(destFilename).
		// SetMimeType(contentType). // TODO is this okay?
		SaveX(ctx)
	filex.Update().
		AddVersionIDs(storedFilex.ID).
		SaveX(ctx)

	// TODO not very clean; only in case contentType is empty
	_, err = qq.UpdateMimeType(ctx, false, model.NewStoredFile(storedFilex))
	if err != nil {
		log.Println(err)
		// not critical
	}

	return filex, nil
}

// near duplicate in S3FileSystem
// TODO panic instead of error and rename to MimeType()? indexer must than handle panic!!
func (qq *FileSystem) UpdateMimeType(ctx ctxx.Context, force bool, filex *model.StoredFile) (string, error) {
	if filex.Data.MimeType != "" && !force {
		return filex.Data.MimeType, nil
	}

	relPath, err := filex.RelFilePath()
	if err != nil {
		log.Println(err)
		return "", err
	}

	path, err := securejoin.SecureJoin(qq.metaPath, relPath)
	if err != nil {
		log.Println(err)
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	// seems to be necessary to remove zero values which can cause false detection; not verified
	// by me, see:
	// https://gist.github.com/rayrutjes/db9b9ea8e02255d62ce2?permalink_comment_id=3418419#gistcomment-3418419
	buf = buf[:n]

	mimeType := http.DetectContentType(buf)
	filex.Data = filex.Data.Update().SetMimeType(mimeType).SaveX(ctx)

	// after probing mimetype
	_, err = f.Seek(0, 0)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	return mimeType, nil
}
*/
