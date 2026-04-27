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

type TagsFilterDialogData struct {
	CurrentDirID string
}

type TagsFilterDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewTagsFilterDialog(infra *common.Infra, actions *Actions) *TagsFilterDialog {
	config := actionx.NewConfig(
		actions.Route("tags-filter-dialog"),
		true,
	)
	return &TagsFilterDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *TagsFilterDialog) Data(currentDirID string) *TagsFilterDialogData {
	return &TagsFilterDialogData{
		CurrentDirID: currentDirID,
	}
}

func (qq *TagsFilterDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[TagsFilterDialogData](rw, req, ctx)
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

func (qq *TagsFilterDialog) Widget(
	ctx ctxx.Context,
	data *TagsFilterDialogData,
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

func (qq *TagsFilterDialog) ID() string {
	return "filterTagsDialog"
}
