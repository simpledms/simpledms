package browse

import (
	"io"
	"log"
	"mime/multipart"
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

	data, uploadedFile, err := qq.readUpload(req)
	if err != nil {
		return err
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

	prepared, fileName, err := qq.prepareUpload(ctx, data.FileID, filename)
	if err != nil {
		return err
	}
	if err := uploadPreparedFile(qq.infra, ctx, uploadedFile, prepared); err != nil {
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("New version uploaded for «%s».", fileName))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}

func (qq *UploadFileVersionCmd) prepareUpload(
	ctx ctxx.Context,
	fileID string,
	filename string,
) (*filesystem.PreparedUpload, string, error) {
	type prepareResult struct {
		prepared *filesystem.PreparedUpload
		fileName string
	}

	result, err := txx.WithTenantWriteSpaceTx(ctx.SpaceCtx(), func(writeCtx *ctxx.SpaceContext) (*prepareResult, error) {
		filex := qq.infra.FileRepo.GetX(writeCtx, fileID)
		if filex.Data.IsDirectory {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot upload versions for directories.")
		}
		if err := fileutil.EnsureFileDoesNotExist(
			writeCtx,
			filename,
			filex.Data.ParentID,
			filex.Data.IsInInbox,
		); err != nil {
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
		return &prepareResult{prepared: prepared, fileName: filex.Data.Name}, nil
	})
	if err != nil {
		return nil, "", err
	}
	return result.prepared, result.fileName, nil
}

func (qq *UploadFileVersionCmd) readUpload(
	req *httpx.Request,
) (*UploadFileVersionCmdData, *uploadx.MultipartFile, error) {
	reader, err := req.MultipartReader()
	if err != nil {
		return nil, nil, err
	}
	data := &UploadFileVersionCmdData{}
	var uploadedFile *uploadx.MultipartFile
	for uploadedFile == nil {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		uploadedFile, err = qq.readUploadPart(data, part)
		if err != nil {
			return nil, nil, err
		}
	}
	return data, uploadedFile, nil
}

func (qq *UploadFileVersionCmd) readUploadPart(
	data *UploadFileVersionCmdData,
	part *multipart.Part,
) (*uploadx.MultipartFile, error) {
	switch part.FormName() {
	case "FileID":
		value, err := readMultipartValue(part, 1024)
		data.FileID = value
		return nil, err
	case "File":
		if data.FileID == "" {
			_ = part.Close()
			return nil, e.NewHTTPErrorf(
				http.StatusBadRequest,
				"Upload metadata must be sent before the file.",
			)
		}
		uploadedFile, err := uploadx.NewMultipartFile(part)
		if err != nil {
			_ = part.Close()
		}
		return uploadedFile, err
	default:
		_ = part.Close()
		return nil, nil
	}
}
