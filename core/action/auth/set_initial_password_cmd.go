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

type SetInitialPasswordCmdData struct {
	NewPassword     string `validate:"required" form_attr_type:"password"`
	ConfirmPassword string `validate:"required" form_attr_type:"password"` // eqfield=NewPassword was removed
}

type SetInitialPasswordCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SetInitialPasswordCmdData]
}

func NewSetInitialPasswordCmd(infra *common.Infra, actions *Actions) *SetInitialPasswordCmd {
	config := actionx.NewConfig(actions.Route("set-initial-password-cmd"), false)
	return &SetInitialPasswordCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[SetInitialPasswordCmdData](infra, config, widget.T("Set password")),
	}
}

func (qq *SetInitialPasswordCmd) Data(newPassword, confirmPassword string) *SetInitialPasswordCmdData {
	return &SetInitialPasswordCmdData{
		NewPassword:     newPassword,
		ConfirmPassword: confirmPassword,
	}
}

func (qq *SetInitialPasswordCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	// Ensure user is logged in
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to change your password.")
	}

	data, err := autil.FormData[SetInitialPasswordCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	accountx := ctx.MainCtx().Account
	accountm := account.NewAccount(accountx)

	// Set the new password
	err = accountm.SetPassword(ctx, data.NewPassword, data.ConfirmPassword)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", events.InitialPasswordSet.String())

	rw.AddRenderables(widget.NewSnackbarf("Initial password set successfully."))
	return nil
}
