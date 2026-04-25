package managetags

import (
	"fmt"
	"slices"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type TagListPartialData struct {
	ParentTagID int64 // to load group children
}

type TagListPartialState struct {
	// IMPORTANT URL name used in code below
	ExpandedGroups []int64 `url:"expanded_groups,omitempty"` // TODO or ExpandedTags?
}

type TagListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewTagListPartial(infra *common.Infra, actions *Actions) *TagListPartial {
	config := actionx.NewConfig("tag-list", true)
	return &TagListPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *TagListPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[TagListPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[TagListPartialState](rw, req)

	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, data, state))
}

func (qq *TagListPartial) Data(parentTagID int64) *TagListPartialData {
	return &TagListPartialData{
		ParentTagID: parentTagID,
	}
}

func (qq *TagListPartial) Widget(ctx ctxx.Context, data *TagListPartialData, state *TagListPartialState) *widget.List {
	// duplicate in EditAssignedTags.ListView
	// TODO is this necessary? children are eagerly loaded... maybe just necessary in EditAssignedTags
	//		because of checkboxes?
	isLoadingPartial := data.ParentTagID > 0

	tagsQuery := ctx.SpaceCtx().Space.QueryTags().
		Order(tag.ByName()).
		WithChildren(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		}).
		WithSubTags(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		})
	if isLoadingPartial {
		tagsQuery = tagsQuery.Where(tag.HasGroupWith(tag.ID(data.ParentTagID)))
	} else {
		// load just top layer into collection, but eagerly load children
		// for more efficient processing
		tagsQuery = tagsQuery.Where(tag.Not(tag.HasGroup()))
	}
	tags := tagsQuery.AllX(ctx)

	var allListItems []*widget.ListItem
	var tagListItems []*widget.ListItem
	var groupListItems []*widget.ListItem

	if !isLoadingPartial {
		// TODO or as FAB? would be more consistent with rest of application
		//		but less consistent with adding tags to group
		allListItems = append(allListItems,
			// TODO segment into two list items?
			&widget.ListItem{
				Headline: widget.T("Create new tag or group"),
				Leading:  widget.NewIcon("new_label"),
				Type:     widget.ListItemTypeHelper,
				HTMXAttrs: qq.actions.Tagging.CreateTagCmd.ModalLinkAttrs(
					qq.actions.Tagging.CreateTagCmd.Data(0), ""),
			},
		)
	}

	for _, tagx := range tags {
		listItem := qq.listItem(ctx, state, tagx)
		if tagx.Type == tagtype.Group {
			groupListItems = append(groupListItems, listItem)
		} else {
			tagListItems = append(tagListItems, listItem)
		}
	}

	allListItems = append(allListItems, groupListItems...)
	allListItems = append(allListItems, tagListItems...)

	if isLoadingPartial {
		return &widget.List{
			Children: allListItems,
		}
	}

	// TODO empty state? or just add a list item?

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.id(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: "#tagList",
			HxSwap:   "outerHTML",
			// TODO currently loads all tags always, but could be limited to a tag group
			//		if a child tag is created
			HxTrigger: events.HxTrigger(
				event.TagCreated,
				event.TagUpdated,
				event.TagDeleted,
				// event.TagMovedToGroup.
			),
		},
		Children: allListItems,
	}
}

func (qq *TagListPartial) listItem(
	ctx ctxx.Context,
	state *TagListPartialState,
	tagx *enttenant.Tag,
) *widget.ListItem {
	var icon *widget.Icon
	var supportingText *widget.Text
	var trailing widget.IWidget
	// var radioGroupName string
	var isCollapsible bool
	var isOpen bool
	var childItems []widget.IWidget
	var htmxAttrs widget.HTMXAttrs

	if tagx.Type == tagtype.Group {
		icon = widget.NewIcon("folder_special")
		// TODO prefetch or via view?
		childCount := tagx.QueryChildren().CountX(ctx)

		childTagsStr := widget.Tf("Group, %d tag", childCount)
		if childCount > 1 || childCount == 0 {
			childTagsStr = widget.Tf("Group, %d tags", childCount)
		}
		supportingText = childTagsStr
		// radioGroupName = qq.id() + "RadioGroup"
		childItems = append(childItems, &widget.ListItem{
			Type:     widget.ListItemTypeHelper,
			Leading:  widget.NewIcon("new_label"),
			Headline: widget.T("Create new tag"), // group not possible
			HTMXAttrs: qq.actions.Tagging.CreateTagCmd.ModalLinkAttrs(
				qq.actions.Tagging.CreateTagCmd.Data(tagx.ID),
				"",
			),
		})

		isCollapsible = true

		isOpen = slices.Contains(state.ExpandedGroups, tagx.ID)
		if isOpen {
			// children are eagerly loaded
			// TODO why is there partial load support then?
			for _, childTag := range tagx.Edges.Children {
				childItems = append(
					childItems,
					qq.listItem(ctx, state, childTag),
				)
			}
			// necessary for replacing close/open icon indicator and for workaround
			// for a idiomorph issue where the old request is still fired if not overwriten
			htmxAttrs = widget.HTMXAttrs{
				HxPost:   qq.actions.ToggleTagGroupCmd.Endpoint(),
				HxVals:   util.JSON(qq.actions.ToggleTagGroupCmd.Data(tagx.ID)),
				HxTarget: "#" + qq.listItemID(tagx.ID),
				HxSwap:   "outerHTML",
				/*HxOn: event.UnsafeHxOnQueryParamDeleteFromSlice(
					"click",
					"expanded_groups",
					fmt.Sprintf("%d", tagx.ID),
				),*/
			}
			trailing = widget.NewIcon("keyboard_arrow_up")
		} else {
			htmxAttrs = widget.HTMXAttrs{
				// just replace because eagerly loaded
				// HxPost:    route.ManageTagsWithState(state)(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
				HxPost:   qq.actions.ToggleTagGroupCmd.Endpoint(),
				HxVals:   util.JSON(qq.actions.ToggleTagGroupCmd.Data(tagx.ID)),
				HxTarget: "#" + qq.listItemID(tagx.ID),
				HxSwap:   "outerHTML",
			}
			trailing = widget.NewIcon("keyboard_arrow_down")
		}
	} else if tagx.Type == tagtype.Super {
		icon = widget.NewIcon("label_important")

		supportingText = widget.T("Super tag")
		// TODO show number of subTags?

		// trailing = wx.NewIcon("keyboard_arrow_right")
		// radioGroupName = qq.id() + "RadioGroup"
	} else {
		icon = widget.NewIcon("label")
	}

	return &widget.ListItem{
		Widget: widget.Widget[widget.ListItem]{
			ID: qq.listItemID(tagx.ID),
		},
		HTMXAttrs: htmxAttrs,
		// RadioGroupName: radioGroupName,
		/*HTMXAttrs: wx.HTMXAttrs{
			HxTarget: "#details",
			HxSwap:   "outerHTML",
			HxSelect: "#details",
			HxGet:    route.ManageTag(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, tagx.ID),
		},*/
		Leading:        icon,
		Headline:       widget.Tu(tagx.Name),
		SupportingText: supportingText,
		Trailing:       trailing,
		ContextMenu:    NewTagContextMenuWidget(qq.actions).Widget(ctx, tagx),
		Child:          childItems,
		IsCollapsible:  isCollapsible,
		IsOpen:         isOpen,
	}
}

func (qq *TagListPartial) listItemID(tagID int64) string {
	return fmt.Sprintf("tagListItem-%d", tagID)
}

func (qq *TagListPartial) id() string {
	return "tagList"
}
