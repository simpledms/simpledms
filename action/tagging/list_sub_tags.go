package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ListSubTagsData struct {
	SuperTagID int64
}

type ListSubTags struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListSubTags(infra *common.Infra, actions *Actions) *ListSubTags {
	return &ListSubTags{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-sub-tags"),
			true,
		),
	}
}

func (qq *ListSubTags) Data(superTagID int64) *ListSubTagsData {
	return &ListSubTagsData{
		SuperTagID: superTagID,
	}
}

func (qq *ListSubTags) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListSubTagsData](rw, req, ctx)
	if err != nil {
		return err
	}

	subTags := ctx.TenantCtx().TTx.Tag.
		GetX(ctx, data.SuperTagID).
		QuerySubTags().
		WithGroup().
		Where(tag.TypeEQ(tagtype.Simple)).
		AllX(ctx)

	qq.infra.Renderer().RenderX(
		rw,
		ctx,
		qq.Widget(subTags),
	)
	return nil
}

func (qq *ListSubTags) Widget(subTagsWithParent []*enttenant.Tag) *wx.List {
	var listItems []*wx.ListItem

	for _, subTag := range subTagsWithParent {
		headline := wx.T(subTag.Name)
		var supportingText *wx.Text
		if subTag.Edges.Group != nil {
			// headline = NewTextf("%s: %s", tagx.Edges.Parent.Name, headline.Data)
			supportingText = wx.Tf("Group «%s»", subTag.Edges.Group.Name)
		}
		listItems = append(
			listItems, &wx.ListItem{
				Leading:        wx.NewIcon("label"),
				Headline:       headline,
				SupportingText: supportingText,
			},
		)
	}

	return &wx.List{
		Children: listItems,
	}
}
