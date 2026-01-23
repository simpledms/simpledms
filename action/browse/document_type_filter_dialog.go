package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *DocumentTypeFilterDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

func (qq *DocumentTypeFilterDialog) Widget(ctx ctxx.Context, data *DocumentTypeFilterDialogData, listDirState *ListDirPartialState) *wx.Dialog {
	return &wx.Dialog{
		Widget: wx.Widget[wx.Dialog]{
			ID: qq.ID(),
		},
		Headline:     wx.T("Document type | Filter"),
		IsOpenOnLoad: true,
		Layout:       wx.DialogLayoutSideSheet,
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
