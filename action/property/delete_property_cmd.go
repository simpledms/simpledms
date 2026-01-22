package property

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *DeletePropertyCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeletePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// first query ensures it belongs to current space
	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)
	ctx.SpaceCtx().TTx.Property.DeleteOne(propertyx).ExecX(ctx)

	rw.Header().Set("HX-Trigger", event.PropertyDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Field deleted."))

	return nil
}
