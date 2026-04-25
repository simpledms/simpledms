package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type ChangePasswordCmdData struct {
	CurrentOrTemporaryPassword string `validate:"required" form_attr_type:"password"`
	NewPassword                string `validate:"required" form_attr_type:"password"`
	ConfirmPassword            string `validate:"required" form_attr_type:"password"`
}

type ChangePasswordCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ChangePasswordCmdData]
}

func NewChangePasswordCmd(infra *common.Infra, actions *Actions) *ChangePasswordCmd {
	config := actionx.NewConfig(actions.Route("change-password-cmd"), false)
	return &ChangePasswordCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[ChangePasswordCmdData](
			infra,
			config,
			widget.T("Change password"),
			widget.T("Change"),
		),
	}
}

func (qq *ChangePasswordCmd) Data(currentPassword, newPassword, confirmPassword string) *ChangePasswordCmdData {
	return &ChangePasswordCmdData{
		CurrentOrTemporaryPassword: currentPassword,
		NewPassword:                newPassword,
		ConfirmPassword:            confirmPassword,
	}
}

func (qq *ChangePasswordCmd) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	// Ensure user is logged in
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to change your password.")
	}

	data, err := autil.FormData[ChangePasswordCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	accountx := ctx.MainCtx().Account
	accountm := account.NewAccount(accountx)

	err = accountm.ChangePassword(ctx, data.CurrentOrTemporaryPassword, data.NewPassword, data.ConfirmPassword)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", events.PasswordChanged.String())

	rw.AddRenderables(widget.NewSnackbarf("Password changed successfully."))
	return nil
}
