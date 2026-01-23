package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type TagsFilterDialogPartialData struct {
	CurrentDirID string
}

type TagsFilterDialogPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewTagsFilterDialogPartial(infra *common.Infra, actions *Actions) *TagsFilterDialogPartial {
	config := actionx.NewConfig(
		actions.Route("tags-filter-dialog-partial"),
		true,
	)
	return &TagsFilterDialogPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *TagsFilterDialogPartial) Data(currentDirID string) *TagsFilterDialogPartialData {
	return &TagsFilterDialogPartialData{
		CurrentDirID: currentDirID,
	}
}

func (qq *TagsFilterDialogPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[TagsFilterDialogPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirPartialState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data, state),
	)
}

func (qq *TagsFilterDialogPartial) Widget(
	ctx ctxx.Context,
	data *TagsFilterDialogPartialData,
	listDirState *ListDirPartialState,
) renderable.Renderable {
	// if listDirState.OpenDialog == qq.ID() {
	// TODO remove state from URL
	// return &wx.View{}
	// }

	return &wx.Dialog{
		Widget: wx.Widget[wx.Dialog]{
			ID: qq.ID(),
		},
		Headline:     wx.T("Tags | Filter"),
		IsOpenOnLoad: true,
		Layout:       wx.DialogLayoutSideSheet,
		Child: qq.actions.ListFilterTagsPartial.Widget(
			ctx,
			data.CurrentDirID,
			listDirState.ListFilterTagsPartialState.CheckedTagIDs,
		),
	}

}

func (qq *TagsFilterDialogPartial) ID() string {
	return "filterTagsDialog"
}
