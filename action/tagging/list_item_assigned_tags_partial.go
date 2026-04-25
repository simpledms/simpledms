package tagging

import (
	"fmt"
	"strings"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
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

type ListItemAssignedTagsPartialData struct {
	TagID int64
}

type ListItemAssignedTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListItemAssignedTagsPartial(infra *common.Infra, actions *Actions) *ListItemAssignedTagsPartial {
	return &ListItemAssignedTagsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-item-assigned-tags-partial"),
			true,
		),
	}
}

func (qq *ListItemAssignedTagsPartial) Data(tagID int64) *ListItemAssignedTagsPartialData {
	return &ListItemAssignedTagsPartialData{
		TagID: tagID,
	}
}

func (qq *ListItemAssignedTagsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListItemAssignedTagsPartialData](rw, req, ctx)
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
		qq.Widget(ctx, tagx),
	)
	return nil
}

func (qq *ListItemAssignedTagsPartial) Widget(ctx ctxx.Context, tagx *enttenant.Tag) *widget.ListItem {
	headline := widget.Tu(tagx.Name)
	var supportingText *widget.Text
	if tagx.Edges.Group != nil {
		// headline = NewTextf("%s: %s", tagx.Edges.Parent.Name, headline.Data)
		supportingText = widget.Tf("Group «%s»", tagx.Edges.Group.Name)
	}

	icon := widget.NewIcon("label")
	var htmxAttrs widget.HTMXAttrs
	listItemID := autil.GenerateID(fmt.Sprintf("ListAssignedTagsPartial-%d-", tagx.ID))

	if tagx.Type == tagtype.Super {
		icon = widget.NewIcon("label_important")
		supportingText = widget.T("Super tag")

		if len(tagx.Edges.SubTags) > 0 {
			var tagNames []string
			for _, subTag := range tagx.Edges.SubTags {
				tagNames = append(tagNames, subTag.Name)
			}
			// TODO add group to tags if it makes sense
			supportingText = widget.Tf("Composed of %s", strings.Join(tagNames, ", "))
		}

		htmxAttrs = widget.HTMXAttrs{
			HxTrigger: event.SuperTagUpdated.Handler(tagx.ID),
			HxPost:    qq.actions.AssignedTags.ListItem.Endpoint(),
			HxVals:    util.JSON(qq.actions.AssignedTags.ListItem.Data(tagx.ID)),
			HxTarget:  "#" + listItemID,
			HxSwap:    "outerHTML",
		}
	}

	return &widget.ListItem{
		Widget: widget.Widget[widget.ListItem]{
			ID: listItemID,
		},
		HTMXAttrs:      htmxAttrs,
		Leading:        icon,
		Headline:       headline,
		SupportingText: supportingText,
		// Trailing:      nil,
		ContextMenu: NewTagContextMenuWidget(qq.actions).Widget(ctx, "", tagx),
	}
}
