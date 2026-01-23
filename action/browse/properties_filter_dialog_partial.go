package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type PropertiesFilterDialogPartialData struct {
	CurrentDirID string
}

type PropertiesFilterDialogPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewPropertiesFilterDialogPartial(infra *common.Infra, actions *Actions) *PropertiesFilterDialogPartial {
	config := actionx.NewConfig(
		actions.Route("properties-filter-dialog-partial"),
		true,
	)
	return &PropertiesFilterDialogPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *PropertiesFilterDialogPartial) Data(currentDirID string) *PropertiesFilterDialogPartialData {
	return &PropertiesFilterDialogPartialData{
		CurrentDirID: currentDirID,
	}
}

func (qq *PropertiesFilterDialogPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[PropertiesFilterDialogPartialData](rw, req, ctx)
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

func (qq *PropertiesFilterDialogPartial) Widget(
	ctx ctxx.Context,
	data *PropertiesFilterDialogPartialData,
	listDirState *ListDirPartialState,
) *wx.Dialog {
	return &wx.Dialog{
		Widget: wx.Widget[wx.Dialog]{
			ID: qq.ID(),
		},
		Headline:     wx.T("Fields | Filter"),
		IsOpenOnLoad: true,
		Layout:       wx.DialogLayoutSideSheet,
		Child: qq.actions.ListFilterPropertiesPartial.Widget(
			ctx,
			qq.actions.ListFilterPropertiesPartial.Data(data.CurrentDirID, 0),
			listDirState,
		),
	}
}

func (qq *PropertiesFilterDialogPartial) ID() string {
	return "filterPropertiesDialog"
}
