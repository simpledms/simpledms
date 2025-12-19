package property

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	"github.com/simpledms/simpledms/app/simpledms/model/common/fieldtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreatePropertyData struct {
	Name string              `validate:"required"`
	Type fieldtype.FieldType `validate:"required"`
	Unit string              // TODO show only for Types where it makes sense
}

type CreatePropertyState struct {
}

// TODO AddProperty or CreateProperty?
type CreateProperty struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreatePropertyData]
}

func NewCreateProperty(infra *common.Infra, actions *Actions) *CreateProperty {
	config := actionx.NewConfig(
		actions.Route("create-property"),
		false,
	)
	formHelper := autil.NewFormHelper[CreatePropertyData](
		infra,
		config,
		wx.T("Add field"),
	)
	return &CreateProperty{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *CreateProperty) Data(name string) *CreatePropertyData {
	return &CreatePropertyData{
		Name: name,
	}
}

func (qq *CreateProperty) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreatePropertyData](rw, req, ctx)
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
