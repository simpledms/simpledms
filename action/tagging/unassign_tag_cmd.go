package tagging

import (
	"log"
	"strings"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
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
	)
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

func (qq *UnassignTagCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UnassignTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	hxTarget := req.Header.Get("HX-Target") // id without leading #

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	tag, err := taggingmodel.NewTagService().UnassignFromFile(ctx, filex.Data.ID, data.TagID)
	if err != nil {
		return err
	}

	if hxTarget == qq.actions.AssignedTags.EditListItem.listItemID(data.FileID, data.TagID) {
		// must be set before writing to rw
		rw.Header().Set("HX-Trigger", event.TagUpdated.String())

		qq.infra.Renderer().RenderX(
			rw,
			ctx,
			qq.actions.AssignedTags.EditListItem.ListItem(ctx, data.FileID, tag),
		)
	} else if strings.HasPrefix(hxTarget, "assignedTagsList-") {
		// rw.WriteHeader(http.StatusOK)
	} else {
		log.Println("target not found, was", hxTarget)
		// rw.WriteHeader(http.StatusNoContent)
	}

	qq.infra.Renderer().RenderX(rw, ctx, wx.NewSnackbarf("«%s» unassigned.", tag.Name))
	return nil
}
