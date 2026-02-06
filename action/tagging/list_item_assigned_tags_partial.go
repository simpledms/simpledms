package tagging

import (
	"fmt"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *ListItemAssignedTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

func (qq *ListItemAssignedTagsPartial) Widget(ctx ctxx.Context, tagx *enttenant.Tag) *wx.ListItem {
	headline := wx.Tu(tagx.Name)
	var supportingText *wx.Text
	if tagx.Edges.Group != nil {
		// headline = NewTextf("%s: %s", tagx.Edges.Parent.Name, headline.Data)
		supportingText = wx.Tf("Group «%s»", tagx.Edges.Group.Name)
	}

	icon := wx.NewIcon("label")
	var htmxAttrs wx.HTMXAttrs
	listItemID := autil.GenerateID(fmt.Sprintf("ListAssignedTagsPartial-%d-", tagx.ID))

	if tagx.Type == tagtype.Super {
		icon = wx.NewIcon("label_important")
		supportingText = wx.T("Super tag")

		if len(tagx.Edges.SubTags) > 0 {
			var tagNames []string
			for _, subTag := range tagx.Edges.SubTags {
				tagNames = append(tagNames, subTag.Name)
			}
			// TODO add group to tags if it makes sense
			supportingText = wx.Tf("Composed of %s", strings.Join(tagNames, ", "))
		}

		htmxAttrs = wx.HTMXAttrs{
			HxTrigger: event.SuperTagUpdated.Handler(tagx.ID),
			HxPost:    qq.actions.AssignedTags.ListItem.Endpoint(),
			HxVals:    util.JSON(qq.actions.AssignedTags.ListItem.Data(tagx.ID)),
			HxTarget:  "#" + listItemID,
			HxSwap:    "outerHTML",
		}
	}

	return &wx.ListItem{
		Widget: wx.Widget[wx.ListItem]{
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
