package trash

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type RestoreFileCmdData struct {
	FileID string
}

type RestoreFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewRestoreFileCmd(infra *common.Infra, actions *Actions) *RestoreFileCmd {
	return &RestoreFileCmd{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("restore-file-cmd"),
			false,
		),
	}
}

func (qq *RestoreFileCmd) Data(fileID string) *RestoreFileCmdData {
	return &RestoreFileCmdData{
		FileID: fileID,
	}
}

func (qq *RestoreFileCmd) DataWithOptions(fileID string) *RestoreFileCmdData {
	return &RestoreFileCmdData{
		FileID: fileID,
	}
}

func (qq *RestoreFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[RestoreFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	result, err := repos.Write.RestoreDeletedFile(ctx, data.FileID)
	if err != nil {
		return err
	}

	if !result.ParentExists {
		rw.AddRenderables(wx.NewSnackbarf("The original parent folder is missing. Restored to Inbox."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("File restored."))
	}

	rw.Header().Set("HX-Retarget", "#details")
	rw.Header().Set("HX-Reswap", "innerHTML")
	// TODO not nice because logic to reload list and close details is implemented by handling FileRestored event
	rw.Header().Set("HX-Replace-Url", route.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))
	rw.Header().Set("HX-Trigger", event.FileRestored.String())

	return qq.infra.Renderer().Render(rw, ctx, &wx.View{})
}
