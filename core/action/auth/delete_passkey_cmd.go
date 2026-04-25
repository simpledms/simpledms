package auth

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/db/entmain/passkeycredential"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type DeletePasskeyCmdData struct {
	PasskeyID string `validate:"required"`
}

type DeletePasskeyCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account.PasskeyService
	*actionx.Config
}

func NewDeletePasskeyCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account.PasskeyService,
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

func (qq *DeletePasskeyCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	accountm := account.NewAccount(mainCtx.Account)
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
	rw.Header().Set("HX-Trigger", events.AccountUpdated.String())

	return nil
}
