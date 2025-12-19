package auth

import (
	"net/http"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/entmain/account"
	"github.com/simpledms/simpledms/app/simpledms/entx"
	"github.com/simpledms/simpledms/app/simpledms/model/modelmain"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteAccountCmdData struct {
	AccountID string `validate:"required"`
}

type DeleteAccountCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteAccountCmd(infra *common.Infra, actions *Actions) *DeleteAccountCmd {
	config := actionx.NewConfig(actions.Route("delete-account-cmd"), false)
	return &DeleteAccountCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteAccountCmd) Data(accountID string) *DeleteAccountCmdData {
	return &DeleteAccountCmdData{
		AccountID: accountID,
	}
}

func (qq *DeleteAccountCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteAccountCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if ctx.MainCtx().Account.PublicID.String() != data.AccountID {
		return e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to delete this account.")
	}

	// could also use ctx.MainCtx().Account for the moment, but this is more robust against future changes
	accountx := ctx.MainCtx().MainTx.Account.Query().Where(account.PublicID(entx.NewCIText(data.AccountID))).OnlyX(ctx)
	accountm := modelmain.NewAccount(accountx)
	accountm.UnsafeDelete(ctx)

	rw.AddRenderables(wx.NewSnackbarf("Account deleted."))
	http.Redirect(rw, req.Request, "/", http.StatusSeeOther)

	return nil
}
