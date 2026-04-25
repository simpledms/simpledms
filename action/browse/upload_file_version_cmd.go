package browse

import (
	"log"
	"math"
	"net/http"
	"path/filepath"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	"github.com/marcobeierer/go-core/util/fileutil"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/txx"
	"github.com/marcobeierer/go-core/util/uploadx"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
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
func (qq *UploadFileVersionCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	nilableUploadLimitBytes, err := qq.infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		return err
	}
	if nilableUploadLimitBytes != nil {
		bodyLimitBytes := *nilableUploadLimitBytes
		const multipartOverheadBytes int64 = 1 * 1024 * 1024
		if bodyLimitBytes < math.MaxInt64-multipartOverheadBytes {
			bodyLimitBytes += multipartOverheadBytes
		}
		req.Request.Body = http.MaxBytesReader(rw, req.Request.Body, bodyLimitBytes)
	}

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
	if err := fileutil.EnsureFileDoesNotExist(ctx, filename, filex.Data.ParentID, filex.Data.IsInInbox); err != nil {
		return err
	}

	prep, err := txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*filesystem.PreparedUpload, error) {
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
		uploadx.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep, err, true)
		return err
	}

	_, err = txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		return nil, qq.infra.FileSystem().FinalizePreparedUpload(writeCtx, prep, fileInfo, fileSize)
	})
	if err != nil {
		uploadx.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep, err, false)
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("New version uploaded for «%s».", filex.Data.Name))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}
