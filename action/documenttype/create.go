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

type CreateData struct {
	Name string `validate:"required"`
}

type Create struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateData]
}

func NewCreate(infra *common.Infra, actions *Actions) *Create {
	config := actionx.NewConfig(
		actions.Route("add-document-type"),
		false,
	)
	formHelper := autil.NewFormHelper[CreateData](
		infra,
		config,
		wx.T("Add document type"),
	)
	return &Create{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *Create) Data(name string) *CreateData {
	return &CreateData{
		Name: name,
	}
}

func (qq *Create) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateData](rw, req, ctx)
	if err != nil {
		return err
	}

	// ctx.SpaceCtx().Space.QueryDocumentTypes().Create().SetName(data.Name).SaveX(ctx)
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
