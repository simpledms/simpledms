package inbox

import (
	"html/template"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SourceFilterDialog struct {
	infra *common.Infra
	*actionx.Config
}

func NewSourceFilterDialog(infra *common.Infra, actions *Actions) *SourceFilterDialog {
	return &SourceFilterDialog{
		infra: infra,
		Config: actionx.NewConfig(
			actions.Route("source-filter-dialog"),
			true,
		),
	}
}

func (qq *SourceFilterDialog) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	state := autil.StateX[InboxPageState](rw, req)
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(&state.FilesListPartialState))
}

func (qq *SourceFilterDialog) Widget(state *FilesListPartialState) *widget.Dialog {
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:     widget.T("Source | Filter"),
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutSideSheet,
		Child: &widget.Container{
			Widget: widget.Widget[widget.Container]{
				ID: "inboxSourceFilter",
			},
			Child: qq.chips(state),
		},
	}
}

func (qq *SourceFilterDialog) chips(state *FilesListPartialState) []*widget.FilterChip {
	chips := make([]*widget.FilterChip, 0, len(filesource.Values()))
	for _, source := range filesource.Values() {
		sourceValue := source.String()
		checked := state.hasSource(source)
		chips = append(chips, &widget.FilterChip{
			Type:      widget.FilterChipTypeCheckbox,
			Label:     autil.FileSourceLabel(source),
			Name:      "SourceValues",
			Value:     sourceValue,
			IsChecked: checked,
			HTMXAttrs: widget.HTMXAttrs{
				HxOn: &widget.HxOn{
					Event: "change",
					Handler: template.JS(
						"if (event.target.checked) { " +
							"_appendQueryParamSliceValue('source', '" + sourceValue + "'); " +
							"} else { _deleteQueryParamSliceValue('source', '" + sourceValue + "'); } " +
							"this.dispatchEvent(new CustomEvent('" +
							event.SourceFilterChanged.String() + "', { bubbles: true }))",
					),
				},
			},
		})
	}
	return chips
}

func (qq *SourceFilterDialog) ID() string {
	return "filterSourceDialog"
}
