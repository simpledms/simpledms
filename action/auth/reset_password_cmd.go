package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	account2 "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/mailer"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ResetPasswordCmdData struct {
	Email string `validate:"required,email"`
}

type ResetPasswordCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ResetPasswordCmdData]
}

func NewResetPasswordCmd(infra *common.Infra, actions *Actions) *ResetPasswordCmd {
	config := actionx.NewConfig(actions.Route("reset-password"), false)
	return &ResetPasswordCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[ResetPasswordCmdData](
			infra,
			config,
			wx.T("Reset password"),
			wx.T("Reset"),
		),
	}
}

func (qq *ResetPasswordCmd) Data(email string) *ResetPasswordCmdData {
	return &ResetPasswordCmdData{
		Email: email,
	}
}

func (qq *ResetPasswordCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[ResetPasswordCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	accountx, err := ctx.VisitorCtx().MainTx.Account.Query().Where(account.Email(entx.NewCIText(data.Email))).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Account not found.")
		}

		log.Println(err)
		return err
	}

	accountm := account2.NewAccount(accountx)

	newPassword, expiresAt, err := accountm.GenerateTemporaryPassword(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	mailer.NewMailer().ResetPassword(ctx, accountx, newPassword, expiresAt)
	rw.AddRenderables(wx.NewSnackbarf("A new temporary password was sent to your email address."))

	return nil
}
