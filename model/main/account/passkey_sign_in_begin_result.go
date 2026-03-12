package account

import "github.com/go-webauthn/webauthn/protocol"

type PasskeySignInBeginResult struct {
	ChallengeID string
	Options     *protocol.CredentialAssertion
}
