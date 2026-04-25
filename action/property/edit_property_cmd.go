package property

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type EditPropertyCmdData struct {
	PropertyID int64  `validate:"required" form_attr_type:"hidden"`
	Name       string `validate:"required"`
	// don't allow to change type because that would mess with (corrupt)
	// values if already assigned to files
	Unit string // TODO show only for Types where it makes sense
}

type EditPropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditPropertyCmdData]
}

func NewEditPropertyCmd(infra *common.Infra, actions *Actions) *EditPropertyCmd {
	config := actionx.NewConfig(actions.Route("edit-property-cmd"), false)
	return &EditPropertyCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditPropertyCmdData](infra, config, widget.T("Edit field")),
	}
}

func (qq *EditPropertyCmd) Data(propertyID int64, name string, unit string) *EditPropertyCmdData {
	return &EditPropertyCmdData{
		PropertyID: propertyID,
		Name:       name,
		Unit:       unit,
	}
}

func (qq *EditPropertyCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditPropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	_, err = propertymodel.NewPropertyService().Edit(
		ctx,
		ctx.SpaceCtx().Space,
		data.PropertyID,
		data.Name,
		data.Unit,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.PropertyUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Field updated."))

	return nil
}
