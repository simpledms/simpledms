package browse

import (
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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

	data, uploadedFile, err := qq.readUpload(req)
	if err != nil {
		return err
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
	parentDir := qq.infra.FileRepo.GetX(ctx, data.ParentDirID)
	_, err = filesystem.NewFileIngestionService(qq.infra.FileSystem()).Ingest(
		ctx.SpaceCtx(),
		uploadedFile.Reader,
		filename,
		parentDir.Data.ID,
		data.AddToInbox,
		filesource.WebInterface,
		uploadedFile.ExpectedBytes,
		nil,
	)
	if err != nil {
		return err
	}

	rw.AddRenderables(widget.NewSnackbarf("«%s» uploaded.", filename))
	// TODO does triggering event have an effect? request comes from uppy and isn't a HTMX request...
	rw.Header().Add("HX-Trigger", event.FileUploaded.String())

	return nil
}

func (qq *UploadFileCmd) readUpload(
	req *httpx.Request,
) (*UploadFileCmdData, *uploadx.MultipartFile, error) {
	reader, err := req.MultipartReader()
	if err != nil {
		return nil, nil, err
	}
	data := &UploadFileCmdData{}
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

func (qq *UploadFileCmd) readUploadPart(
	data *UploadFileCmdData,
	part *multipart.Part,
) (*uploadx.MultipartFile, error) {
	switch part.FormName() {
	case "ParentDirID":
		value, err := readMultipartValue(part, 1024)
		data.ParentDirID = value
		return nil, err
	case "Filename":
		value, err := readMultipartValue(part, 4096)
		data.Filename = value
		return nil, err
	case "AddToInbox":
		value, err := readMultipartValue(part, 16)
		value = strings.TrimSpace(value)
		data.AddToInbox = value == "on" || value == "true" || value == "1"
		return nil, err
	case "File":
		if data.ParentDirID == "" {
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

func readMultipartValue(part *multipart.Part, maxBytes int64) (string, error) {
	value, err := io.ReadAll(io.LimitReader(part, maxBytes))
	_ = part.Close()
	return string(value), err
}
