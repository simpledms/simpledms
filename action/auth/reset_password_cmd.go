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
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/mailer"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ResetPasswordCmdData struct {
	Email string `validate:"required,email"`
}

type ResetPasswordCmd struct {
	infra              *common.Infra
	actions            *Actions
	requestRateLimiter *account2.RequestRateLimiter
	*actionx.Config
	*autil.FormHelper[ResetPasswordCmdData]
}

func NewResetPasswordCmd(
	infra *common.Infra,
	actions *Actions,
	requestRateLimiter *account2.RequestRateLimiter,
) *ResetPasswordCmd {
	config := actionx.NewConfig(actions.Route("reset-password-cmd"), false)
	return &ResetPasswordCmd{
		infra:              infra,
		actions:            actions,
		requestRateLimiter: requestRateLimiter,
		Config:             config,
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
	if !qq.requestRateLimiter.Allow(
		rateLimitKey("reset-password-ip", clientIPFromRequest(req)),
		resetRateLimitWindow,
		resetRateLimitPerIP,
	) {
		return e.NewHTTPErrorf(
			http.StatusTooManyRequests,
			"Too many password reset requests. Please try again shortly.",
		)
	}
	if !qq.requestRateLimiter.Allow(
		rateLimitKey("reset-password-email", normalizeRateLimitedEmail(data.Email)),
		resetRateLimitWindow,
		resetRateLimitPerEmail,
	) {
		return e.NewHTTPErrorf(
			http.StatusTooManyRequests,
			"Too many password reset requests. Please try again shortly.",
		)
	}
	confirmationSnackbar := wx.NewSnackbarf(
		"If an account with this email exists, a new temporary password was sent.",
	)

	accountx, err := ctx.VisitorCtx().MainTx.Account.Query().Where(account.Email(entx.NewCIText(data.Email))).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			rw.AddRenderables(confirmationSnackbar)
			return nil
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

	mailer.NewMailer().ResetPassword(
		ctx,
		accountx,
		newPassword,
		expiresAt,
		qq.infra.SystemConfig().AbsoluteURL("/"),
	)
	rw.AddRenderables(confirmationSnackbar)

	return nil
}
