package browse

import (
	"fmt"
	"log"
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/uix/events"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
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
	actions  *Actions
	fileRepo filemodel.FileRepository
	service  *filemodel.FileVersionFromInboxService
	*actionx.Config
}

func NewFileVersionFromInboxCmd(infra *common.Infra, actions *Actions) *FileVersionFromInboxCmd {
	config := actionx.NewConfig(actions.Route("file-version-from-inbox-cmd"), false)
	return &FileVersionFromInboxCmd{
		actions:  actions,
		fileRepo: infra.FileRepo,
		service:  filemodel.NewFileVersionFromInboxService(),
		Config:   config,
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

func (qq *FileVersionFromInboxCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	sourceFile, err := qq.actions.FileVersionFromInboxListPartial.findInboxFile(ctx, data.SourceFileID)
	if err != nil {
		log.Println(err)
		return err
	}

	targetFile := qq.fileRepo.GetX(ctx, data.TargetFileID)

	_, err = qq.service.MergeFromInbox(ctx, sourceFile, targetFile.Data)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.AddRenderables(
		wx.NewSnackbarf("Added new version from inbox."),
	)

	rw.Header().Set("HX-Trigger", fmt.Sprintf("%s, %s, %s, %s", event.FileUploaded.String(), event.FileUpdated.String(), event.FileDeleted.String(), events.CloseDialog.String()))

	return nil
}
