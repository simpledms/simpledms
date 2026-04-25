package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type RegeneratePasskeyCodesCmdData struct{}

type RegeneratePasskeyCodesCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account.PasskeyService
	store          *account.PasskeyRecoveryCodesStore
	*actionx.Config
}

func NewRegeneratePasskeyCodesCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account.PasskeyService,
	store *account.PasskeyRecoveryCodesStore,
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

func (qq *RegeneratePasskeyCodesCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		events.AccountUpdated.String(): true,
		"passkeyRecoveryCodesGenerated": map[string]string{
			"recoveryCodesToken": recoveryCodesToken,
		},
	})
	if err != nil {
		log.Println(err)
		rw.Header().Set("HX-Trigger", events.AccountUpdated.String())
	} else {
		rw.Header().Set("HX-Trigger", string(triggerPayload))
	}

	return writeJSONResponse(rw, http.StatusOK, struct {
		RecoveryCodesToken string `json:"recoveryCodesToken"`
	}{
		RecoveryCodesToken: recoveryCodesToken,
	})
}
