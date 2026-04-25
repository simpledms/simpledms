package tagging

import (
	"fmt"
	"html/template"
	"slices"
	"strings"

	"github.com/simpledms/simpledms/db/entx"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type EditAssignedTagsItemPartialData struct {
	FileID string
	TagID  int64
}

type EditAssignedTagsItemPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewEditAssignedTagsItemPartial(infra *common.Infra, actions *Actions) *EditAssignedTagsItemPartial {
	return &EditAssignedTagsItemPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("edit-assigned-tags-item-partial"),
			true,
		),
	}
}

func (qq *EditAssignedTagsItemPartial) Data(fileID string, tagID int64) *EditAssignedTagsItemPartialData {
	return &EditAssignedTagsItemPartialData{
		FileID: fileID,
		TagID:  tagID,
	}
}

func (qq *EditAssignedTagsItemPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditAssignedTagsItemPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	tagx := ctx.TenantCtx().TTx.
		Tag.Query().
		WithSubTags(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		}).
		Where(tag.ID(data.TagID)).
		OnlyX(ctx)

	qq.infra.Renderer().RenderX(
		rw,
		ctx,
		qq.ListItem(ctx, data.FileID, tagx),
	)
	return nil
}

func (qq *EditAssignedTagsItemPartial) listItemID(fileID string, tagID int64) string {
	return fmt.Sprintf("listItemTag-%s-%d", fileID, tagID)
}

// error prone implementation, may lead to multiple calls to isCheckedFn if not
// used with care; // TODO maybe use a View instead?
func (qq *EditAssignedTagsItemPartial) IsCheckedFn(ctx ctxx.Context, fileID string) func(tagID int64) bool {
	// always loads all assigned tags, not optimal, but should be acceptable,
	// because there is just a small number...
	assignedTags := ctx.TenantCtx().TTx.
		File.Query().
		Where(file.PublicID(entx.NewCIText(fileID))).
		QueryTags().
		// Where(tag.HasParentsWith(tag.ID(data.GroupTagID))).
		// Where(tag.Not(tag.HasParents())).
		// WithChildren().
		AllX(ctx)
	return func(tagID int64) bool {
		return slices.ContainsFunc(assignedTags, func(assignedTag *enttenant.Tag) bool {
			return assignedTag.ID == tagID
		})
	}
}

// TODO ListItem or Widget
func (qq *EditAssignedTagsItemPartial) ListItem(
	ctx ctxx.Context,
	fileID string,
	tagx *enttenant.Tag,
) *widget.ListItem {
	listItem := qq.listItem(ctx, fileID, tagx, qq.IsCheckedFn(ctx, fileID))
	if tagx.Type == tagtype.Group {
		// if used as response/target
		// TODO is this the correct place?
		listItem.IsOpen = true
	}
	return listItem
}

func (qq *EditAssignedTagsItemPartial) listItem(
	ctx ctxx.Context,
	fileID string,
	tagx *enttenant.Tag,
	isCheckedFn func(tagID int64) bool,
) *widget.ListItem {
	var hxPost string
	var hxVals template.JS
	if isCheckedFn(tagx.ID) {
		hxPost = qq.actions.AssignedTags.UnassignTagCmd.Endpoint()
		hxVals = util.JSON(qq.actions.AssignedTags.UnassignTagCmd.Data(fileID, tagx.ID))
	} else {
		hxPost = qq.actions.AssignedTags.AssignTagCmd.Endpoint()
		hxVals = util.JSON(qq.actions.AssignedTags.AssignTagCmd.Data(fileID, tagx.ID))
	}

	id := qq.listItemID(fileID, tagx.ID)

	var icon *widget.Icon
	var supportingText string
	var trailing widget.IWidget
	var isCollapsible bool
	var htmxAttrs widget.HTMXAttrs

	var childItems []widget.IWidget

	if tagx.Type == tagtype.Group {
		// TODO find something betteer
		// folder_special
		// note_stack
		icon = widget.NewIcon("folder_special")

		if tagx.Edges.Children == nil {
			// TODO is there a better way to do this?
			tagx.Edges.Children = tagx.QueryChildren().AllX(ctx)
		}

		childTagsStr := fmt.Sprintf("%d child tag", len(tagx.Edges.Children))
		if len(tagx.Edges.Children) > 1 || len(tagx.Edges.Children) == 0 {
			childTagsStr += "s"
		}

		selectedCount := 0
		for _, childTag := range tagx.Edges.Children {
			if isCheckedFn(childTag.ID) {
				selectedCount++
			}
		}
		// TODO doesn't get updated on change; impl a web component?
		// TODO selected can be better indicated by color of icon or a badge?
		selectedStr := fmt.Sprintf("%d selected", selectedCount)

		// TODO indicate if children are checked with checkbox? via bg color?
		supportingText = fmt.Sprintf(
			"%s, %s",
			childTagsStr,
			selectedStr,
		)
		trailing = widget.NewIcon("keyboard_arrow_down")

		childItems = append(childItems, &widget.ListItem{
			Type:     widget.ListItemTypeHelper,
			Leading:  widget.NewIcon("new_label"),
			Headline: widget.T("Create new tag"), // group not possible
			HTMXAttrs: qq.actions.AssignedTags.CreateAndAssignTagCmd.ModalLinkAttrs(
				qq.actions.AssignedTags.CreateAndAssignTagCmd.Data(fileID, tagx.ID),
				"#"+qq.listItemID(fileID, tagx.ID),
				// "#"+qq.actions.AssignedTags.Edit.hxTargetID(),
			),
		})

		// children are eagerly loaded
		for _, childTag := range tagx.Edges.Children {
			// TODO fix is checked
			childItems = append(
				childItems,
				qq.listItem(ctx, fileID, childTag, isCheckedFn),
			)
		}

		isCollapsible = true
	} else if tagx.Type == tagtype.Super {
		icon = widget.NewIcon("label_important")

		supportingText = "Super tag"

		subTags, err := tagx.Edges.SubTagsOrErr()
		if err != nil {
			subTags = tagx.QuerySubTags().Order(tag.ByName()).AllX(ctx)
		}

		if len(subTags) > 0 {
			var tagNames []string
			for _, subTag := range subTags {
				tagNames = append(tagNames, subTag.Name)
			}
			// TODO add group to tags if it makes sense
			supportingText = fmt.Sprintf("Composed of %s", strings.Join(tagNames, ", "))
		}

		htmxAttrs = widget.HTMXAttrs{
			HxTrigger: event.SuperTagUpdated.Handler(tagx.ID),
			HxPost:    qq.actions.AssignedTags.EditListItem.Endpoint(),
			HxVals:    util.JSON(qq.actions.AssignedTags.EditListItem.Data(fileID, tagx.ID)),
			HxTarget:  "#" + id,
			HxSwap:    "outerHTML",
		}

		trailing = &widget.Checkbox{
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    hxPost,
				HxTrigger: "change",
				HxVals:    hxVals,
				HxTarget:  "#" + id,
				HxSwap:    "outerHTML",
			},
			IsChecked: isCheckedFn(tagx.ID),
		}
	} else {
		icon = widget.NewIcon("label")
		trailing = &widget.Checkbox{
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    hxPost,
				HxTrigger: "change",
				HxVals:    hxVals,
				HxTarget:  "#" + id,
				HxSwap:    "outerHTML",
			},
			IsChecked: isCheckedFn(tagx.ID),
		}
	}

	if tagx.Type == tagtype.Simple || tagx.Type == tagtype.Super {
		// TODO should link complete listItem, not content and trailing separatly,
		//		results in a small gap because if margin
		//
		// impl on refactoring on 27.10.24, nur sure if correct, would solve comment above
		htmxAttrs = widget.HTMXAttrs{
			HxPost:   hxPost,
			HxVals:   hxVals,
			HxTarget: "#" + id,
			HxSwap:   "outerHTML",
		}
	}

	return &widget.ListItem{
		Widget: widget.Widget[widget.ListItem]{
			ID: id,
		},
		HTMXAttrs:      htmxAttrs,
		Leading:        icon,
		Headline:       widget.T(tagx.Name),
		SupportingText: widget.Tu(supportingText),
		Trailing:       trailing,
		IsCollapsible:  isCollapsible,
		ContextMenu:    NewTagContextMenuWidget(qq.actions).Widget(ctx, fileID, tagx),
		Child:          childItems,
	}
}
