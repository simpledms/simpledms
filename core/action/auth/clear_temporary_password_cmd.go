package auth

import (
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
		FormHelper: autil.NewFormHelper[ClearTemporaryPasswordCmdData](infra, config, widget.T("Clear temporary password")),
	}
}

func (qq *ClearTemporaryPasswordCmd) Data() *ClearTemporaryPasswordCmdData {
	return &ClearTemporaryPasswordCmdData{}
}

func (qq *ClearTemporaryPasswordCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to change your password.")
	}

	accountx := ctx.MainCtx().Account
	accountm := account.NewAccount(accountx)
	accountm.ClearTemporaryPassword(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", events.TemporaryPasswordCleared.String())

	rw.AddRenderables(widget.NewSnackbarf("Temporary password cleared successfully."))
	return nil
}
