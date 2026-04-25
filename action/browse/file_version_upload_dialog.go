package browse

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type FileVersionUploadDialogData struct {
	FileID string
}

type FileVersionUploadDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersionUploadDialog(infra *common.Infra, actions *Actions) *FileVersionUploadDialog {
	config := actionx.NewConfig(actions.Route("file-version-upload-dialog"), true)
	return &FileVersionUploadDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionUploadDialog) Data(fileID string) *FileVersionUploadDialogData {
	return &FileVersionUploadDialogData{
		FileID: fileID,
	}
}

func (qq *FileVersionUploadDialog) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionUploadDialogData](rw, req, ctx)
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
			Headline:     widget.T("Upload new version"),
			IsOpenOnLoad: true,
			Child: &widget.FileUpload{
				Endpoint:           qq.actions.UploadFileVersionCmd.Endpoint(),
				FileID:             data.FileID,
				MaxUploadSizeBytes: maxUploadSizeBytes,
			},
		},
	)
}
