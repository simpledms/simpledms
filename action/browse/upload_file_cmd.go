package browse

import (
	"log"
	"math"
	"net/http"
	"path/filepath"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	"github.com/marcobeierer/go-core/util/fileutil"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/txx"
	"github.com/marcobeierer/go-core/util/uploadx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type UploadFileCmdData struct {
	ParentDirID string `form_attr_type:"hidden"`
	File        []byte `schema:"-"`
	// for renaming
	// TODO preset to uploaded file name
	// TODO option to quickly rename according to pattern defined for folder
	Filename   string // TODO only if in FolderMode
	AddToInbox bool   // TODO only in non-folder mode
}

type UploadFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UploadFileCmdData]
}

func NewUploadFileCmd(
	infra *common.Infra,
	actions *Actions,
) *UploadFileCmd {
	config := actionx.NewConfig(
		actions.Route("upload-file-cmd"),
		false,
	).EnableManualTxManagement()

	formHelper := autil.NewFormHelper[UploadFileCmdData](
		infra,
		config,
		widget.T("Upload file"),
		// "#fileList",
	)
	formHelper.SetIsMultipartFormData(true)

	return &UploadFileCmd{
		infra,
		actions,
		config,
		formHelper,
	}
}

func (qq *UploadFileCmd) Data(parentDirID string, filename string, addToInbox bool) *UploadFileCmdData {
	return &UploadFileCmdData{
		ParentDirID: parentDirID,
		File:        []byte(""),
		Filename:    filename,
		AddToInbox:  addToInbox,
	}
}

// very similar to UploadFileVersionCmd
func (qq *UploadFileCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	data, err := autil.FormData[UploadFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.ParentDirID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No parent dir provided.")
	}
	if !ctx.SpaceCtx().TenantCtx().IsReadOnlyTx() {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Read-only request context required.")
	}

	uploadedFile, uploadedFileHeader, err := req.FormFile("File")
	if err != nil {
		// TODO also triggers if no file provided
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not read file")
	}
	defer func() {
		if err := uploadedFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	filename := uploadedFileHeader.Filename
	if data.Filename != "" {
		filename = data.Filename
	}
	filename = filepath.Clean(filename)

	parentDir := qq.infra.FileRepo.GetX(ctx, data.ParentDirID)

	if err := fileutil.EnsureFileDoesNotExist(ctx, filename, parentDir.Data.ID, data.AddToInbox); err != nil {
		return err
	}

	type uploadPrepareResult struct {
		prepared *filesystem.PreparedUpload
		filex    *enttenant.File
	}

	prep, err := txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*uploadPrepareResult, error) {
		prepared, filex, err := qq.infra.FileSystem().PrepareFileUpload(
			writeCtx,
			filename,
			parentDir.Data.ID,
			data.AddToInbox,
		)
		if err != nil {
			return nil, err
		}
		return &uploadPrepareResult{prepared: prepared, filex: filex}, nil
	})
	if err != nil {
		return err
	}

	fileInfo, fileSize, err := qq.infra.FileSystem().UploadPreparedFileWithExpectedSize(
		ctx,
		uploadedFile,
		prep.prepared,
		uploadedFileHeader.Size,
	)
	if err != nil {
		uploadx.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep.prepared, err, true)
		return err
	}

	_, err = txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		return nil, qq.infra.FileSystem().FinalizePreparedUpload(writeCtx, prep.prepared, fileInfo, fileSize)
	})
	if err != nil {
		uploadx.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep.prepared, err, false)
		return err
	}

	rw.AddRenderables(widget.NewSnackbarf("«%s» uploaded.", prep.filex.Name))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}
