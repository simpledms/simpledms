package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SetInitialPasswordData struct {
	NewPassword     string `validate:"required" form_attr_type:"password"`
	ConfirmPassword string `validate:"required" form_attr_type:"password"` // eqfield=NewPassword was removed
}

type SetInitialPassword struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SetInitialPasswordData]
}

func NewSetInitialPassword(infra *common.Infra, actions *Actions) *SetInitialPassword {
	config := actionx.NewConfig(actions.Route("set-initial-password"), false)
	return &SetInitialPassword{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[SetInitialPasswordData](infra, config, wx.T("Set password")),
	}
}

func (qq *SetInitialPassword) Data(newPassword, confirmPassword string) *SetInitialPasswordData {
	return &SetInitialPasswordData{
		NewPassword:     newPassword,
		ConfirmPassword: confirmPassword,
	}
}

func (qq *SetInitialPassword) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// Ensure user is logged in
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to change your password.")
	}

	data, err := autil.FormData[SetInitialPasswordData](rw, req, ctx)
	if err != nil {
		return err
	}

	accountx := ctx.MainCtx().Account
	accountm := modelmain.NewAccount(accountx)

	// Set the new password
	err = accountm.SetPassword(ctx, data.NewPassword, data.ConfirmPassword)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.InitialPasswordSet.String())

	rw.AddRenderables(wx.NewSnackbarf("Initial password set successfully."))
	return nil
}
