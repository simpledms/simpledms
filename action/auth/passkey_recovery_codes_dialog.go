package auth

import (
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/account"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

const passkeyRecoveryCodesDialogID = "passkeyRecoveryCodesDialog"

type PasskeyRecoveryCodesDialogData struct {
	Token string `validate:"required"`
}

type PasskeyRecoveryCodesDialog struct {
	infra   *common.Infra
	actions *Actions
	store   *account2.PasskeyRecoveryCodesStore
	*actionx.Config
}

func NewPasskeyRecoveryCodesDialog(
	infra *common.Infra,
	actions *Actions,
	store *account2.PasskeyRecoveryCodesStore,
) *PasskeyRecoveryCodesDialog {
	config := actionx.NewConfig(actions.Route("passkey-recovery-codes-dialog"), true).EnableSetupSessionAccess()
	return &PasskeyRecoveryCodesDialog{
		infra:   infra,
		actions: actions,
		store:   store,
		Config:  config,
	}
}

func (qq *PasskeyRecoveryCodesDialog) Data(token string) *PasskeyRecoveryCodesDialogData {
	return &PasskeyRecoveryCodesDialogData{
		Token: token,
	}
}

func (qq *PasskeyRecoveryCodesDialog) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to view backup codes.")
	if err != nil {
		return err
	}

	data, err := autil.FormData[PasskeyRecoveryCodesDialogData](rw, req, ctx)
	if err != nil {
		return err
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Pragma", "no-cache")

	recoveryCodesText, ok := qq.store.Consume(data.Token)
	if !ok {
		return e.NewHTTPErrorf(http.StatusBadRequest, "The backup codes are no longer available. Please generate a new set.")
	}

	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(recoveryCodesText, qq.accountDisplayName(mainCtx)))
}

func (qq *PasskeyRecoveryCodesDialog) Widget(recoveryCodesText string, accountName string) *wx.Dialog {
	content := &wx.RecoveryCodesDialogContent{
		CodesText:   strings.TrimSpace(recoveryCodesText),
		AccountName: strings.TrimSpace(accountName),
	}

	return &wx.Dialog{
		Widget: wx.Widget[wx.Dialog]{
			ID: passkeyRecoveryCodesDialogID,
		},
		Headline:     wx.T("Passkey backup codes"),
		IsOpenOnLoad: true,
		FooterActions: []wx.IWidget{
			content.PrintButton(),
			content.CopyButton(),
			content.DownloadButton(),
		},
		Child: content,
	}
}

func (qq *PasskeyRecoveryCodesDialog) accountDisplayName(ctx *ctxx.MainContext) string {
	name := strings.TrimSpace(
		strings.TrimSpace(ctx.Account.FirstName) + " " + strings.TrimSpace(ctx.Account.LastName),
	)
	if name != "" {
		return name
	}

	return ctx.Account.Email.String()
}
