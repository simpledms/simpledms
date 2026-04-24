package browse

import (
	"fmt"
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionFromInboxFormData struct {
	TargetFileID string `form_attr_type:"hidden"`
	SourceFileID string `form_attr_type:"hidden"`
}

type FileVersionFromInboxCmdData struct {
	FileVersionFromInboxFormData `structs:",flatten"`
	ConfirmWarning               bool
}

type FileVersionFromInboxCmd struct {
	actions *Actions
	service *filemodel.FileVersionFromInboxService
	*actionx.Config
}

func NewFileVersionFromInboxCmd(_ *common.Infra, actions *Actions) *FileVersionFromInboxCmd {
	config := actionx.NewConfig(actions.Route("file-version-from-inbox-cmd"), false)
	return &FileVersionFromInboxCmd{
		actions: actions,
		service: filemodel.NewFileVersionFromInboxService(),
		Config:  config,
	}
}

func (qq *FileVersionFromInboxCmd) Data(targetFileID, sourceFileID string) *FileVersionFromInboxCmdData {
	return &FileVersionFromInboxCmdData{
		FileVersionFromInboxFormData: FileVersionFromInboxFormData{
			TargetFileID: targetFileID,
			SourceFileID: sourceFileID,
		},
	}
}

func (qq *FileVersionFromInboxCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionFromInboxCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if !data.ConfirmWarning {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Please confirm that the source file metadata will be lost.")
	}

	if data.TargetFileID == "" || data.SourceFileID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
	}

	_, err = qq.service.MergeFromInbox(ctx, data.SourceFileID, data.TargetFileID)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(
		wx.NewSnackbarf("Added new version from inbox."),
	)

	rw.Header().Set("HX-Trigger", fmt.Sprintf("%s, %s, %s, %s", event.FileUploaded.String(), event.FileUpdated.String(), event.FileDeleted.String(), event.CloseDialog.String()))

	return nil
}
