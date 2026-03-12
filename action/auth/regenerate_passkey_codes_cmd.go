package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type RegeneratePasskeyCodesCmdData struct{}

type RegeneratePasskeyCodesCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	store          *account2.PasskeyRecoveryCodesStore
	*actionx.Config
}

func NewRegeneratePasskeyCodesCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
	store *account2.PasskeyRecoveryCodesStore,
) *RegeneratePasskeyCodesCmd {
	config := actionx.NewConfig(actions.Route("regenerate-passkey-codes-cmd"), false).EnableSetupSessionAccess()
	return &RegeneratePasskeyCodesCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		store:          store,
		Config:         config,
	}
}

func (qq *RegeneratePasskeyCodesCmd) Data() *RegeneratePasskeyCodesCmdData {
	return &RegeneratePasskeyCodesCmdData{}
}

func (qq *RegeneratePasskeyCodesCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to manage backup codes.")
	if err != nil {
		return err
	}

	recoveryCodes, err := qq.passkeyService.RegenerateRecoveryCodes(mainCtx, mainCtx.Account, 10)
	if err != nil {
		return err
	}

	log.Printf("passkey backup codes regenerated account_id=%d", mainCtx.Account.ID)

	recoveryCodesToken := qq.store.Store(recoveryCodes)
	triggerPayload, err := json.Marshal(map[string]any{
		event.AccountUpdated.String(): true,
		"passkeyRecoveryCodesGenerated": map[string]string{
			"recoveryCodesToken": recoveryCodesToken,
		},
	})
	if err != nil {
		log.Println(err)
		rw.Header().Set("HX-Trigger", event.AccountUpdated.String())
	} else {
		rw.Header().Set("HX-Trigger", string(triggerPayload))
	}

	return writeJSONResponse(rw, http.StatusOK, struct {
		RecoveryCodesToken string `json:"recoveryCodesToken"`
	}{
		RecoveryCodesToken: recoveryCodesToken,
	})
}
