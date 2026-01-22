package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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
	config := actionx.NewConfig(actions.Route("change-password"), false)
	return &ChangePasswordCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[ChangePasswordCmdData](
			infra,
			config,
			wx.T("Change password"),
			wx.T("Change"),
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
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
	rw.Header().Set("HX-Trigger", event.PasswordChanged.String())

	rw.AddRenderables(wx.NewSnackbarf("Password changed successfully."))
	return nil
}
