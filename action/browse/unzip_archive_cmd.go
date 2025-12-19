package browse

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/recoverx"
)

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
	config := actionx.NewConfig(actions.Route("unzip-archive-cmd"), false)
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

	// only used in non-folder mode
	parentDirID := filem.Data.ParentID

	// create all directories first because they cannot be created on demand concurrently; files in the same
	// subdirectory may gets processed concurrently and thus both go routines would try to create
	// the subdirectory...
	for _, zippedFile := range zipArchiveReader.File {
		err = qq.createDirectoryStructure(ctx, zippedFile, parentDirID)
		if err != nil {
			log.Println(err)
			return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not create directory structure.")
		}
	}

	hasErr := false
	ch := make(chan bool, 25) // TODO as config option?
	wg := sync.WaitGroup{}

	for _, zippedFile := range zipArchiveReader.File {
		wg.Add(1)
		ch <- true // acquire slot

		go func() {
			defer recoverx.Recover("unzipAndSaveFile")

			defer wg.Done()
			defer func() {
				<-ch // release slot
			}()

			err = qq.unzipAndSaveFile(ctx, zippedFile, parentDirID)
			if err != nil {
				log.Println(err)
				// TODO more detailed? thus all failed filenames?
				rw.AddRenderables(wx.NewSnackbarf("Could not extract all files from archive.").SetIsError(true))

				// FIXME return and rollback? could be implemented by adding uploadToken to files
				//		and only persist if the upload was successful
				hasErr = true

				// continue
			}
		}()
	}

	wg.Wait()

	// If DeleteOnSuccess is true, mark the ZIP file as deleted
	if data.DeleteOnSuccess && !hasErr {
		// TODO via FileSystem?
		filem.Data.Update().
			SetDeletedAt(time.Now()).
			SetDeleter(ctx.SpaceCtx().User).
			SaveX(ctx)
	}

	rw.Header().Set("HX-Trigger", event.ZIPArchiveUnzipped.String())
	if !hasErr {
		rw.AddRenderables(wx.NewSnackbarf("Archive unzipped."))
	}

	return nil
}

func (qq *UnzipArchiveCmd) createDirectoryStructure(ctx ctxx.Context, zippedFile *zip.File, parentDirID int64) error {
	if !zippedFile.FileInfo().IsDir() {
		// do nothing
		return nil
	}

	if !ctx.SpaceCtx().Space.IsFolderMode {
		// do nothing
		return nil
	}

	parentDir := ctx.SpaceCtx().Space.QueryFiles().Where(file.ID(parentDirID)).OnlyX(ctx)

	// Name is relative path
	_, err := qq.infra.FileSystem().MakeDirAllIfNotExists(ctx, parentDir, zippedFile.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (qq *UnzipArchiveCmd) unzipAndSaveFile(ctx ctxx.Context, zippedFile *zip.File, parentDirID int64) error {
	if zippedFile.FileInfo().IsDir() {
		// do nothing
		return nil
	}

	if ctx.SpaceCtx().Space.IsFolderMode {
		parentDir := ctx.SpaceCtx().Space.QueryFiles().Where(file.ID(parentDirID)).OnlyX(ctx)

		// if file and in folder mode, set parentDirID
		pathWithoutFilename := filepath.Dir(zippedFile.Name)

		newParentDir, err := qq.infra.FileSystem().MakeDirAllIfNotExists(ctx, parentDir, pathWithoutFilename)
		if err != nil {
			log.Println(err)
			return err
		}

		parentDirID = newParentDir.ID
	} else {
		// nothing to do in non folder mode
	}

	// Open the file in the ZIP archive
	fileToSave, err := zippedFile.Open()
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not open file in archive.")
	}
	defer func() {
		err = fileToSave.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	// Get the filename from the ZIP entry
	filename := filepath.Base(zippedFile.Name)

	// Save the file to a temporary location
	_, err = qq.infra.FileSystem().SaveFile(
		ctx,
		fileToSave,
		filename,
		false,
		parentDirID,
	)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not extract file from archive.")
	}

	return nil
}
