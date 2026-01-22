package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ListSubTagsPartialData struct {
	SuperTagID int64
}

type ListSubTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListSubTagsPartial(infra *common.Infra, actions *Actions) *ListSubTagsPartial {
	return &ListSubTagsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-sub-tags"),
			true,
		),
	}
}

func (qq *ListSubTagsPartial) Data(superTagID int64) *ListSubTagsPartialData {
	return &ListSubTagsPartialData{
		SuperTagID: superTagID,
	}
}

func (qq *ListSubTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListSubTagsPartialData](rw, req, ctx)
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

func (qq *ListSubTagsPartial) Widget(subTagsWithParent []*enttenant.Tag) *wx.List {
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
