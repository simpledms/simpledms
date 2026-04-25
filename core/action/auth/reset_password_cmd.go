package auth

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	account3 "github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/model/mailer"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type ResetPasswordCmdData struct {
	Email string `validate:"required,email"`
}

type ResetPasswordCmd struct {
	infra              *common.Infra
	actions            *Actions
	requestRateLimiter *account3.RequestRateLimiter
	*actionx.Config
	*autil.FormHelper[ResetPasswordCmdData]
}

func NewResetPasswordCmd(
	infra *common.Infra,
	actions *Actions,
	requestRateLimiter *account3.RequestRateLimiter,
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
			widget.T("Reset password"),
			widget.T("Reset"),
		),
	}
}

func (qq *ResetPasswordCmd) Data(email string) *ResetPasswordCmdData {
	return &ResetPasswordCmdData{
		Email: email,
	}
}

func (qq *ResetPasswordCmd) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
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
	confirmationSnackbar := widget.NewSnackbarf(
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

	accountm := account3.NewAccount(accountx)

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
