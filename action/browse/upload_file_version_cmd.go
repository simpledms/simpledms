package browse

import (
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/simpledms/simpledms/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
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
	nilableUploadLimitBytes, err := qq.infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		return err
	}
	uploadx.LimitMultipartBody(rw, req.Request, nilableUploadLimitBytes)

	reader, err := req.MultipartReader()
	if err != nil {
		return err
	}
	data := &UploadFileVersionCmdData{}
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
		case "FileID":
			value, err := io.ReadAll(io.LimitReader(part, 1024))
			_ = part.Close()
			if err != nil {
				return err
			}
			data.FileID = string(value)
		case "File":
			if data.FileID == "" {
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

	if data.FileID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file provided.")
	}

	if uploadedFile == nil {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No file provided.")
	}
	defer func() {
		if err := uploadedFile.Closer.Close(); err != nil {
			log.Println(err)
		}
	}()

	filename := filepath.Clean(uploadedFile.Filename)

	type uploadVersionPrepareResult struct {
		prepared *filesystem.PreparedUpload
		fileName string
	}

	prep, err := txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*uploadVersionPrepareResult, error) {
		filex := qq.infra.FileRepo.GetX(writeCtx, data.FileID)
		if filex.Data.IsDirectory {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot upload versions for directories.")
		}
		if err := fileutil.EnsureFileDoesNotExist(writeCtx, filename, filex.Data.ParentID, filex.Data.IsInInbox); err != nil {
			return nil, err
		}
		prepared, err := qq.infra.FileSystem().PrepareFileVersionUpload(
			writeCtx,
			filename,
			filex.Data.ID,
		)
		if err != nil {
			return nil, err
		}
		return &uploadVersionPrepareResult{prepared: prepared, fileName: filex.Data.Name}, nil
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

	rw.AddRenderables(wx.NewSnackbarf("New version uploaded for «%s».", prep.fileName))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}
