package documenttype

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
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
