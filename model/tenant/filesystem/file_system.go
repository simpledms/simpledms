package filesystem

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/entx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
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
	if currentParentDir == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Current parent directory is required.")
	}

	currentParentDirDTO, err := qq.MakeDirAllIfNotExistsByID(ctx, currentParentDir.ID, pathToCreate)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &enttenant.File{
		ID:          currentParentDirDTO.ID,
		PublicID:    entx.NewCIText(currentParentDirDTO.PublicID),
		ParentID:    currentParentDirDTO.ParentID,
		SpaceID:     currentParentDirDTO.SpaceID,
		Name:        currentParentDirDTO.Name,
		IsDirectory: currentParentDirDTO.IsDirectory,
		DeletedAt:   currentParentDirDTO.DeletedAt,
	}, nil
}

func (qq *FileSystem) MakeDirAllIfNotExistsByID(
	ctx ctxx.Context,
	currentParentDirID int64,
	pathToCreate string,
) (*filemodel.FileDTO, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	pathToCreate = filepath.Clean(pathToCreate)
	// filepath.SplitList uses filepath.ListSeparator, thus not the same
	// TODO correct that os specific? for unzip it seems so because paths are handled on server
	pathElems := strings.Split(pathToCreate, string(filepath.Separator))
	readRepo := filemodel.NewEntSpaceFileReadRepository(ctx.SpaceCtx().Space.ID)

	currentParentDirDTO, err := readRepo.FileByID(ctx, currentParentDirID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	currentParentDirPublicID := currentParentDirDTO.PublicID

	for _, pathElem := range pathElems {
		if pathElem == "" || pathElem == "." {
			continue
		}

		newCurrentDirx, err := readRepo.FileByParentIDAndName(ctx, currentParentDirID, pathElem)
		if err != nil && !enttenant.IsNotFound(err) {
			log.Println(err)
			return nil, err
		}

		if enttenant.IsNotFound(err) {
			newCurrentDir, err := qq.MakeDir(ctx, currentParentDirPublicID, pathElem)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			currentParentDirID = newCurrentDir.Data.ID
			currentParentDirPublicID = newCurrentDir.Data.PublicID.String()
			continue
		} else {
			// TODO good? correct location
			if !newCurrentDirx.IsDirectory {
				return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Path element is file, not a directory.")
			}
		}

		currentParentDirID = newCurrentDirx.ID
		currentParentDirPublicID = newCurrentDirx.PublicID
	}

	currentParentDirDTO, err = readRepo.FileByID(ctx, currentParentDirID)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return currentParentDirDTO, nil
}

// TODO public ID or private ID?
func (qq *FileSystem) MakeDir(ctx ctxx.Context, parentDirID string, newDirName string) (*filemodel.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	newDirName = filepath.Clean(newDirName)

	if !filenamex.IsAllowed(newDirName) {
		log.Println("filename is not allowed", newDirName)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "The provided filename is not allowed.")
	}

	readRepo := filemodel.NewEntSpaceFileReadRepository(ctx.SpaceCtx().Space.ID)
	parentDir := readRepo.FileByPublicIDX(ctx, parentDirID)
	if !parentDir.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Parent is not a directory.")
	}

	if readRepo.FileExistsByNameAndParentX(ctx, newDirName, parentDir.ID, false) ||
		readRepo.FileExistsByNameAndParentX(ctx, newDirName, parentDir.ID, true) {
		log.Println("duplicate file", newDirName)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "A folder with this name already exists.")
	}

	repos := filemodel.NewEntSpaceFileRepositoryFactory().ForSpaceX(ctx)
	fileDTO, err := repos.Write.CreateDirectory(ctx, parentDir.ID, newDirName)
	if err != nil {
		return nil, err
	}

	return filemodel.NewFile(&enttenant.File{
		ID:          fileDTO.ID,
		PublicID:    entx.NewCIText(fileDTO.PublicID),
		ParentID:    fileDTO.ParentID,
		SpaceID:     fileDTO.SpaceID,
		Name:        fileDTO.Name,
		IsDirectory: fileDTO.IsDirectory,
		DeletedAt:   fileDTO.DeletedAt,
	}), nil
}

func (qq *FileSystem) MoveByPublicIDs(
	ctx ctxx.Context,
	destDirPublicID string,
	filePublicID string,
	newFilename string,
	dirNameToCreate string,
) (*filemodel.FileWithParentDTO, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}

	repos := filemodel.NewEntSpaceFileRepositoryFactory().ForSpaceX(ctx)
	destDir := repos.Read.FileByPublicIDX(ctx, destDirPublicID)
	fileWithParent := repos.Read.FileByPublicIDWithParentX(ctx, filePublicID)

	if !destDir.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Destination is not a directory.")
	}
	if fileWithParent.ID == destDir.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot move directory to itself.")
	}
	if fileWithParent.ParentID == destDir.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Destination is current location.")
	}

	if dirNameToCreate != "" {
		newDir, err := qq.MakeDir(ctx, destDirPublicID, dirNameToCreate)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		destDir = &filemodel.FileDTO{
			ID:          newDir.Data.ID,
			PublicID:    newDir.Data.PublicID.String(),
			ParentID:    newDir.Data.ParentID,
			SpaceID:     newDir.Data.SpaceID,
			Name:        newDir.Data.Name,
			IsDirectory: newDir.Data.IsDirectory,
			DeletedAt:   newDir.Data.DeletedAt,
		}
	}

	isDescendant, err := qq.FileTree().IsDescendantOf(ctx, destDir.ID, fileWithParent.ID)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if isDescendant {
		log.Println("cannot move file into child directory")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "cannot move file into child directory")
	}

	var nilableNewFilename *string
	if newFilename != "" {
		nilableNewFilename = &newFilename
	}

	_, err = repos.Write.MoveFileByIDX(ctx, fileWithParent.ID, destDir.ID, nilableNewFilename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return repos.Read.FileByPublicIDWithParentX(ctx, fileWithParent.PublicID), nil
}

func (qq *FileSystem) RenameByPublicID(
	ctx ctxx.Context,
	filePublicID string,
	newFilename string,
) (*filemodel.FileDTO, error) {
	if newFilename == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "New filename is empty.")
	}

	repos := filemodel.NewEntSpaceFileRepositoryFactory().ForSpaceX(ctx)
	filex := repos.Read.FileByPublicIDX(ctx, filePublicID)
	if filex.Name == newFilename {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "New filename is the same as old.")
	}

	err := repos.Write.RenameFileByIDX(ctx, filex.ID, newFilename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return repos.Read.FileByPublicIDX(ctx, filePublicID), nil
}

func (qq *FileSystem) CurrentVersionByFileIDX(ctx ctxx.Context, fileID int64) (*storedfilemodel.StoredFile, error) {
	version, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileID(fileID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		WithStoredFile().
		First(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if version.Edges.StoredFile == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File has no stored version.")
	}

	return storedfilemodel.NewStoredFile(version.Edges.StoredFile), nil
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
	_, err = qq.UpdateMimeType(ctx, false, storedfilemodel.NewStoredFile(storedFilex))
	if err != nil {
		log.Println(err)
		// not critical
	}

	return filex, nil
}

// near duplicate in S3FileSystem
// TODO panic instead of error and rename to MimeType()? indexer must than handle panic!!
func (qq *FileSystem) UpdateMimeType(ctx ctxx.Context, force bool, filex *storedfilemodel.StoredFile) (string, error) {
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
