package auth

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type PasskeySignInBeginCmdData struct {
}

type PasskeySignInBeginCmd struct {
	infra              *common.Infra
	actions            *Actions
	passkeyService     *account.PasskeyService
	requestRateLimiter *account.RequestRateLimiter
	*actionx.Config
}

func NewPasskeySignInBeginCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account.PasskeyService,
	requestRateLimiter *account.RequestRateLimiter,
) *PasskeySignInBeginCmd {
	config := actionx.NewConfig(actions.Route("passkey-sign-in-begin-cmd"), false)
	return &PasskeySignInBeginCmd{
		infra:              infra,
		actions:            actions,
		passkeyService:     passkeyService,
		requestRateLimiter: requestRateLimiter,
		Config:             config,
	}
}

func (qq *PasskeySignInBeginCmd) Data() *PasskeySignInBeginCmdData {
	return &PasskeySignInBeginCmdData{}
}

func (qq *PasskeySignInBeginCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	if !qq.requestRateLimiter.Allow(
		rateLimitKey("passkey-begin-ip", clientIPFromRequest(req)),
		passkeyBeginRateLimitWindow,
		passkeyBeginRateLimitPerIP,
	) {
		return e.NewHTTPErrorf(http.StatusTooManyRequests, "Too many passkey requests. Please try again shortly.")
	}
	if !qq.requestRateLimiter.Allow(
		"passkey-begin-global",
		passkeyBeginRateLimitWindow,
		passkeyBeginRateLimitGlobal,
	) {
		return e.NewHTTPErrorf(http.StatusTooManyRequests, "Too many passkey requests. Please try again shortly.")
	}

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
