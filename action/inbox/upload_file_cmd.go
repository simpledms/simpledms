package inbox

// package action

import (
	"io"
	"log"
	"net/http"

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
	"github.com/simpledms/simpledms/util/txx"
	"github.com/simpledms/simpledms/util/uploadx"
)

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
	formHelper := autil.NewFormHelperX[UploadFileCmdData](
		infra,
		config,
		widget.T("Upload file"),
		widget.T("Upload"),
	)
	formHelper.SetIsMultipartFormData(true)
	return &UploadFileCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
		// inboxDirInfo: infra.Factory().InboxDirInfo(),
	}
}

func (qq *UploadFileCmd) Data() *UploadFileCmdData {
	return &UploadFileCmdData{}
}

func (qq *UploadFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	nilableUploadLimitBytes, err := qq.infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		return err
	}
	uploadx.LimitMultipartBody(rw, req.Request, nilableUploadLimitBytes)

	uploadedFile, err := qq.readUploadedFile(req)
	if err != nil {
		return err
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
	rootID, err := txx.WithTenantReadSpaceTx(ctx.SpaceCtx(),
		func(readCtx *ctxx.SpaceContext) (int64, error) {
			return readCtx.SpaceRootDir().ID, nil
		})
	if err != nil {
		return err
	}
	_, err = filesystem.NewFileIngestionService(qq.infra.FileSystem()).Ingest(
		ctx.SpaceCtx(),
		uploadedFile.Reader,
		filename,
		rootID,
		true,
		filesource.WebInterface,
		uploadedFile.ExpectedBytes,
		nil,
	)
	if err != nil {
		return err
	}
	// The ingestion service commits its manually managed writes before notifying the browser.
	rw.Header().Set("HX-Trigger", event.FileUploaded.String())
	rw.AddRenderables(widget.NewSnackbarf("«%s» uploaded.", filename))
	return nil
}

func (qq *UploadFileCmd) readUploadedFile(req *httpx.Request) (*uploadx.MultipartFile, error) {
	reader, err := req.MultipartReader()
	if err != nil {
		return nil, err
	}
	var uploadedFile *uploadx.MultipartFile
	for uploadedFile == nil {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.FormName() != "File" {
			_ = part.Close()
			continue
		}
		uploadedFile, err = uploadx.NewMultipartFile(part)
		if err != nil {
			_ = part.Close()
			return nil, err
		}
	}
	return uploadedFile, nil
}
