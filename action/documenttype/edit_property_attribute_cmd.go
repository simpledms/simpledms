package documenttype

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditPropertyAttributeCmdData struct {
	ID           int64 `form_attr_type:"hidden"`
	IsNameGiving bool
}

type EditPropertyAttributeCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditPropertyAttributeCmdData]
}

func NewEditPropertyAttributeCmd(infra *common.Infra, actions *Actions) *EditPropertyAttributeCmd {
	config := actionx.NewConfig(
		actions.Route("edit-property-attribute"),
		false,
	)
	return &EditPropertyAttributeCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditPropertyAttributeCmdData](infra, config, wx.T("Edit field attribute")),
	}
}

func (qq *EditPropertyAttributeCmd) Data(id int64, isNameGiving bool) *EditPropertyAttributeCmdData {
	return &EditPropertyAttributeCmdData{
		ID:           id,
		IsNameGiving: isNameGiving,
	}
}

func (qq *EditPropertyAttributeCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditPropertyAttributeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = ctx.TenantCtx().TTx.Attribute.UpdateOneID(data.ID).
		SetIsNameGiving(data.IsNameGiving).
		Exec(ctx)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Attribute updated."))

	return nil
}
