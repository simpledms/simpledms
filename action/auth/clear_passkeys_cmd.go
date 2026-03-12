package auth

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/account"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type ClearPasskeysCmdData struct{}

type ClearPasskeysCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewClearPasskeysCmd(infra *common.Infra, actions *Actions) *ClearPasskeysCmd {
	config := actionx.NewConfig(actions.Route("clear-passkeys-cmd"), false)
	return &ClearPasskeysCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ClearPasskeysCmd) Data() *ClearPasskeysCmdData {
	return &ClearPasskeysCmdData{}
}

func (qq *ClearPasskeysCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to manage passkeys.")
	if err != nil {
		return err
	}

	accountm := account2.NewAccount(mainCtx.Account)
	isTenantPolicyEnforced, err := accountm.IsTenantPasskeyAuthEnforced(mainCtx)
	if err != nil {
		log.Println(err)
		return err
	}
	if isTenantPolicyEnforced {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Passkeys cannot be removed because a tenant requires passkey login.")
	}

	err = accountm.ClearPasskeys(mainCtx)
	if err != nil {
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("All passkeys were removed."))

	log.Printf("passkeys cleared account_id=%d", mainCtx.Account.ID)

	return nil
}
