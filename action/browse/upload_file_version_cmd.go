package browse

import (
	"log"
	"net/http"
	"path/filepath"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFileVersionCmdData struct {
	FileID string `form_attr_type:"hidden"`
	File   []byte `schema:"-"`
}

type UploadFileVersionCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUploadFileVersionCmd(
	infra *common.Infra,
	actions *Actions,
) *UploadFileVersionCmd {
	config := actionx.NewConfig(
		actions.Route("upload-file-version-cmd"),
		false,
	).EnableManualTxManagement()

	return &UploadFileVersionCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *UploadFileVersionCmd) Data(fileID string) *UploadFileVersionCmdData {
	return &UploadFileVersionCmdData{
		FileID: fileID,
		File:   []byte(""),
	}
}

// very similar to UploadFileCmd
func (qq *UploadFileVersionCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UploadFileVersionCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.FileID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file provided.")
	}

	uploadedFile, uploadedFileHeader, err := req.FormFile("File")
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not read file")
	}
	defer func() {
		if err := uploadedFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	filename := filepath.Clean(uploadedFileHeader.Filename)

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	if filex.Data.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Cannot upload versions for directories.")
	}
	if err := autil.EnsureFileDoesNotExist(ctx, filename, filex.Data.ParentID, filex.Data.IsInInbox); err != nil {
		return err
	}

	prep, err := autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*filesystem.PreparedUpload, error) {
		return qq.infra.FileSystem().PrepareFileVersionUpload(
			writeCtx,
			filename,
			filex.Data.ID,
		)
	})
	if err != nil {
		return err
	}

	fileInfo, fileSize, err := qq.infra.FileSystem().UploadPreparedFileWithExpectedSize(
		ctx,
		uploadedFile,
		prep,
		uploadedFileHeader.Size,
	)
	if err != nil {
		autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep, err, true)
		return err
	}

	_, err = autil.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		return nil, qq.infra.FileSystem().FinalizePreparedUpload(writeCtx, prep, fileInfo, fileSize)
	})
	if err != nil {
		autil.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep, err, false)
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("New version uploaded for «%s».", filex.Data.Name))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}
