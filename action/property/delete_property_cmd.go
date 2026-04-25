package property

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type DeletePropertyCmdData struct {
	PropertyID int64 `validate:"required"`
}

type DeletePropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeletePropertyCmd(infra *common.Infra, actions *Actions) *DeletePropertyCmd {
	config := actionx.NewConfig(actions.Route("delete-property-cmd"), false)
	return &DeletePropertyCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeletePropertyCmd) Data(propertyID int64) *DeletePropertyCmdData {
	return &DeletePropertyCmdData{
		PropertyID: propertyID,
	}
}

func (qq *DeletePropertyCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeletePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = propertymodel.NewPropertyService().Delete(ctx, ctx.SpaceCtx().Space, data.PropertyID)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.PropertyDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Field deleted."))

	return nil
}
