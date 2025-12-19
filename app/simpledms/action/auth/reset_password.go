package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/entmain"
	"github.com/simpledms/simpledms/app/simpledms/entmain/account"
	"github.com/simpledms/simpledms/app/simpledms/entx"
	"github.com/simpledms/simpledms/app/simpledms/model/mailer"
	"github.com/simpledms/simpledms/app/simpledms/model/modelmain"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ResetPasswordData struct {
	Email string `validate:"required,email"`
}

type ResetPassword struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ResetPasswordData]
}

func NewResetPassword(infra *common.Infra, actions *Actions) *ResetPassword {
	config := actionx.NewConfig(actions.Route("reset-password"), false)
	return &ResetPassword{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[ResetPasswordData](
			infra,
			config,
			wx.T("Reset password"),
			wx.T("Reset"),
		),
	}
}

func (qq *ResetPassword) Data(email string) *ResetPasswordData {
	return &ResetPasswordData{
		Email: email,
	}
}

func (qq *ResetPassword) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[ResetPasswordData](rw, req, ctx)
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

	accountm := modelmain.NewAccount(accountx)

	newPassword, expiresAt, err := accountm.GenerateTemporaryPassword(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	mailer.NewMailer().ResetPassword(ctx, accountx, newPassword, expiresAt)
	rw.AddRenderables(wx.NewSnackbarf("A new temporary password was sent to your email address."))

	return nil
}
