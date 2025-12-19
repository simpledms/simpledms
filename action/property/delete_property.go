package property

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant/property"
	"github.com/simpledms/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeletePropertyData struct {
	PropertyID int64 `validate:"required"`
}

type DeleteProperty struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteProperty(infra *common.Infra, actions *Actions) *DeleteProperty {
	config := actionx.NewConfig(actions.Route("delete-property"), false)
	return &DeleteProperty{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteProperty) Data(propertyID int64) *DeletePropertyData {
	return &DeletePropertyData{
		PropertyID: propertyID,
	}
}

func (qq *DeleteProperty) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeletePropertyData](rw, req, ctx)
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
