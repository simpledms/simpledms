package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *FileVersionUploadDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionUploadDialogData](rw, req, ctx)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(
		rw,
		ctx,
		&wx.Dialog{
			Layout:       wx.DialogLayoutStable,
			Headline:     wx.T("Upload new version"),
			IsOpenOnLoad: true,
			Child: &wx.FileUpload{
				Endpoint: qq.actions.UploadFileVersionCmd.Endpoint(),
				FileID:   data.FileID,
			},
		},
	)
}
