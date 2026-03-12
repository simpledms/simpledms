package auth

import (
	"encoding/json"
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type PasskeyRegisterFinishCmdData struct{}

type PasskeyRegisterFinishCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	store          *account2.PasskeyRecoveryCodesStore
	*actionx.Config
}

func NewPasskeyRegisterFinishCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
	store *account2.PasskeyRecoveryCodesStore,
) *PasskeyRegisterFinishCmd {
	config := actionx.NewConfig(actions.Route("passkey-register-finish-cmd"), false).EnableSetupSessionAccess()
	return &PasskeyRegisterFinishCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		store:          store,
		Config:         config,
	}
}

func (qq *PasskeyRegisterFinishCmd) Data() *PasskeyRegisterFinishCmdData {
	return &PasskeyRegisterFinishCmdData{}
}

func (qq *PasskeyRegisterFinishCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to register a passkey.")
	if err != nil {
		return err
	}

	var requestBody struct {
		ChallengeID string          `json:"challengeId"`
		Credential  json.RawMessage `json:"credential"`
		Name        string          `json:"name"`
	}
	if err := decodeJSONBody(req, &requestBody); err != nil {
		return err
	}

	if requestBody.ChallengeID == "" || len(requestBody.Credential) == 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid passkey registration payload.")
	}

	recoveryCodes, err := qq.passkeyService.FinishRegistration(
		mainCtx,
		req,
		mainCtx.Account,
		requestBody.ChallengeID,
		requestBody.Credential,
		requestBody.Name,
	)
	if err != nil {
		return err
	}

	var recoveryCodesToken string
	if len(recoveryCodes) > 0 {
		recoveryCodesToken = qq.store.Store(recoveryCodes)
	}

	return writeJSONResponse(rw, http.StatusOK, struct {
		RecoveryCodesToken string `json:"recoveryCodesToken,omitempty"`
	}{
		RecoveryCodesToken: recoveryCodesToken,
	})
}
