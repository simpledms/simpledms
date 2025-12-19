package browse

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *FileUploadDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileUploadDialogData](rw, req, ctx)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(
		rw,
		ctx,

		&wx.Dialog{
			Layout:       wx.DialogLayoutStable,
			Headline:     wx.T("File upload"),
			IsOpenOnLoad: true,
			Child: &wx.FileUpload{
				ParentDirID: data.ParentDirID,
				AddToInbox:  data.AddToInbox,
			},
		},
	)
}
