package tagging

import (
	"context"
	"fmt"
	"html/template"
	"slices"

	"github.com/google/uuid"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditSubTagsPartialData struct {
	TagID        int64
	OnlyAssigned bool
}

// TODO or EditComposedSubTags or EditSubTagsOfSuperTag?
//
//	Only composed tags can have subtags, thus should be clear?!
type EditSubTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewEditSubTagsPartial(
	infra *common.Infra,
	actions *Actions,
) *EditSubTagsPartial {
	return &EditSubTagsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("edit-sub-tags-partial"),
			true, // TODO is this correct?
		),
	}
}

func (qq *EditSubTagsPartial) Data(tagID int64, onlyAssigned bool) *EditSubTagsPartialData {
	return &EditSubTagsPartialData{
		TagID:        tagID,
		OnlyAssigned: onlyAssigned,
	}
}

func (qq *EditSubTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditSubTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	superTag := ctx.TenantCtx().TTx.Tag.
		Query().
		Where(tag.ID(data.TagID)).
		OnlyX(ctx)

	// subTags := superTag.QuerySubTags().WithGroup().AllX(ctx)

	assignableListItems := qq.assignableListItems(ctx, superTag)
	// assignedSubTagsListView := qq.actions.SubTags.List.Widget(subTags)

	list := &wx.List{
		Children: assignableListItems,
	}

	/*
		bottomAppBar := &BottomAppBar{
			Actions: []IWidget{
				&Button{
					Icon: NewIcon("toggle_on"),
					HTMXAttrs: HTMXAttrs{
						HxPost:   qq.EndpointWithParams(actionx.ResponseWrapperNone, ""),
						HxVals:   util.JSON(qq.Data(data.TagID, !data.OnlyAssigned)),
						HxTarget: "#" + list.GetID(),
					},
				},
			},
		}
	*/

	wrapper := req.URL.Query().Get("wrapper")
	qq.infra.Renderer().RenderX(
		rw,
		ctx,
		autil.WrapWidget(
			// TODO indicate that composed tag, is there a subheader?
			wx.Tf(
				"Tags of «%s»", // of or for?
				superTag.Name,
			),
			nil,
			list,
			actionx.ResponseWrapper(wrapper),
			wx.DialogLayoutStable,
		),
	)
	return nil
}

func (qq *EditSubTagsPartial) assignableListItems(
	ctx ctxx.Context,
	superTag *enttenant.Tag,
) []*wx.ListItem {
	assignableTags := ctx.TenantCtx().TTx.
		Tag.Query().
		Where(
			tag.Not(tag.HasGroup()),
			tag.TypeNEQ(tagtype.Super),
		).
		Order(tag.ByName()).
		WithChildren(
			func(query *enttenant.TagQuery) {
				query.Order(tag.ByName())
				query.Where(tag.TypeNEQ(tagtype.Super))
			},
		).
		AllX(ctx)

	var allListItems []*wx.ListItem
	var tagListItems []*wx.ListItem
	var groupListItems []*wx.ListItem

	isCheckedFn := qq.isCheckedFn(ctx, superTag)
	for _, assignableTag := range assignableTags {
		listItem := qq.listItem(ctx, superTag, assignableTag, isCheckedFn)

		if assignableTag.Type == tagtype.Group {
			if len(assignableTag.Edges.Children) == 0 {
				// don't show empty groups
				continue
			}
			groupListItems = append(groupListItems, listItem)
		} else {
			tagListItems = append(tagListItems, listItem)
		}
	}

	allListItems = append(allListItems, groupListItems...)
	allListItems = append(allListItems, tagListItems...)

	return allListItems
}

// error prone implementation, may lead to multiple calls to isCheckedFn if not
// used with care; // TODO maybe use a View instead?
func (qq *EditSubTagsPartial) isCheckedFn(
	ctx ctxx.Context,
	superTag *enttenant.Tag,
) func(subTagID int64) bool {
	// always loads all sub tags, not optimal, but should be acceptable,
	// because there is just a small number...
	subTags := ctx.TenantCtx().TTx.
		Tag.
		QuerySubTags(superTag).
		AllX(ctx)
	return func(subTagID int64) bool {
		return slices.ContainsFunc(
			subTags,
			func(subTag *enttenant.Tag) bool {
				return subTag.ID == subTagID
			},
		)
	}
}

func (qq *EditSubTagsPartial) ListItem(
	ctx ctxx.Context,
	superTag *enttenant.Tag,
	subTagWithChildren *enttenant.Tag,
) *wx.ListItem {
	return qq.listItem(
		ctx,
		superTag,
		subTagWithChildren,
		qq.isCheckedFn(
			ctx,
			superTag,
		),
	)
}

func (qq *EditSubTagsPartial) listItem(
	ctx context.Context,
	superTag *enttenant.Tag,
	subTagWithChildren *enttenant.Tag,
	isCheckedFn func(tagID int64) bool,
) *wx.ListItem {
	var hxPost string
	var hxVals template.JS
	if isCheckedFn(subTagWithChildren.ID) {
		hxPost = qq.actions.SubTags.UnassignSubTagCmd.Endpoint()
		hxVals = util.JSON(qq.actions.SubTags.UnassignSubTagCmd.Data(superTag.ID, subTagWithChildren.ID))
	} else {
		hxPost = qq.actions.SubTags.AssignSubTagCmd.Endpoint()
		hxVals = util.JSON(qq.actions.SubTags.AssignSubTagCmd.Data(superTag.ID, subTagWithChildren.ID))
	}

	id := qq.listItemID(
		superTag.ID,
		subTagWithChildren.ID,
	)

	var icon *wx.Icon
	var supportingText string
	var trailing wx.IWidget
	var isCollapsible bool

	var childItems []wx.IWidget

	if subTagWithChildren.Type == tagtype.Group {
		// TODO find something betteer
		// folder_special
		// note_stack
		icon = wx.NewIcon("folder_special")

		childTagsStr := fmt.Sprintf(
			"%d child tag",
			len(subTagWithChildren.Edges.Children),
		)
		if len(subTagWithChildren.Edges.Children) > 1 || len(subTagWithChildren.Edges.Children) == 0 {
			childTagsStr += "s"
		}

		selectedCount := 0
		for _, childTag := range subTagWithChildren.Edges.Children {
			if isCheckedFn(childTag.ID) {
				selectedCount++
			}
		}
		// TODO doesn't get updated on change; impl a web component?
		// TODO selected can be better indicated by color of icon or a badge?
		selectedStr := fmt.Sprintf(
			"%d selected",
			selectedCount,
		)

		// TODO indicate if children are checked with checkbox? via bg color?
		supportingText = fmt.Sprintf(
			"Group, %s, %s",
			childTagsStr,
			selectedStr,
		)
		trailing = wx.NewIcon("keyboard_arrow_down")

		// children are eagerly loaded
		for _, childTag := range subTagWithChildren.Edges.Children {
			// TODO fix is checked
			childItems = append(
				childItems,
				qq.listItem(
					ctx,
					superTag,
					childTag,
					isCheckedFn,
				),
			)
		}

		isCollapsible = true
	} else {
		icon = wx.NewIcon("label")
		trailing = &wx.Checkbox{
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:    hxPost,
				HxTrigger: "change",
				HxVals:    hxVals,
				HxTarget:  "#" + id,
				HxSwap:    "outerHTML",
			},
			IsChecked: isCheckedFn(subTagWithChildren.ID),
		}
	}

	htmxAttrs := wx.HTMXAttrs{}

	if subTagWithChildren.Type == tagtype.Simple {
		// TODO should link complete listItem, not content and trailing separatly,
		//		results in a small gap because if margin
		//
		// impl on refactoring on 27.10.24, nur sure if correct, would solve comment above
		htmxAttrs = wx.HTMXAttrs{
			HxPost:   hxPost,
			HxVals:   hxVals,
			HxTarget: "#" + id,
			HxSwap:   "outerHTML",
		}
	}

	return &wx.ListItem{
		Widget: wx.Widget[wx.ListItem]{
			ID: id,
		},
		HTMXAttrs:      htmxAttrs, // TODO or ContentOnly?
		Leading:        icon,
		Headline:       wx.T(subTagWithChildren.Name),
		SupportingText: wx.Tu(supportingText),
		Trailing:       trailing,
		IsCollapsible:  isCollapsible,
		Child:          childItems,
	}
}

func (qq *EditSubTagsPartial) listItemID(superTagID, subTag int64) string {
	return fmt.Sprintf(
		"listItemSubTag-%d-%d-%s",
		superTagID,
		subTag,
		uuid.NewString(),
	)
}
