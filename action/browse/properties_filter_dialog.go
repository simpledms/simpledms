package browse

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type PropertiesFilterDialogData struct {
	CurrentDirID string
}

type PropertiesFilterDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewPropertiesFilterDialog(infra *common.Infra, actions *Actions) *PropertiesFilterDialog {
	config := actionx.NewConfig(
		actions.Route("properties-filter-dialog"),
		true,
	)
	return &PropertiesFilterDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *PropertiesFilterDialog) Data(currentDirID string) *PropertiesFilterDialogData {
	return &PropertiesFilterDialogData{
		CurrentDirID: currentDirID,
	}
}

func (qq *PropertiesFilterDialog) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[PropertiesFilterDialogData](rw, req, ctx)
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

func (qq *PropertiesFilterDialog) Widget(
	ctx ctxx.Context,
	data *PropertiesFilterDialogData,
	listDirState *ListDirPartialState,
) *widget.Dialog {
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:     widget.T("Fields | Filter"),
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutSideSheet,
		Child: qq.actions.ListFilterPropertiesPartial.Widget(
			ctx,
			qq.actions.ListFilterPropertiesPartial.Data(data.CurrentDirID, 0),
			listDirState,
		),
	}
}

func (qq *PropertiesFilterDialog) ID() string {
	return "filterPropertiesDialog"
}
