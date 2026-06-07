package account

import "github.com/go-webauthn/webauthn/protocol"

type PasskeyRegistrationBeginResult struct {
	ChallengeID string
	Options     *protocol.CredentialCreation
}
