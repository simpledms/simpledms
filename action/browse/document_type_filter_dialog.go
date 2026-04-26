package browse

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
)

type DocumentTypeFilterDialogData struct {
	CurrentDirID string
}

type DocumentTypeFilterDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDocumentTypeFilterDialog(infra *common.Infra, actions *Actions) *DocumentTypeFilterDialog {
	config := actionx.NewConfig(
		actions.Route("document-type-filter-dialog"),
		true,
	)
	return &DocumentTypeFilterDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DocumentTypeFilterDialog) Data(currentDirID string) *DocumentTypeFilterDialogData {
	return &DocumentTypeFilterDialogData{
		CurrentDirID: currentDirID,
	}
}

func (qq *DocumentTypeFilterDialog) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DocumentTypeFilterDialogData](rw, req, ctx)
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

func (qq *DocumentTypeFilterDialog) Widget(ctx ctxx.Context, data *DocumentTypeFilterDialogData, listDirState *ListDirPartialState) *widget.Dialog {
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:     widget.T("Document type | Filter"),
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutSideSheet,
		Child: qq.actions.DocumentTypeFilterPartial.Widget(
			ctx,
			qq.actions.DocumentTypeFilterPartial.Data(data.CurrentDirID),
			listDirState,
		),
	}
}

func (qq *DocumentTypeFilterDialog) ID() string {
	return "filterDocumentTypeDialog"
}
