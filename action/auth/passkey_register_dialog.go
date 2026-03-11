package auth

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	_, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to register a passkey.")
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(rw, ctx, qq.Widget())
}

func (qq *PasskeyRegisterDialog) Widget() *wx.Dialog {
	content := &wx.PasskeyRegisterDialogContent{
		Widget: wx.Widget[wx.PasskeyRegisterDialogContent]{
			ID: passkeyRegisterDialogContentID,
		},
		DialogID: passkeyRegisterDialogID,
	}

	return &wx.Dialog{
		Widget: wx.Widget[wx.Dialog]{
			ID: passkeyRegisterDialogID,
		},
		Headline:     wx.T("Register passkey"),
		SubmitLabel:  content.SubmitLabel(),
		FormID:       content.FormID(),
		IsOpenOnLoad: true,
		Child:        content,
	}
}
