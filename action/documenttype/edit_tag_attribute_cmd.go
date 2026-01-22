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

type EditTagAttributeCmdData struct {
	ID           int64  `form_attr_type:"hidden"`
	NewName      string `validate:"required"`
	IsNameGiving bool
}

type EditTagAttributeCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditTagAttributeCmdData]
}

func NewEditTagAttributeCmd(infra *common.Infra, actions *Actions) *EditTagAttributeCmd {
	config := actionx.NewConfig(
		actions.Route("edit-tag-attribute"),
		false,
	)
	return &EditTagAttributeCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditTagAttributeCmdData](infra, config, wx.T("Edit tag attribute")),
	}
}

func (qq *EditTagAttributeCmd) Data(id int64, newName string, isNameGiving bool) *EditTagAttributeCmdData {
	return &EditTagAttributeCmdData{
		ID:           id,
		NewName:      newName,
		IsNameGiving: isNameGiving,
	}
}

func (qq *EditTagAttributeCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditTagAttributeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = ctx.TenantCtx().TTx.Attribute.UpdateOneID(data.ID).
		SetName(data.NewName).
		SetIsNameGiving(data.IsNameGiving).
		Exec(ctx)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Attribute «%s» updated.", data.NewName))

	return nil
}
