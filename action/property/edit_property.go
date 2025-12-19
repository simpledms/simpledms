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

type EditPropertyData struct {
	PropertyID int64  `validate:"required" form_attr_type:"hidden"`
	Name       string `validate:"required"`
	// don't allow to change type because that would mess with (corrupt)
	// values if already assigned to files
	Unit string // TODO show only for Types where it makes sense
}

type EditProperty struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditPropertyData]
}

func NewEditProperty(infra *common.Infra, actions *Actions) *EditProperty {
	config := actionx.NewConfig(actions.Route("edit-property"), false)
	return &EditProperty{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditPropertyData](infra, config, wx.T("Edit field")),
	}
}

func (qq *EditProperty) Data(propertyID int64, name string, unit string) *EditPropertyData {
	return &EditPropertyData{
		PropertyID: propertyID,
		Name:       name,
		Unit:       unit,
	}
}

func (qq *EditProperty) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditPropertyData](rw, req, ctx)
	if err != nil {
		return err
	}

	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)

	propertyx = propertyx.Update().
		SetName(data.Name).
		SetUnit(data.Unit).
		SaveX(ctx)

	rw.Header().Set("HX-Trigger", event.PropertyUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Field updated."))

	return nil
}
