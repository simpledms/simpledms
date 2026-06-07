package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AssignSubTagCmdData struct {
	SuperTagID int64
	SubTagID   int64
}

type AssignSubTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAssignSubTagCmd(infra *common.Infra, actions *Actions) *AssignSubTagCmd {
	return &AssignSubTagCmd{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("assign-sub-tag-cmd"),
			false, // TODO is this correct?
		),
	}
}

func (qq *AssignSubTagCmd) Data(superTagID int64, subTagID int64) *AssignSubTagCmdData {
	return &AssignSubTagCmdData{
		SuperTagID: superTagID,
		SubTagID:   subTagID,
	}
}

func (qq *AssignSubTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignSubTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	superTag, subTag, err := taggingmodel.NewTagService().AssignSubTag(
		ctx,
		data.SuperTagID,
		data.SubTagID,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.SuperTagUpdated.String(superTag.ID))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.SubTags.Edit.ListItem(ctx, superTag, subTag),
		wx.NewSnackbarf("«%s» assigned.", subTag.Name),
	)
}
