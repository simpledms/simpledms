package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileUploadDialogPartialData struct {
	ParentDirID string
	AddToInbox  bool
}

type FileUploadDialogPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileUploadDialogPartial(infra *common.Infra, actions *Actions) *FileUploadDialogPartial {
	config := actionx.NewConfig(actions.Route("file-upload-dialog"), true)
	return &FileUploadDialogPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileUploadDialogPartial) Data(parentDirID string, addToInbox bool) *FileUploadDialogPartialData {
	return &FileUploadDialogPartialData{
		ParentDirID: parentDirID,
		AddToInbox:  addToInbox,
	}
}

func (qq *FileUploadDialogPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileUploadDialogPartialData](rw, req, ctx)
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
