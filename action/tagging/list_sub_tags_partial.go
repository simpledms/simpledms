package tagging

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
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
			actions.Route("list-sub-tags-partial"),
			true,
		),
	}
}

func (qq *ListSubTagsPartial) Data(superTagID int64) *ListSubTagsPartialData {
	return &ListSubTagsPartialData{
		SuperTagID: superTagID,
	}
}

func (qq *ListSubTagsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListSubTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	subTags := ctx.AppCtx().TTx.Tag.
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

func (qq *ListSubTagsPartial) Widget(subTagsWithParent []*enttenant.Tag) *widget.List {
	var listItems []*widget.ListItem

	for _, subTag := range subTagsWithParent {
		headline := widget.T(subTag.Name)
		var supportingText *widget.Text
		if subTag.Edges.Group != nil {
			// headline = NewTextf("%s: %s", tagx.Edges.Parent.Name, headline.Data)
			supportingText = widget.Tf("Group «%s»", subTag.Edges.Group.Name)
		}
		listItems = append(
			listItems, &widget.ListItem{
				Leading:        widget.NewIcon("label"),
				Headline:       headline,
				SupportingText: supportingText,
			},
		)
	}

	return &widget.List{
		Children: listItems,
	}
}
