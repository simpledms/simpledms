package property

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
		wx.T("Add field"),
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

func (qq *CreatePropertyCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreatePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	_ = ctx.SpaceCtx().TTx.Property.Create().
		SetName(data.Name).
		SetType(data.Type).
		SetUnit(data.Unit).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SaveX(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.PropertyCreated.String())

	return qq.infra.Renderer().Render(
		rw, ctx,
		wx.NewSnackbarf("Field «%s» created.", data.Name),
	)
}
