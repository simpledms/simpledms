package documenttype

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditTagAttributeData struct {
	ID           int64  `form_attr_type:"hidden"`
	NewName      string `validate:"required"`
	IsNameGiving bool
}

type EditTagAttribute struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditTagAttributeData]
}

func NewEditTagAttribute(infra *common.Infra, actions *Actions) *EditTagAttribute {
	config := actionx.NewConfig(
		actions.Route("edit-tag-attribute"),
		false,
	)
	return &EditTagAttribute{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditTagAttributeData](infra, config, wx.T("Edit tag attribute")),
	}
}

func (qq *EditTagAttribute) Data(id int64, newName string, isNameGiving bool) *EditTagAttributeData {
	return &EditTagAttributeData{
		ID:           id,
		NewName:      newName,
		IsNameGiving: isNameGiving,
	}
}

func (qq *EditTagAttribute) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditTagAttributeData](rw, req, ctx)
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
