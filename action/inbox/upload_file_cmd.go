package inbox

// package action

import (
	"log"
	"math"
	"net/http"
	"path/filepath"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/marcobeierer/go-core/util/txx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/util/fileutil"
	"github.com/simpledms/simpledms/util/uploadx"
)

type uploadPrepareResult struct {
	prepared *filesystem.PreparedUpload
	file     *enttenant.File
}

type UploadFileCmdData struct {
	File []byte `schema:"-"`
}

type UploadFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UploadFileCmdData]
	// inboxDirInfo *ent.FileInfo
}

func NewUploadFileCmd(infra *common.Infra, actions *Actions) *UploadFileCmd {
	config := actionx.NewConfig(
		actions.Route("upload-file-cmd"),
		false,
	).EnableManualTxManagement()
	formHelper := autil.NewFormHelper[UploadFileCmdData](
		infra,
		config,
		widget.T("Upload file"),
	)
	formHelper.SetIsMultipartFormData(true)
	return &UploadFileCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *UploadFileCmd) Data() *UploadFileCmdData {
	return &UploadFileCmdData{}
}

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

	_, err = autil.FormData[UploadFileCmdData](rw, req, ctx)
	if err != nil {
		return err
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
	filename = filepath.Clean(filename)

	if err := fileutil.EnsureFileDoesNotExist(ctx, filename, ctx.SpaceCtx().SpaceRootDir().ID, true); err != nil {
		return err
	}

	prep, err := txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*uploadPrepareResult, error) {
		prepared, filex, err := qq.infra.FileSystem().PrepareFileUpload(
			writeCtx,
			filename,
			ctx.SpaceCtx().SpaceRootDir().ID,
			true,
		)
		if err != nil {
			return nil, err
		}
		return &uploadPrepareResult{prepared: prepared, file: filex}, nil
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

	rw.Header().Set("HX-Retarget", "#innerContent")
	rw.Header().Set("HX-Reswap", "innerHTML")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.InboxPage.WidgetHandler(rw, req, ctx, prep.file.PublicID.String()),
		widget.NewSnackbarf("«%s» uploaded.", prep.file.Name),
	)
}
