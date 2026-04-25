package documenttype

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
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
		actions.Route("edit-tag-attribute-cmd"),
		false,
	)
	return &EditTagAttributeCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditTagAttributeCmdData](infra, config, widget.T("Edit tag attribute")),
	}
}

func (qq *EditTagAttributeCmd) Data(id int64, newName string, isNameGiving bool) *EditTagAttributeCmdData {
	return &EditTagAttributeCmdData{
		ID:           id,
		NewName:      newName,
		IsNameGiving: isNameGiving,
	}
}

func (qq *EditTagAttributeCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditTagAttributeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	attributex, err := documenttypemodel.QueryAttributeByID(ctx, data.ID)
	if err != nil {
		return err
	}

	err = attributex.RenameAndSetIsNameGiving(ctx, data.NewName, data.IsNameGiving)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Attribute «%s» updated.", data.NewName))

	return nil
}
