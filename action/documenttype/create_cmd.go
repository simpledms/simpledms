package documenttype

// package action

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

type CreateCmdData struct {
	Name string `validate:"required"`
}

type CreateCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateCmdData]
}

func NewCreateCmd(infra *common.Infra, actions *Actions) *CreateCmd {
	config := actionx.NewConfig(
		actions.Route("add-document-type-cmd"),
		false,
	)
	formHelper := autil.NewFormHelper[CreateCmdData](
		infra,
		config,
		widget.T("Add document type"),
	)
	return &CreateCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *CreateCmd) Data(name string) *CreateCmdData {
	return &CreateCmdData{
		Name: name,
	}
}

func (qq *CreateCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	_, err = documenttypemodel.Create(
		ctx,
		ctx.SpaceCtx().Space.ID,
		data.Name,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeCreated.String())
	rw.Header().Set("HX-Reswap", "none")

	// prevents snackbar and closing modal
	// rw.Header().Set("HX-Location", "/")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		widget.NewSnackbarf("Document type created."),
	)
}
