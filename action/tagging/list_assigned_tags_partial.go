package tagging

import (
	"context"
	"fmt"
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/tagging"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ListAssignedTagsPartialData struct {
	FileID string
	// TODO enum / const? // TODO abstract config in struct?
	// TODO or as url param like `wrapper`?
	// later stuff like filters might added
	Layout string
}

// TODO or ListTagAssignments or ListAssignedTagsPartial or ListAssignedTagsPartial?
type ListAssignedTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListAssignedTagsPartial(infra *common.Infra, actions *Actions) *ListAssignedTagsPartial {
	return &ListAssignedTagsPartial{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("list-assigned-tags-partial"),
			true,
		),
	}
}

func (qq *ListAssignedTagsPartial) Data(fileID string) *ListAssignedTagsPartialData {
	return &ListAssignedTagsPartialData{
		FileID: fileID,
	}
}

func (qq *ListAssignedTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListAssignedTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	var widget renderable.Renderable
	tags := qq.tags(ctx, data)

	if data.Layout == "chips" {
		widget = qq.Chips(ctx, data, tags)
	} else if data.Layout == "list" || data.Layout == "" {
		hxTarget := req.URL.Query().Get("hx-target")
		widget = qq.List(ctx, data.FileID, tags, hxTarget)
	} else {
		log.Println("layout not supported, was", data.Layout)
		return e.NewHTTPErrorf(http.StatusBadRequest, "layout not supported")
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		widget,
	)
}

func (qq *ListAssignedTagsPartial) tags(ctx ctxx.Context, data *ListAssignedTagsPartialData) []*enttenant.Tag {
	return ctx.TenantCtx().TTx.File.Query().
		Where(file.PublicID(entx.NewCIText(data.FileID))).
		QueryTags().
		WithGroup().
		WithSubTags(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		}).
		Order(tag.ByName()).
		AllX(ctx)

}

func (qq *ListAssignedTagsPartial) ListView(
	ctx ctxx.Context,
	data *ListAssignedTagsPartialData,
) *wx.ScrollableContent {
	tags := qq.tags(ctx, data)
	if len(tags) == 0 {
		return qq.actions.AssignedTags.Edit.ListView(
			ctx,
			qq.actions.AssignedTags.Edit.Data(data.FileID, 0),
		)
	}
	return qq.List(ctx, data.FileID, tags, "#"+qq.actions.AssignedTags.Edit.hxTargetID())
}

func (qq *ListAssignedTagsPartial) List(
	ctx ctxx.Context,
	fileID string,
	tags []*enttenant.Tag,
	hxTarget string,
) *wx.ScrollableContent {
	var listItems []*wx.ListItem

	// TODO edit Attribute?

	for _, tagx := range tags {
		listItems = append(listItems, qq.actions.AssignedTags.ListItem.Widget(ctx, tagx))
	}

	bottomAppBar := &wx.BottomAppBar{
		Actions: []wx.IWidget{
			&wx.IconButton{
				Icon:    "edit_square",
				Tooltip: wx.T("Edit assigned tags"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.actions.AssignedTags.Edit.EndpointWithParams(actionx.ResponseWrapperNone, hxTarget),
					HxVals: util.JSON(qq.actions.AssignedTags.Edit.Data(fileID, 0)),
					// TODO is this a good idea? or try to select closest tab?
					HxTarget: hxTarget,
					HxSwap:   "outerHTML",
				},
			},
		},
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.actions.AssignedTags.Edit.hxTargetID(),
		},
		Children: &wx.List{
			Children: listItems,
		},
		BottomAppBar: bottomAppBar,
	}
}

func (qq *ListAssignedTagsPartial) Chips(
	ctx context.Context,
	data *ListAssignedTagsPartialData,
	tags []*enttenant.Tag,
) *wx.Container {
	var bottomSheetChildren []wx.IWidget
	for _, tagx := range tags {
		chipID := autil.GenerateID("tag-chip")

		bottomSheetChildren = append(bottomSheetChildren, &wx.Chip{
			Widget: wx.Widget[wx.Chip]{
				ID: chipID,
			},
			Label: wx.T(tagging.NewTag(tagx).String()),
			Trailing: (&wx.Button{
				Icon: wx.NewIcon("close"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.actions.AssignedTags.UnassignTagCmd.Endpoint(),
					HxVals:   util.JSON(qq.actions.AssignedTags.UnassignTagCmd.Data(data.FileID, tagx.ID)),
					HxTarget: "#" + chipID,
				},
			}).Small(),
		})
	}
	/* TODO
	bottomSheetChildren = append(
		bottomSheetChildren,
		qq.actions.EditAssignedTagsPartial.ModalLink(
			&EditAssignedTagsPartialData{
				FileID: data.FileID,
			},
			&Button{
				Label:   NewText("Assign tags"),
				Icon:    NewIcon("new_label"),
				IsSmall: true,
			},
			"",
		))

	*/

	id := autil.GenerateID("assignedTagsList")
	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: id,
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:    qq.Config.Endpoint(),
			HxTrigger: fmt.Sprintf("%s from:body", event.TagUpdated.String()),
			HxVals:    util.JSON(data),
			HxTarget:  "#" + id,
		},
		// TODO morph / impl as default?

		Child: bottomSheetChildren,
	}
}
