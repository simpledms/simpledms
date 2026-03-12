package auth

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type PasskeySignInBeginCmdData struct{}

type PasskeySignInBeginCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
}

func NewPasskeySignInBeginCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *PasskeySignInBeginCmd {
	config := actionx.NewConfig(actions.Route("passkey-sign-in-begin-cmd"), false)
	return &PasskeySignInBeginCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
	}
}

func (qq *PasskeySignInBeginCmd) Data() *PasskeySignInBeginCmdData {
	return &PasskeySignInBeginCmdData{}
}

func (qq *PasskeySignInBeginCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	result, err := qq.passkeyService.BeginDiscoverableSignIn(ctx, req)
	if err != nil {
		return err
	}

	response := struct {
		ChallengeID string                        `json:"challengeId"`
		Options     *protocol.CredentialAssertion `json:"options"`
	}{
		ChallengeID: result.ChallengeID,
		Options:     result.Options,
	}

	return writeJSONResponse(rw, http.StatusOK, response)
}
