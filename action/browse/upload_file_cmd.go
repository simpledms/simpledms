package browse

import (
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/fileutil"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/txx"
	"github.com/simpledms/simpledms/util/uploadx"
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
func (qq *UploadFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	nilableUploadLimitBytes, err := qq.infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		return err
	}
	uploadx.LimitMultipartBody(rw, req.Request, nilableUploadLimitBytes)

	reader, err := req.MultipartReader()
	if err != nil {
		return err
	}
	data := &UploadFileCmdData{}
	var uploadedFile *uploadx.MultipartFile

	for uploadedFile == nil {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch part.FormName() {
		case "ParentDirID":
			value, err := io.ReadAll(io.LimitReader(part, 1024))
			_ = part.Close()
			if err != nil {
				return err
			}
			data.ParentDirID = string(value)
		case "Filename":
			value, err := io.ReadAll(io.LimitReader(part, 4096))
			_ = part.Close()
			if err != nil {
				return err
			}
			data.Filename = string(value)
		case "AddToInbox":
			value, err := io.ReadAll(io.LimitReader(part, 16))
			_ = part.Close()
			if err != nil {
				return err
			}
			val := strings.TrimSpace(string(value))
			data.AddToInbox = val == "on" || val == "true" || val == "1"
		case "File":
			if data.ParentDirID == "" {
				_ = part.Close()
				return e.NewHTTPErrorf(http.StatusBadRequest, "Upload metadata must be sent before the file.")
			}
			uploadedFile, err = uploadx.NewMultipartFile(part)
			if err != nil {
				_ = part.Close()
				return err
			}
		default:
			_ = part.Close()
		}
	}

	if data.ParentDirID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No parent dir provided.")
	}
	if !ctx.SpaceCtx().TenantCtx().IsReadOnlyTx() {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Read-only request context required.")
	}

	if uploadedFile == nil {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file provided.")
	}
	defer func() {
		if err := uploadedFile.Closer.Close(); err != nil {
			log.Println(err)
		}
	}()

	filename := uploadedFile.Filename
	if data.Filename != "" {
		filename = data.Filename
	}
	filename = filepath.Clean(filename)

	type uploadPrepareResult struct {
		prepared *filesystem.PreparedUpload
	}

	prep, err := txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*uploadPrepareResult, error) {
		parentDir := qq.infra.FileRepo.GetX(writeCtx, data.ParentDirID)
		if err := fileutil.EnsureFileDoesNotExist(writeCtx, filename, parentDir.Data.ID, data.AddToInbox); err != nil {
			return nil, err
		}
		prepared, err := qq.infra.FileSystem().PrepareFileUpload(
			writeCtx,
			filename,
			parentDir.Data.ID,
			data.AddToInbox,
		)
		if err != nil {
			return nil, err
		}
		return &uploadPrepareResult{prepared: prepared}, nil
	})
	if err != nil {
		return err
	}

	var uploadResult *filesystem.PreparedUploadResult
	if uploadedFile.ExpectedBytes != nil {
		uploadResult, err = qq.infra.FileSystem().UploadPreparedFileWithExpectedSize(
			ctx,
			uploadedFile.Reader,
			prep.prepared,
			*uploadedFile.ExpectedBytes,
		)
	} else {
		uploadResult, err = qq.infra.FileSystem().UploadPreparedFile(ctx, uploadedFile.Reader, prep.prepared)
	}
	if err != nil {
		uploadx.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep.prepared, err, true)
		return err
	}

	_, err = txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		return nil, qq.infra.FileSystem().FinalizePreparedUploadWithoutMime(writeCtx, prep.prepared, uploadResult)
	})
	if err != nil {
		uploadx.HandleStoredFileUploadFailure(ctx.SpaceCtx(), qq.infra.FileSystem(), prep.prepared, err, true)
		return err
	}
	if _, err := qq.infra.FileSystem().UpdateMimeTypeAfterFinalization(
		ctx.SpaceCtx(),
		true,
		prep.prepared.StoredFileID,
	); err != nil {
		log.Println(err)
	}

	rw.AddRenderables(widget.NewSnackbarf("«%s» uploaded.", filename))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}
