package documenttype

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
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
		actions.Route("edit-property-attribute-cmd"),
		false,
	)
	return &EditPropertyAttributeCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditPropertyAttributeCmdData](infra, config, widget.T("Edit field attribute")),
	}
}

func (qq *EditPropertyAttributeCmd) Data(id int64, isNameGiving bool) *EditPropertyAttributeCmdData {
	return &EditPropertyAttributeCmdData{
		ID:           id,
		IsNameGiving: isNameGiving,
	}
}

func (qq *EditPropertyAttributeCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditPropertyAttributeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	attributex, err := documenttypemodel.QueryAttributeByID(ctx, data.ID)
	if err != nil {
		return err
	}

	err = attributex.SetIsNameGiving(ctx, data.IsNameGiving)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Attribute updated."))

	return nil
}
