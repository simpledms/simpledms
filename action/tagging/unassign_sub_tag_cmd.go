package tagging

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type UnassignSubTagCmdData struct {
	SuperTagID int64
	SubTagID   int64
}

type UnassignSubTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUnassignSubTagCmd(infra *common.Infra, actions *Actions) *UnassignSubTagCmd {
	return &UnassignSubTagCmd{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("unassign-sub-tag-cmd"),
			false,
		),
	}
}

func (qq *UnassignSubTagCmd) Data(superTagID int64, subTagID int64) *UnassignSubTagCmdData {
	return &UnassignSubTagCmdData{
		SuperTagID: superTagID,
		SubTagID:   subTagID,
	}
}

func (qq *UnassignSubTagCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UnassignSubTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	superTag, subTag, err := taggingmodel.NewTagService().UnassignSubTag(
		ctx,
		data.SuperTagID,
		data.SubTagID,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.SuperTagUpdated.String(superTag.ID))

	qq.infra.Renderer().RenderX(
		rw,
		ctx,
		qq.actions.SubTags.Edit.ListItem(ctx, superTag, subTag),
		wx.NewSnackbarf("«%s» unassigned.", subTag.Name),
	)
	return nil
}
