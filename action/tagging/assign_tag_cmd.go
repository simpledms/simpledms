package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	taggingmodel "github.com/simpledms/simpledms/model/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AssignTagCmdData struct {
	FileID string
	TagID  int64
}

type AssignTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAssignTagCmd(
	infra *common.Infra,
	actions *Actions,
) *AssignTagCmd {
	config := actionx.NewConfig(
		actions.Route("assign-tag-cmd"),
		false,
	)
	return &AssignTagCmd{
		infra,
		actions,
		config,
	}
}

func (qq *AssignTagCmd) Data(fileID string, tagID int64) *AssignTagCmdData {
	return &AssignTagCmdData{
		FileID: fileID,
		TagID:  tagID,
	}
}

func (qq *AssignTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	tag, err := taggingmodel.NewTagService().AssignToFile(
		ctx,
		filex.Data.ID,
		data.TagID,
		ctx.SpaceCtx().Space.ID,
	)
	if err != nil {
		return err
	}

	// must be set before writing to rw
	rw.Header().Set("HX-Trigger", event.TagUpdated.String())

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.AssignedTags.EditListItem.ListItem(ctx, data.FileID, tag),
		wx.NewSnackbarf("«%s» assigned.", tag.Name),
	)
}
