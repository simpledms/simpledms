package tagging

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type MoveTagToGroupCmdData struct {
	TagID      int64 `form_attr_type:"hidden"`
	GroupTagID int64 `form_attr_type:"hidden"`
}

type MoveTagToGroupCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[MoveTagToGroupCmdData]
}

func NewMoveTagToGroupCmd(infra *common.Infra, actions *Actions) *MoveTagToGroupCmd {
	config := actionx.NewConfig(
		actions.Route("move-tag-to-group"),
		false,
	)
	return &MoveTagToGroupCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[MoveTagToGroupCmdData](
			infra,
			config,
			wx.T("Move tag to group"),
		),
	}
}

func (qq *MoveTagToGroupCmd) Data(tagID int64, groupTagID int64) *MoveTagToGroupCmdData {
	return &MoveTagToGroupCmdData{
		TagID:      tagID,
		GroupTagID: groupTagID,
	}
}

func (qq *MoveTagToGroupCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MoveTagToGroupCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	var snackbar *wx.Snackbar

	if data.GroupTagID == 0 {
		ctx.TenantCtx().TTx.Tag.UpdateOneID(data.TagID).ClearGroupID().SaveX(ctx)
		snackbar = wx.NewSnackbarf("Deselected group.")
	} else {
		ctx.TenantCtx().TTx.Tag.UpdateOneID(data.TagID).SetGroupID(data.GroupTagID).SaveX(ctx)
		groupTag := ctx.TenantCtx().TTx.Tag.GetX(ctx, data.GroupTagID)
		snackbar = wx.NewSnackbarf("Moved to group «%s».", groupTag.Name)
	}

	// TODO group ID or tag ID?
	// rw.Header().Set("HX-Trigger", event.TagMovedToGroup.String(data.TagID))

	// not sure why necessary with HX-Reswap=none, but doesn't work without it
	rw.Header().Set("HX-Trigger-After-Swap", event.TagUpdated.String())
	rw.Header().Add("HX-Reswap", "none")
	rw.AddRenderables(snackbar)

	return nil
}

func (qq *MoveTagToGroupCmd) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MoveTagToGroupCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	groupTags := ctx.TenantCtx().TTx.Tag.Query().Where(tag.TypeEQ(tagtype.Group)).AllX(ctx)
	tag := ctx.TenantCtx().TTx.Tag.Query().Where(tag.ID(data.TagID)).OnlyX(ctx)

	hxTarget := req.URL.Query().Get("hx-target")

	var listItems []*wx.ListItem

	if tag.GroupID > 0 {
		listItems = append(listItems, &wx.ListItem{
			Headline: wx.T("Deselect group"),
			Type:     wx.ListItemTypeHelper,
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(data.TagID, 0)),
				HxOn:     event.CloseDialog.HxOn("click"),
				HxTarget: hxTarget,
			},
		})
	}

	for _, groupTag := range groupTags {
		if groupTag.ID == tag.GroupID {
			continue
		}
		listItems = append(listItems, &wx.ListItem{
			Headline: wx.Tu(groupTag.Name),

			HTMXAttrs: wx.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(data.TagID, groupTag.ID)),
				HxOn:     event.CloseDialog.HxOn("click"),
				HxTarget: hxTarget,
			},
		})
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		autil.WrapWidget(
			wx.T("Move tag to group"),
			nil,
			&wx.List{
				Children: listItems,
			},
			actionx.ResponseWrapperDialog,
			wx.DialogLayoutDefault,
		),
	)
}
