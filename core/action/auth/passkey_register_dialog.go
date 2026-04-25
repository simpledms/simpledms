package auth

import (
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

const passkeyRegisterDialogID = "passkeyRegisterDialog"
const passkeyRegisterDialogContentID = "passkeyRegisterDialogContent"

type PasskeyRegisterDialogData struct{}

type PasskeyRegisterDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewPasskeyRegisterDialog(infra *common.Infra, actions *Actions) *PasskeyRegisterDialog {
	config := actionx.NewConfig(actions.Route("passkey-register-dialog"), true).EnableSetupSessionAccess()
	return &PasskeyRegisterDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *PasskeyRegisterDialog) Data() *PasskeyRegisterDialogData {
	return &PasskeyRegisterDialogData{}
}

func (qq *PasskeyRegisterDialog) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	_, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to register a passkey.")
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(rw, ctx, qq.Widget())
}

func (qq *PasskeyRegisterDialog) Widget() *widget.Dialog {
	content := &widget.PasskeyRegisterDialogContent{
		Widget: widget.Widget[widget.PasskeyRegisterDialogContent]{
			ID: passkeyRegisterDialogContentID,
		},
		DialogID: passkeyRegisterDialogID,
	}

	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: passkeyRegisterDialogID,
		},
		Headline:     widget.T("Register passkey"),
		SubmitLabel:  content.SubmitLabel(),
		FormID:       content.FormID(),
		IsOpenOnLoad: true,
		Child:        content,
	}
}
