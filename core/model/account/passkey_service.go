package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/simpledms/simpledms/core/util"
	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/passkeycredential"
	"github.com/simpledms/simpledms/db/entmain/webauthnchallenge"
	"github.com/simpledms/simpledms/db/entx"
)

const (
	passkeyCeremonyAuthentication = "authentication"
	passkeyCeremonyRegistration   = "registration"

	passkeyRateLimitWindow = time.Minute

	passkeyAuthenticationBeginLimit  = 40
	passkeyAuthenticationFinishLimit = 80
	passkeyRegistrationBeginLimit    = 20
	passkeyRegistrationFinishLimit   = 40
)

type PasskeyService struct {
	publicOrigin        string
	rpID                string
	rpName              string
	recoveryRateLimiter *attemptRateLimiter
}

func NewPasskeyService(publicOrigin, rpID, rpName string) *PasskeyService {
	return &PasskeyService{
		publicOrigin:        strings.TrimSpace(publicOrigin),
		rpID:                strings.TrimSpace(rpID),
		rpName:              strings.TrimSpace(rpName),
		recoveryRateLimiter: newAttemptRateLimiter(),
	}
}

func (qq *PasskeyService) BeginDiscoverableSignIn(
	ctx ctxx.Context,
	req *httpx.Request,
) (*PasskeySignInBeginResult, error) {
	err := qq.enforcePasskeyRateLimit(
		ctx,
		passkeyCeremonyAuthentication,
		nil,
		qq.passkeyClientKey(req),
		passkeyAuthenticationBeginLimit,
	)
	if err != nil {
		return nil, err
	}

	wa, err := qq.newWebAuthn(req)
	if err != nil {
		return nil, err
	}

	options, sessionData, err := wa.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		return nil, err
	}

	challengeID, err := qq.createPasskeyChallenge(
		ctx,
		passkeyCeremonyAuthentication,
		nil,
		qq.passkeyClientKey(req),
		sessionData,
	)
	if err != nil {
		return nil, err
	}

	return &PasskeySignInBeginResult{
		ChallengeID: challengeID,
		Options:     options,
	}, nil
}

func (qq *PasskeyService) FinishDiscoverableSignIn(
	ctx ctxx.Context,
	req *httpx.Request,
	challengeID string,
	credentialJSON json.RawMessage,
) (*entmain.Account, error) {
	err := qq.enforcePasskeyRateLimit(
		ctx,
		passkeyCeremonyAuthentication,
		nil,
		qq.passkeyClientKey(req),
		passkeyAuthenticationFinishLimit,
	)
	if err != nil {
		return nil, err
	}

	challengex, sessionData, err := qq.loadPasskeyChallenge(
		ctx,
		challengeID,
		passkeyCeremonyAuthentication,
		nil,
	)
	if err != nil {
		return nil, err
	}
	qq.markPasskeyChallengeUsed(ctx, challengex)

	wa, err := qq.newWebAuthn(req)
	if err != nil {
		return nil, err
	}

	var accountx *entmain.Account
	var credentialRow *entmain.PasskeyCredential

	userHandler := func(rawID, userHandle []byte) (webauthn.User, error) {
		resolvedAccountx, resolvedCredentialRow, user, err := qq.passkeyUserForCredentialID(ctx, rawID)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		accountx = resolvedAccountx
		credentialRow = resolvedCredentialRow
		return user, nil
	}

	_, credential, err := wa.FinishPasskeyLogin(
		userHandler,
		*sessionData,
		qq.passkeyCredentialRequest(req, credentialJSON),
	)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Invalid passkey sign-in.")
	}

	if accountx == nil || credentialRow == nil {
		return nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Passkey sign-in failed.")
	}

	accountm := NewAccount(accountx)
	isRequired, err := accountm.IsPasskeyLoginRequired(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !isRequired {
		return nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Passkey login is not enabled for this account.")
	}

	encodedCredential, err := qq.encodePasskeyCredential(credential)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	credentialRow.Update().
		SetCredentialJSON(encodedCredential).
		SetLastUsedAt(time.Now()).
		SaveX(ctx)

	log.Printf("passkey sign-in success account_id=%d credential_id_len=%d", accountx.ID, len(credential.ID))

	return accountx, nil
}

func (qq *PasskeyService) BeginRegistration(
	ctx ctxx.Context,
	req *httpx.Request,
	accountx *entmain.Account,
) (*PasskeyRegistrationBeginResult, error) {
	accountID := accountx.ID
	err := qq.enforcePasskeyRateLimit(
		ctx,
		passkeyCeremonyRegistration,
		&accountID,
		qq.passkeyClientKey(req),
		passkeyRegistrationBeginLimit,
	)
	if err != nil {
		return nil, err
	}

	wa, err := qq.newWebAuthn(req)
	if err != nil {
		return nil, err
	}

	user, err := qq.passkeyUserForAccount(ctx, accountx)
	if err != nil {
		return nil, err
	}

	options, sessionData, err := wa.BeginRegistration(
		user,
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationRequired,
		}),
	)
	if err != nil {
		return nil, err
	}

	challengeID, err := qq.createPasskeyChallenge(
		ctx,
		passkeyCeremonyRegistration,
		&accountID,
		qq.passkeyClientKey(req),
		sessionData,
	)
	if err != nil {
		return nil, err
	}

	return &PasskeyRegistrationBeginResult{
		ChallengeID: challengeID,
		Options:     options,
	}, nil
}

func (qq *PasskeyService) FinishRegistration(
	ctx ctxx.Context,
	req *httpx.Request,
	accountx *entmain.Account,
	challengeID string,
	credentialJSON json.RawMessage,
	credentialName string,
) ([]string, error) {
	accountID := accountx.ID
	err := qq.enforcePasskeyRateLimit(
		ctx,
		passkeyCeremonyRegistration,
		&accountID,
		qq.passkeyClientKey(req),
		passkeyRegistrationFinishLimit,
	)
	if err != nil {
		return nil, err
	}

	challengex, sessionData, err := qq.loadPasskeyChallenge(
		ctx,
		challengeID,
		passkeyCeremonyRegistration,
		&accountID,
	)
	if err != nil {
		return nil, err
	}
	qq.markPasskeyChallengeUsed(ctx, challengex)

	wa, err := qq.newWebAuthn(req)
	if err != nil {
		return nil, err
	}

	user, err := qq.passkeyUserForAccount(ctx, accountx)
	if err != nil {
		return nil, err
	}

	credential, err := wa.FinishRegistration(
		user,
		*sessionData,
		qq.passkeyCredentialRequest(req, credentialJSON),
	)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Passkey registration failed.")
	}

	encodedCredential, err := qq.encodePasskeyCredential(credential)
	if err != nil {
		return nil, err
	}

	credentialName = strings.TrimSpace(credentialName)
	if credentialName == "" {
		credentialsCount := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
			Where(passkeycredential.AccountID(accountID)).
			CountX(ctx)
		credentialName = fmt.Sprintf("Passkey %d", credentialsCount+1)
	}

	ctx.VisitorCtx().MainTx.PasskeyCredential.Create().
		SetAccountID(accountID).
		SetCredentialID(credential.ID).
		SetCredentialJSON(encodedCredential).
		SetName(credentialName).
		SaveX(ctx)

	accountm := NewAccount(accountx)
	accountm.EnablePasskeyLogin(ctx)

	recoveryCodes, err := accountm.GenerateAndSetPasskeyRecoveryCodes(ctx, 10)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Printf("passkey registered account_id=%d credential_id_len=%d", accountID, len(credential.ID))

	return recoveryCodes, nil
}

func (qq *PasskeyService) RegenerateRecoveryCodes(
	ctx ctxx.Context,
	accountx *entmain.Account,
	count int,
) ([]string, error) {
	hasPasskey, err := qq.HasPasskeyCredentials(ctx, accountx.ID)
	if err != nil {
		return nil, err
	}
	if !hasPasskey {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "You need at least one passkey before creating backup codes.")
	}

	accountm := NewAccount(accountx)
	recoveryCodes, err := accountm.GenerateAndSetPasskeyRecoveryCodes(ctx, count)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return recoveryCodes, nil
}

func (qq *PasskeyService) SignInWithRecoveryCode(
	ctx ctxx.Context,
	email string,
	recoveryCode string,
) (*entmain.Account, error) {
	invalidRecoveryErr := e.NewHTTPErrorf(http.StatusUnauthorized, "Invalid backup sign-in credentials.")

	accountx, err := qq.AccountByEmail(ctx, email)
	if err != nil {
		var httpErr *e.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode() != http.StatusBadRequest {
			return nil, err
		}
		return nil, invalidRecoveryErr
	}

	accountm := NewAccount(accountx)
	recoveryRateLimitKey := strings.ToLower(strings.TrimSpace(email))
	if recoveryRateLimitKey != "" && !qq.recoveryRateLimiter.Allow(recoveryRateLimitKey, 10*time.Second) {
		return nil, e.NewHTTPErrorf(
			http.StatusTooManyRequests,
			"Too many backup sign-in attempts. Please try again in 10 seconds.",
		)
	}
	qq.recoveryRateLimiter.Record(recoveryRateLimitKey)

	isPasskeyRequired, err := accountm.IsPasskeyLoginRequired(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !isPasskeyRequired {
		return nil, invalidRecoveryErr
	}

	isValid, err := accountm.ConsumePasskeyRecoveryCode(ctx, recoveryCode)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !isValid {
		return nil, invalidRecoveryErr
	}

	return accountx, nil
}

func (qq *PasskeyService) AssistedRecoveryCodesForEmail(
	ctx ctxx.Context,
	targetEmail string,
	count int,
) (*entmain.Account, []string, error) {
	targetAccountx, err := qq.AccountByEmail(ctx, targetEmail)
	if err != nil {
		return nil, nil, err
	}

	hasPasskey, err := qq.HasPasskeyCredentials(ctx, targetAccountx.ID)
	if err != nil {
		return nil, nil, err
	}
	if !hasPasskey {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "Target account has no passkeys configured.")
	}

	targetAccountm := NewAccount(targetAccountx)
	recoveryCodes, err := targetAccountm.GenerateAndSetPasskeyRecoveryCodes(ctx, count)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}

	return targetAccountx, recoveryCodes, nil
}

func (qq *PasskeyService) OwnCredentialByPublicID(
	ctx ctxx.Context,
	accountID int64,
	passkeyPublicID string,
) (*entmain.PasskeyCredential, error) {
	credentialx, err := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
		Where(
			passkeycredential.PublicID(entx.NewCIText(passkeyPublicID)),
			passkeycredential.AccountID(accountID),
		).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Passkey not found.")
		}

		log.Println(err)
		return nil, err
	}

	return credentialx, nil
}

func (qq *PasskeyService) AccountByEmail(ctx ctxx.Context, email string) (*entmain.Account, error) {
	accountx, err := ctx.VisitorCtx().MainTx.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Account not found.")
		}

		log.Println(err)
		return nil, err
	}

	return accountx, nil
}

func (qq *PasskeyService) HasPasskeyCredentials(ctx ctxx.Context, accountID int64) (bool, error) {
	hasPasskey, err := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(accountID)).
		Exist(ctx)
	if err != nil {
		log.Println(err)
		return false, err
	}

	return hasPasskey, nil
}

func (qq *PasskeyService) newWebAuthn(req *httpx.Request) (*webauthn.WebAuthn, error) {
	origin := qq.publicOrigin
	if origin == "" {
		scheme := "https"
		if req.TLS == nil {
			scheme = "http"
		}
		origin = fmt.Sprintf("%s://%s", scheme, req.Host)
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Invalid passkey origin configuration.")
	}

	rpID := qq.rpID
	if rpID == "" {
		rpID = originURL.Hostname()
	}
	if rpID == "" {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Missing passkey rp id configuration.")
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         qq.rpName,
		RPOrigins:             []string{origin},
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not initialize passkey service.")
	}

	return wa, nil
}

func (qq *PasskeyService) createPasskeyChallenge(
	ctx ctxx.Context,
	ceremony string,
	nilableAccountID *int64,
	clientKey string,
	sessionData *webauthn.SessionData,
) (string, error) {
	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		log.Println(err)
		return "", err
	}

	challengeID := util.NewPublicID()
	challengeExpiry := sessionData.Expires
	if challengeExpiry.IsZero() {
		challengeExpiry = time.Now().Add(5 * time.Minute)
	}

	query := ctx.VisitorCtx().MainTx.WebAuthnChallenge.Create().
		SetChallengeID(challengeID).
		SetCeremony(ceremony).
		SetSessionDataJSON(sessionDataJSON).
		SetExpiresAt(challengeExpiry)

	if nilableAccountID != nil {
		query.SetAccountID(*nilableAccountID)
	}
	query.SetNillableClientKey(qq.nilableString(strings.TrimSpace(clientKey)))

	if err := query.Exec(ctx); err != nil {
		log.Println(err)
		return "", err
	}

	return challengeID, nil
}

func (qq *PasskeyService) loadPasskeyChallenge(
	ctx ctxx.Context,
	challengeID string,
	ceremony string,
	nilableAccountID *int64,
) (*entmain.WebAuthnChallenge, *webauthn.SessionData, error) {
	query := ctx.VisitorCtx().MainTx.WebAuthnChallenge.Query().Where(
		webauthnchallenge.ChallengeID(challengeID),
		webauthnchallenge.Ceremony(ceremony),
		webauthnchallenge.UsedAtIsNil(),
		webauthnchallenge.ExpiresAtGT(time.Now()),
	)

	if nilableAccountID != nil {
		query.Where(webauthnchallenge.AccountID(*nilableAccountID))
	}

	challengex, err := query.Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "Passkey challenge is invalid or expired.")
		}
		log.Println(err)
		return nil, nil, err
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(challengex.SessionDataJSON, &sessionData); err != nil {
		log.Println(err)
		return nil, nil, err
	}

	return challengex, &sessionData, nil
}

func (qq *PasskeyService) enforcePasskeyRateLimit(
	ctx ctxx.Context,
	ceremony string,
	nilableAccountID *int64,
	clientKey string,
	requestLimit int,
) error {
	qq.cleanupExpiredAndUsedPasskeyChallenges(ctx)

	query := ctx.VisitorCtx().MainTx.WebAuthnChallenge.Query().Where(
		webauthnchallenge.Ceremony(ceremony),
		webauthnchallenge.CreatedAtGT(time.Now().Add(-passkeyRateLimitWindow)),
	)

	if nilableAccountID != nil {
		query.Where(webauthnchallenge.AccountID(*nilableAccountID))
	} else {
		query.Where(webauthnchallenge.ClientKey(strings.TrimSpace(clientKey)))
	}

	requestCount, err := query.Count(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	if requestCount >= requestLimit {
		return e.NewHTTPErrorf(http.StatusTooManyRequests, "Too many passkey requests. Please try again shortly.")
	}

	return nil
}

func (qq *PasskeyService) cleanupExpiredAndUsedPasskeyChallenges(ctx ctxx.Context) {
	now := time.Now()
	retainedUsedUntil := now.Add(-passkeyRateLimitWindow)

	_, err := ctx.VisitorCtx().MainTx.WebAuthnChallenge.Delete().Where(
		webauthnchallenge.Or(
			webauthnchallenge.ExpiresAtLTE(now),
			webauthnchallenge.And(
				webauthnchallenge.UsedAtNotNil(),
				webauthnchallenge.UsedAtLT(retainedUsedUntil),
			),
		),
	).Exec(ctx)
	if err != nil {
		log.Println(err)
	}
}

func (qq *PasskeyService) markPasskeyChallengeUsed(ctx ctxx.Context, challengex *entmain.WebAuthnChallenge) {
	challengex.Update().SetUsedAt(time.Now()).ExecX(ctx)
}

func (qq *PasskeyService) passkeyClientKey(req *httpx.Request) string {
	remoteAddr := strings.TrimSpace(req.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = host
	}

	if remoteAddr == "" {
		return "unknown"
	}

	return strings.ToLower(remoteAddr)
}

func (qq *PasskeyService) nilableString(val string) *string {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}

	return &val
}

func (qq *PasskeyService) passkeyCredentialRequest(req *httpx.Request, credentialJSON json.RawMessage) *http.Request {
	credentialRequest := req.Request.Clone(req.Context())
	credentialRequest.Header = req.Header.Clone()
	credentialRequest.Header.Set("Content-Type", "application/json")
	credentialRequest.Body = io.NopCloser(bytes.NewReader(credentialJSON))
	credentialRequest.ContentLength = int64(len(credentialJSON))
	return credentialRequest
}

func (qq *PasskeyService) decodePasskeyCredential(
	credentialx *entmain.PasskeyCredential,
) (*webauthn.Credential, error) {
	var credential webauthn.Credential
	if err := json.Unmarshal(credentialx.CredentialJSON, &credential); err != nil {
		log.Println(err)
		return nil, err
	}
	return &credential, nil
}

func (qq *PasskeyService) encodePasskeyCredential(credential *webauthn.Credential) ([]byte, error) {
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return credentialJSON, nil
}

func (qq *PasskeyService) passkeyUserForAccount(
	ctx ctxx.Context,
	accountx *entmain.Account,
) (*passkeyUser, error) {
	credentialRows, err := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(accountx.ID)).
		All(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	credentials := make([]webauthn.Credential, 0, len(credentialRows))
	for _, credentialRow := range credentialRows {
		credential, err := qq.decodePasskeyCredential(credentialRow)
		if err != nil {
			log.Println(err)
			continue
		}
		credentials = append(credentials, *credential)
	}

	return newPasskeyUser(accountx, credentials), nil
}

func (qq *PasskeyService) passkeyUserForCredentialID(
	ctx ctxx.Context,
	credentialID []byte,
) (*entmain.Account, *entmain.PasskeyCredential, *passkeyUser, error) {
	credentialRow, err := ctx.VisitorCtx().MainTx.PasskeyCredential.Query().
		Where(passkeycredential.CredentialID(credentialID)).
		WithAccount().
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil, nil, nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Unknown passkey credential.")
		}
		log.Println(err)
		return nil, nil, nil, err
	}

	if credentialRow.Edges.Account == nil {
		return nil, nil, nil, e.NewHTTPErrorf(http.StatusUnauthorized, "Passkey account not found.")
	}

	user, err := qq.passkeyUserForAccount(ctx, credentialRow.Edges.Account)
	if err != nil {
		log.Println(err)
		return nil, nil, nil, err
	}

	return credentialRow.Edges.Account, credentialRow, user, nil
}
