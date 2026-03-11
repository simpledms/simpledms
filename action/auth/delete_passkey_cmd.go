package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/passkeycredential"
	account2 "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeletePasskeyCmdData struct {
	PasskeyID string `validate:"required"`
}

type DeletePasskeyCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
}

func NewDeletePasskeyCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *DeletePasskeyCmd {
	config := actionx.NewConfig(actions.Route("delete-passkey-cmd"), false)
	return &DeletePasskeyCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
	}
}

func (qq *DeletePasskeyCmd) Data(passkeyID string) *DeletePasskeyCmdData {
	return &DeletePasskeyCmdData{
		PasskeyID: passkeyID,
	}
}

func (qq *DeletePasskeyCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to manage passkeys.")
	if err != nil {
		return err
	}

	data, err := autil.FormData[DeletePasskeyCmdData](rw, req, mainCtx)
	if err != nil {
		return err
	}

	credentialx, err := qq.passkeyService.OwnCredentialByPublicID(mainCtx, mainCtx.Account.ID, data.PasskeyID)
	if err != nil {
		return err
	}

	accountm := account2.NewAccount(mainCtx.Account)
	isTenantPasskeyRequired, err := accountm.IsTenantPasskeyAuthEnforced(mainCtx)
	if err != nil {
		log.Println(err)
		return err
	}

	passkeyCount := mainCtx.MainTx.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(mainCtx.Account.ID)).
		CountX(mainCtx)
	if isTenantPasskeyRequired && passkeyCount <= 1 {
		return e.NewHTTPErrorf(
			http.StatusBadRequest,
			"A tenant requires passkey login, so at least one passkey must remain.",
		)
	}

	mainCtx.MainTx.PasskeyCredential.DeleteOneID(credentialx.ID).ExecX(mainCtx)

	_, err = accountm.DisablePasskeyLoginAndClearRecoveryCodesIfNoCredentials(mainCtx)
	if err != nil {
		return err
	}

	log.Printf("passkey deleted account_id=%d passkey_id=%s", mainCtx.Account.ID, data.PasskeyID)

	rw.AddRenderables(wx.NewSnackbarf("Passkey removed."))
	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())

	return nil
}
