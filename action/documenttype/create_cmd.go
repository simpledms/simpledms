package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
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
		actions.Route("add-document-type"),
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

	// ctx.SpaceCtx().Space.QueryDocumentTypes().CreateCmd().SetName(data.Name).SaveX(ctx)
	ctx.SpaceCtx().TTx.DocumentType.
		Create().
		SetName(data.Name).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SaveX(ctx)

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
