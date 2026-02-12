package browse

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type unzipPreparedEntry struct {
	zipFile  *zip.File
	prepared *filesystem.PreparedUpload
	fileID   int64
	fileInfo *minio.UploadInfo
	fileSize int64
}

type UnzipArchiveCmdData struct {
	// TODO show warning that flatten if not in folder mode
	// TODO extracted in current folder
	FileID          string `validate:"required" form_attr_type:"hidden"`
	DeleteOnSuccess bool
	// TargetDirID string // TODO only in FolderMode
	// Flat
}

type UnzipArchiveCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UnzipArchiveCmdData]
}

func NewUnzipArchiveCmd(infra *common.Infra, actions *Actions) *UnzipArchiveCmd {
	config := actionx.NewConfig(actions.Route("unzip-archive-cmd"), false).EnableManualTxManagement()
	return &UnzipArchiveCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[UnzipArchiveCmdData](infra, config, wx.T("Unzip archive")),
	}
}

func (qq *UnzipArchiveCmd) Data(fileID string, deleteOnSuccess bool) *UnzipArchiveCmdData {
	return &UnzipArchiveCmdData{
		FileID:          fileID,
		DeleteOnSuccess: deleteOnSuccess,
	}
}

func (qq *UnzipArchiveCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UnzipArchiveCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filem := qq.infra.FileRepo.GetX(ctx, data.FileID)

	if !filem.IsZIPArchive(ctx) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Not a ZIP archive.")
	}

	// Get the current version of the file
	storedFile := filem.CurrentVersion(ctx)

	// Open the ZIP file using S3FileSystem
	zipFileReader, err := qq.infra.FileSystem().OpenFile(ctx, storedFile)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not open ZIP archive.")
	}
	defer func() {
		err = zipFileReader.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	tmpFilePath := fmt.Sprintf(
		"%s/simpledms-unzip-%s-%s-%s.zip",
		os.TempDir(),
		ctx.SpaceCtx().TenantID,
		ctx.SpaceCtx().SpaceID,
		util.NewPublicID(), // storedFile.Data.PublicID,
	)
	tmpFile, err := os.OpenFile(tmpFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		// TODO if already exists, let user know that unzip is in progress
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not unzip the archive.")
	}
	defer func() {
		err = tmpFile.Close()
		if err != nil {
			log.Println(err)
		}
		err = os.Remove(tmpFile.Name())
		if err != nil {
			log.Println(err)
		}
	}()

	tmpFileSize, err := io.Copy(tmpFile, zipFileReader)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not unzip the archive.")
	}
	err = zipFileReader.Close()
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not unzip the archive.")
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not unzip the archive.")
	}

	// Create a zip.Reader from the file
	// We need to read the entire file into memory because zip.Reader requires an io.ReaderAt
	// FIXME possible to stream zipFileReader instead of reading it into memory?
	/*zipData, err := io.ReadAll(zipFileReader)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read ZIP archive.")
	}*/

	// zipArchiveReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	zipArchiveReader, err := zip.NewReader(tmpFile, tmpFileSize)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read ZIP archive.")
	}

	zipArchiveTotalUncompressedSize, err := qq.unzipArchiveTotalUncompressedSize(zipArchiveReader.File)
	if err != nil {
		log.Println(err)
		return err
	}
	err = qq.infra.FileSystem().EnsureTenantStorageLimit(ctx, zipArchiveTotalUncompressedSize)
	if err != nil {
		log.Println(err)
		return err
	}

	// only used in non-folder mode
	parentDirID := filem.Data.ParentID

	preparedEntries, err := autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) ([]*unzipPreparedEntry, error) {
		entries := make([]*unzipPreparedEntry, 0, len(zipArchiveReader.File))
		parentDir := writeCtx.SpaceCtx().Space.QueryFiles().Where(file.ID(parentDirID)).OnlyX(writeCtx)

		for _, zippedFile := range zipArchiveReader.File {
			if zippedFile.FileInfo().IsDir() {
				if !writeCtx.SpaceCtx().Space.IsFolderMode {
					continue
				}

				_, err := qq.infra.FileSystem().MakeDirAllIfNotExists(writeCtx, parentDir, zippedFile.Name)
				if err != nil {
					log.Println(err)
					return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not create directory structure.")
				}
				continue
			}

			fileParentID := parentDirID
			if writeCtx.SpaceCtx().Space.IsFolderMode {
				pathWithoutFilename := filepath.Dir(zippedFile.Name)
				if pathWithoutFilename != "." {
					newParentDir, err := qq.infra.FileSystem().MakeDirAllIfNotExists(writeCtx, parentDir, pathWithoutFilename)
					if err != nil {
						log.Println(err)
						return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not create directory structure.")
					}
					fileParentID = newParentDir.ID
				}
			}

			filename := filepath.Base(zippedFile.Name)
			if err := autil.EnsureFileDoesNotExist(writeCtx, filename, fileParentID, false); err != nil {
				return nil, err
			}

			prepared, filex, err := qq.infra.FileSystem().PrepareFileUpload(
				writeCtx,
				filename,
				fileParentID,
				false,
			)
			if err != nil {
				return nil, err
			}

			entries = append(entries, &unzipPreparedEntry{
				zipFile:  zippedFile,
				prepared: prepared,
				fileID:   filex.ID,
			})
		}

		return entries, nil
	})
	if err != nil {
		return err
	}

	hasErr := false
	for _, entry := range preparedEntries {
		fileToSave, err := entry.zipFile.Open()
		if err != nil {
			log.Println(err)
			hasErr = true
			break
		}

		fileInfo, fileSize, err := qq.infra.FileSystem().UploadPreparedFileWithExpectedSize(
			ctx,
			fileToSave,
			entry.prepared,
			int64(entry.zipFile.UncompressedSize64),
		)
		_ = fileToSave.Close()
		if err != nil {
			log.Println(err)
			hasErr = true
			break
		}

		entry.fileInfo = fileInfo
		entry.fileSize = fileSize
	}

	if hasErr {
		rw.AddRenderables(wx.NewSnackbarf("Could not extract all files from archive.").SetIsError(true))

		for _, entry := range preparedEntries {
			cleanup := entry.fileInfo != nil
			autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), entry.prepared, nil, cleanup)
		}

		fileIDs := make([]int64, 0, len(preparedEntries))
		for _, entry := range preparedEntries {
			fileIDs = append(fileIDs, entry.fileID)
		}
		_, err = autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
			if len(fileIDs) == 0 {
				return nil, nil
			}
			writeCtx.TTx.File.Update().
				Where(file.IDIn(fileIDs...)).
				SetDeletedAt(time.Now()).
				SetDeleter(writeCtx.SpaceCtx().User).
				ExecX(writeCtx)
			return nil, nil
		})
		if err != nil {
			log.Println(err)
		}
	} else {
		_, err = autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
			totalUploadedBytes, err := qq.unzipPreparedEntriesTotalUploadedSize(preparedEntries)
			if err != nil {
				return nil, err
			}
			err = qq.infra.FileSystem().EnsureTenantStorageLimit(writeCtx, totalUploadedBytes)
			if err != nil {
				return nil, err
			}

			for _, entry := range preparedEntries {
				if entry.fileInfo == nil {
					return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not extract all files from archive.")
				}
				err := qq.infra.FileSystem().FinalizePreparedUpload(writeCtx, entry.prepared, entry.fileInfo, entry.fileSize)
				if err != nil {
					return nil, err
				}
			}
			return nil, nil
		})
		if err != nil {
			log.Println(err)
			for _, entry := range preparedEntries {
				cleanup := entry.fileInfo != nil
				autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), entry.prepared, nil, cleanup)
			}
			hasErr = true
		}
	}

	// If DeleteOnSuccess is true, mark the ZIP file as deleted
	if data.DeleteOnSuccess && !hasErr {
		_, err = autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
			writeCtx.TTx.File.UpdateOneID(filem.Data.ID).
				SetDeletedAt(time.Now()).
				SetDeleter(writeCtx.SpaceCtx().User).
				ExecX(writeCtx)
			return nil, nil
		})
		if err != nil {
			log.Println(err)
			hasErr = true
		}
	}

	rw.Header().Set("HX-Trigger", event.ZIPArchiveUnzipped.String())
	if !hasErr {
		rw.AddRenderables(wx.NewSnackbarf("Archive unzipped."))
	}

	return nil
}

func (qq *UnzipArchiveCmd) unzipArchiveTotalUncompressedSize(zipFiles []*zip.File) (int64, error) {
	var total uint64
	for _, zippedFile := range zipFiles {
		if zippedFile.FileInfo().IsDir() {
			continue
		}
		if zippedFile.UncompressedSize64 > uint64(math.MaxInt64) {
			return 0, e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Archive is too large.")
		}
		if total > uint64(math.MaxInt64)-zippedFile.UncompressedSize64 {
			return 0, e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Archive is too large.")
		}

		total += zippedFile.UncompressedSize64
	}

	return int64(total), nil
}

func (qq *UnzipArchiveCmd) unzipPreparedEntriesTotalUploadedSize(preparedEntries []*unzipPreparedEntry) (int64, error) {
	var total int64
	for _, entry := range preparedEntries {
		if entry.fileSize < 0 {
			return 0, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify archive size.")
		}
		if total > math.MaxInt64-entry.fileSize {
			return 0, e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Archive is too large.")
		}

		total += entry.fileSize
	}

	return total, nil
}
