package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
			actions.Route("unassign-sub-tag"),
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

func (qq *UnassignSubTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UnassignSubTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	superTag := ctx.TenantCtx().TTx.
		Tag.
		UpdateOneID(data.SuperTagID).
		RemoveSubTagIDs(data.SubTagID).
		SaveX(ctx)

	subTag := ctx.TenantCtx().TTx.
		Tag.
		Query().
		WithChildren(
			func(query *enttenant.TagQuery) {
				query.Order(tag.ByName())
				query.Where(tag.TypeNEQ(tagtype.Super))
			},
		).
		Where(tag.ID(data.SubTagID)).
		OnlyX(ctx)

	rw.Header().Set("HX-Trigger", event.SuperTagUpdated.String(superTag.ID))

	qq.infra.Renderer().RenderX(
		rw,
		ctx,
		qq.actions.SubTags.Edit.ListItem(ctx, superTag, subTag),
		wx.NewSnackbarf("«%s» unassigned.", subTag.Name),
	)
	return nil
}
