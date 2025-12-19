package auth

import (
	"net/http"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	"github.com/simpledms/simpledms/app/simpledms/model/modelmain"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ClearTemporaryPasswordData struct{}

type ClearTemporaryPassword struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ClearTemporaryPasswordData]
}

func NewClearTemporaryPassword(infra *common.Infra, actions *Actions) *ClearTemporaryPassword {
	config := actionx.NewConfig(actions.Route("clear-temporary-password"), false)
	return &ClearTemporaryPassword{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[ClearTemporaryPasswordData](infra, config, wx.T("Clear temporary password")),
	}
}

func (qq *ClearTemporaryPassword) Data() *ClearTemporaryPasswordData {
	return &ClearTemporaryPasswordData{}
}

func (qq *ClearTemporaryPassword) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if !ctx.IsMainCtx() {
		return e.NewHTTPErrorf(http.StatusUnauthorized, "You must be logged in to change your password.")
	}

	accountx := ctx.MainCtx().Account
	accountm := modelmain.NewAccount(accountx)
	accountm.ClearTemporaryPassword(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.TemporaryPasswordCleared.String())

	rw.AddRenderables(wx.NewSnackbarf("Temporary password cleared successfully."))
	return nil
}
