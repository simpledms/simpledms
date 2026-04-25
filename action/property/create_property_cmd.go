package property

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/model/common/fieldtype"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type CreatePropertyCmdData struct {
	Name string              `validate:"required"`
	Type fieldtype.FieldType `validate:"required"`
	Unit string              // TODO show only for Types where it makes sense
}

type CreatePropertyCmdState struct {
}

// TODO AddProperty or CreatePropertyCmd?
type CreatePropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreatePropertyCmdData]
}

func NewCreatePropertyCmd(infra *common.Infra, actions *Actions) *CreatePropertyCmd {
	config := actionx.NewConfig(
		actions.Route("create-property-cmd"),
		false,
	)
	formHelper := autil.NewFormHelper[CreatePropertyCmdData](
		infra,
		config,
		widget.T("Add field"),
	)
	return &CreatePropertyCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *CreatePropertyCmd) Data(name string) *CreatePropertyCmdData {
	return &CreatePropertyCmdData{
		Name: name,
	}
}

func (qq *CreatePropertyCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreatePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	_, err = propertymodel.NewPropertyService().Create(
		ctx,
		ctx.SpaceCtx().Space.ID,
		data.Name,
		data.Type,
		data.Unit,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.PropertyCreated.String())

	return qq.infra.Renderer().Render(
		rw, ctx,
		widget.NewSnackbarf("Field «%s» created.", data.Name),
	)
}
