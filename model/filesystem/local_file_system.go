package filesystem

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/storagetype"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
)

type LocalFileSystem struct {
	common         *fileSystemCommon
	storageDirPath string
}

func NewLocalFileSystem(metaPath string, storageDirPath string, fileSystemCommon *fileSystemCommon) *LocalFileSystem {
	return &LocalFileSystem{
		common:         fileSystemCommon,
		storageDirPath: storageDirPath,
	}
}

// very similar code in S3FileSystem
func (qq *LocalFileSystem) SaveFile(
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
