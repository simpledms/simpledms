package tagging

// package action

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type MoveTagToGroupCmdData struct {
	TagID      int64 `form_attr_type:"hidden"`
	GroupTagID int64 `form_attr_type:"hidden"`
}

type MoveTagToGroupCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
	*autil.FormHelper[MoveTagToGroupCmdData]
}

func NewMoveTagToGroupCmd(infra *common.Infra, actions *Actions) *MoveTagToGroupCmd {
	config := actionx2.NewConfig(
		actions.Route("move-tag-to-group-cmd"),
		false,
	)
	return &MoveTagToGroupCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[MoveTagToGroupCmdData](
			infra,
			config,
			widget.T("Move tag to group"),
		),
	}
}

func (qq *MoveTagToGroupCmd) Data(tagID int64, groupTagID int64) *MoveTagToGroupCmdData {
	return &MoveTagToGroupCmdData{
		TagID:      tagID,
		GroupTagID: groupTagID,
	}
}

func (qq *MoveTagToGroupCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MoveTagToGroupCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	var snackbar *widget.Snackbar

	isDeselected, groupTag, err := taggingmodel.NewTagService().MoveToGroup(ctx, data.TagID, data.GroupTagID)
	if err != nil {
		return err
	}

	if isDeselected {
		snackbar = widget.NewSnackbarf("Deselected group.")
	} else {
		snackbar = widget.NewSnackbarf("Moved to group «%s».", groupTag.Name)
	}

	// TODO group ID or tag ID?
	// rw.Header().Set("HX-Trigger", event.TagMovedToGroup.String(data.TagID))

	// not sure why necessary with HX-Reswap=none, but doesn't work without it
	rw.Header().Set("HX-Trigger-After-Swap", event.TagUpdated.String())
	rw.Header().Add("HX-Reswap", "none")
	rw.AddRenderables(snackbar)

	return nil
}

func (qq *MoveTagToGroupCmd) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MoveTagToGroupCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	tagService := taggingmodel.NewTagService()

	groupTags, err := tagService.GroupTags(ctx)
	if err != nil {
		return err
	}

	tag, err := tagService.Get(ctx, data.TagID)
	if err != nil {
		return err
	}

	hxTarget := req.URL.Query().Get("hx-target")

	var listItems []*widget.ListItem

	if tag.GroupID > 0 {
		listItems = append(listItems, &widget.ListItem{
			Headline: widget.T("Deselect group"),
			Type:     widget.ListItemTypeHelper,
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(data.TagID, 0)),
				HxOn:     events.CloseDialog.HxOn("click"),
				HxTarget: hxTarget,
			},
		})
	}

	for _, groupTag := range groupTags {
		if groupTag.ID == tag.GroupID {
			continue
		}
		listItems = append(listItems, &widget.ListItem{
			Headline: widget.Tu(groupTag.Name),

			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(data.TagID, groupTag.ID)),
				HxOn:     events.CloseDialog.HxOn("click"),
				HxTarget: hxTarget,
			},
		})
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		autil.WrapWidget(
			widget.T("Move tag to group"),
			nil,
			&widget.List{
				Children: listItems,
			},
			actionx2.ResponseWrapperDialog,
			widget.DialogLayoutDefault,
		),
	)
}
