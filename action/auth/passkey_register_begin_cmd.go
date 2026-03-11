package auth

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type PasskeyRegisterBeginCmdData struct{}

type PasskeyRegisterBeginCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
}

func NewPasskeyRegisterBeginCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *PasskeyRegisterBeginCmd {
	config := actionx.NewConfig(actions.Route("passkey-register-begin-cmd"), false).EnableSetupSessionAccess()
	return &PasskeyRegisterBeginCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
	}
}

func (qq *PasskeyRegisterBeginCmd) Data() *PasskeyRegisterBeginCmdData {
	return &PasskeyRegisterBeginCmdData{}
}

func (qq *PasskeyRegisterBeginCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to register a passkey.")
	if err != nil {
		return err
	}

	result, err := qq.passkeyService.BeginRegistration(mainCtx, req, mainCtx.Account)
	if err != nil {
		return err
	}

	response := struct {
		ChallengeID string                       `json:"challengeId"`
		Options     *protocol.CredentialCreation `json:"options"`
	}{
		ChallengeID: result.ChallengeID,
		Options:     result.Options,
	}

	return writeJSONResponse(rw, http.StatusOK, response)
}
