package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
		wx.T("Add document type"),
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

func (qq *CreateCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	_, err = documenttypemodel.NewDocumentTypeService().Create(
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
		wx.NewSnackbarf("Document type created."),
	)
}
