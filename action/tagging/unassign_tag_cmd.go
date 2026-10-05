package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type UnassignTagCmdData struct {
	FileID string
	TagID  int64
}

type UnassignTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUnassignTagCmd(infra *common.Infra, actions *Actions) *UnassignTagCmd {
	config := actionx.NewConfig(
		actions.Route("unassign-tag-cmd"),
		false,
	).EnableCommittedResponse()
	return &UnassignTagCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *UnassignTagCmd) Data(fileID string, tagID int64) *UnassignTagCmdData {
	return &UnassignTagCmdData{
		FileID: fileID,
		TagID:  tagID,
	}
}

func (qq *UnassignTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UnassignTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	tag, err := taggingmodel.NewTagService().UnassignFromFile(ctx, filex.Data.ID, data.TagID)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.TagUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("«%s» unassigned.", tag.Name))
	return nil
}
