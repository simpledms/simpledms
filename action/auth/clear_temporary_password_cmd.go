package auth

import (
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

type ClearTemporaryPasswordCmdData struct{}

type ClearTemporaryPasswordCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ClearTemporaryPasswordCmdData]
}

func NewClearTemporaryPasswordCmd(infra *common.Infra, actions *Actions) *ClearTemporaryPasswordCmd {
	config := actionx.NewConfig(actions.Route("clear-temporary-password-cmd"), false)
	return &ClearTemporaryPasswordCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[ClearTemporaryPasswordCmdData](infra, config, wx.T("Clear temporary password")),
	}
}

func (qq *ClearTemporaryPasswordCmd) Data() *ClearTemporaryPasswordCmdData {
	return &ClearTemporaryPasswordCmdData{}
}

func (qq *ClearTemporaryPasswordCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to change your password.")
	}

	accountx := ctx.MainCtx().Account
	accountm := account.NewAccount(accountx)
	accountm.ClearTemporaryPassword(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.TemporaryPasswordCleared.String())

	rw.AddRenderables(wx.NewSnackbarf("Temporary password cleared successfully."))
	return nil
}
