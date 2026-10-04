package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FiltersDialogData struct {
	CurrentDirID string
}

type FiltersDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFiltersDialog(infra *common.Infra, actions *Actions) *FiltersDialog {
	config := actionx.NewConfig(
		actions.Route("filters-dialog"),
		true,
	)
	return &FiltersDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FiltersDialog) Data(currentDirID string) *FiltersDialogData {
	return &FiltersDialogData{
		CurrentDirID: currentDirID,
	}
}

func (qq *FiltersDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FiltersDialogData](rw, req, ctx)
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

func (qq *FiltersDialog) Widget(
	ctx ctxx.Context,
	data *FiltersDialogData,
	listDirState *ListDirPartialState,
) *widget.Dialog {
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:     widget.T("Filters"),
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutSideSheet,
		HeaderActions: []widget.IWidget{
			&widget.Button{
				Icon:      widget.NewIcon("restart_alt"),
				Label:     widget.T("Reset"),
				StyleType: widget.ButtonStyleTypeText,
				HTMXAttrs: widget.HTMXAttrs{
					Role:   "button",
					HxPost: qq.actions.ResetFiltersCmd.Endpoint(),
					HxVals: util.JSON(qq.actions.ResetFiltersCmd.Data(data.CurrentDirID)),
					HxSwap: "none",
				},
			},
		},
		Child: qq.actions.FilterTabsPartial.Widget(
			ctx,
			listDirState,
			data.CurrentDirID,
		),
	}
}

func (qq *FiltersDialog) ID() string {
	return "filtersDialog"
}
