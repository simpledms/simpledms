package tagging

import (
	"context"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/db/entx"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
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
	*actionx2.Config
}

func NewListAssignedTagsPartial(infra *common.Infra, actions *Actions) *ListAssignedTagsPartial {
	return &ListAssignedTagsPartial{
		infra,
		actions,
		actionx2.NewConfig(
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

func (qq *ListAssignedTagsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
) *widget.ScrollableContent {
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
) *widget.ScrollableContent {
	var listItems []*widget.ListItem

	// TODO edit Attribute?

	for _, tagx := range tags {
		listItems = append(listItems, qq.actions.AssignedTags.ListItem.Widget(ctx, tagx))
	}

	bottomAppBar := &widget.BottomAppBar{
		Actions: []widget.IWidget{
			&widget.IconButton{
				Icon:    "edit_square",
				Tooltip: widget.T("Edit assigned tags"),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost: qq.actions.AssignedTags.Edit.EndpointWithParams(actionx2.ResponseWrapperNone, hxTarget),
					HxVals: util.JSON(qq.actions.AssignedTags.Edit.Data(fileID, 0)),
					// TODO is this a good idea? or try to select closest tab?
					HxTarget: hxTarget,
					HxSwap:   "outerHTML",
				},
			},
		},
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.actions.AssignedTags.Edit.hxTargetID(),
		},
		Children: &widget.List{
			Children: listItems,
		},
		BottomAppBar: bottomAppBar,
	}
}

func (qq *ListAssignedTagsPartial) Chips(
	ctx context.Context,
	data *ListAssignedTagsPartialData,
	tags []*enttenant.Tag,
) *widget.Container {
	var bottomSheetChildren []widget.IWidget
	for _, tagx := range tags {
		chipID := autil.GenerateID("tag-chip")

		bottomSheetChildren = append(bottomSheetChildren, &widget.Chip{
			Widget: widget.Widget[widget.Chip]{
				ID: chipID,
			},
			Label: widget.T(tagging.NewTag(tagx).String()),
			Trailing: (&widget.Button{
				Icon: widget.NewIcon("close"),
				HTMXAttrs: widget.HTMXAttrs{
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
	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: id,
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.Config.Endpoint(),
			HxTrigger: events.HxTrigger(event.TagUpdated),
			HxVals:    util.JSON(data),
			HxTarget:  "#" + id,
		},
		// TODO morph / impl as default?

		Child: bottomSheetChildren,
	}
}
