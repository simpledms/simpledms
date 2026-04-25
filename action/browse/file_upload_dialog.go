package browse

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
)

type FileUploadDialogData struct {
	ParentDirID string
	AddToInbox  bool
}

type FileUploadDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileUploadDialog(infra *common.Infra, actions *Actions) *FileUploadDialog {
	config := actionx.NewConfig(actions.Route("file-upload-dialog"), true)
	return &FileUploadDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileUploadDialog) Data(parentDirID string, addToInbox bool) *FileUploadDialogData {
	return &FileUploadDialogData{
		ParentDirID: parentDirID,
		AddToInbox:  addToInbox,
	}
}

func (qq *FileUploadDialog) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileUploadDialogData](rw, req, ctx)
	if err != nil {
		return err
	}

	maxUploadSizeBytes := int64(0)
	nilableUploadLimitBytes, err := qq.infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		return err
	}
	if nilableUploadLimitBytes != nil {
		maxUploadSizeBytes = *nilableUploadLimitBytes
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,

		&widget.Dialog{
			Layout:       widget.DialogLayoutStable,
			Headline:     widget.T("File upload"),
			IsOpenOnLoad: true,
			Child: &widget.FileUpload{
				ParentDirID:        data.ParentDirID,
				AddToInbox:         data.AddToInbox,
				MaxUploadSizeBytes: maxUploadSizeBytes,
			},
		},
	)
}
