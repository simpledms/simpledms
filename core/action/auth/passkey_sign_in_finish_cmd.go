package auth

import (
	"encoding/json"
	"net/http"

	"github.com/simpledms/simpledms/core/common"
	account2 "github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/ui/uix/route"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type PasskeySignInFinishCmdData struct{}

type PasskeySignInFinishCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
}

func NewPasskeySignInFinishCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *PasskeySignInFinishCmd {
	config := actionx.NewConfig(actions.Route("passkey-sign-in-finish-cmd"), false)
	return &PasskeySignInFinishCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
	}
}

func (qq *PasskeySignInFinishCmd) Data() *PasskeySignInFinishCmdData {
	return &PasskeySignInFinishCmdData{}
}

func (qq *PasskeySignInFinishCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	var requestBody struct {
		ChallengeID string          `json:"challengeId"`
		Credential  json.RawMessage `json:"credential"`
	}
	if err := decodeJSONBody(req, &requestBody); err != nil {
		return err
	}

	if requestBody.ChallengeID == "" || len(requestBody.Credential) == 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid passkey response payload.")
	}

	accountx, err := qq.passkeyService.FinishDiscoverableSignIn(
		ctx,
		req,
		requestBody.ChallengeID,
		requestBody.Credential,
	)
	if err != nil {
		return err
	}

	err = createAccountSession(
		rw,
		req,
		ctx,
		accountx,
		false,
		qq.infra.SystemConfig().AllowInsecureCookies(),
	)
	if err != nil {
		return err
	}

	return writeJSONResponse(rw, http.StatusOK, struct {
		Redirect string `json:"redirect"`
	}{
		Redirect: route.Dashboard(),
	})
}
