package auth

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type PasskeyRecoverySignInCmdData struct {
	Email            string `validate:"required,email" form_attrs:"autofocus"`
	BackupCode       string `validate:"required" form_attr_type:"password"`
	TemporarySession bool
}

type PasskeyRecoverySignInCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
	*autil.FormHelper[PasskeyRecoverySignInCmdData]
}

func NewPasskeyRecoverySignInCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *PasskeyRecoverySignInCmd {
	config := actionx.NewConfig(actions.Route("passkey-recovery-sign-in-cmd"), false)
	return &PasskeyRecoverySignInCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
		FormHelper: autil.NewFormHelperX[PasskeyRecoverySignInCmdData](
			infra,
			config,
			wx.T("Sign in with backup code"),
			wx.T("Sign in"),
		),
	}
}

func (qq *PasskeyRecoverySignInCmd) Data(email, recoveryCode string) *PasskeyRecoverySignInCmdData {
	return &PasskeyRecoverySignInCmdData{
		Email:      email,
		BackupCode: recoveryCode,
	}
}

func (qq *PasskeyRecoverySignInCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[PasskeyRecoverySignInCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	accountx, err := qq.passkeyService.SignInWithRecoveryCode(
		ctx,
		data.Email,
		data.BackupCode,
	)
	if err != nil {
		return err
	}

	err = createAccountSession(
		rw,
		req,
		ctx,
		accountx,
		data.TemporarySession,
		qq.infra.SystemConfig().AllowInsecureCookies(),
	)
	if err != nil {
		return err
	}

	updatedAccountx, err := ctx.VisitorCtx().MainTx.Account.Get(ctx, accountx.ID)
	if err != nil {
		return err
	}

	recoveryCodesCount := len(updatedAccountx.PasskeyRecoveryCodeHashes)
	rw.AddRenderables(wx.NewSnackbarf("Logged in successfully. %d backup codes left.", recoveryCodesCount))
	rw.Header().Set("HX-Redirect", route.Dashboard())

	return nil
}
