package account

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/simpledms/simpledms/core/db/entmain"
)

type passkeyUser struct {
	account     *entmain.Account
	credentials []webauthn.Credential
}

func newPasskeyUser(accountx *entmain.Account, credentials []webauthn.Credential) *passkeyUser {
	return &passkeyUser{
		account:     accountx,
		credentials: credentials,
	}
}

func (qq *passkeyUser) WebAuthnID() []byte {
	return []byte(qq.account.PublicID.String())
}

func (qq *passkeyUser) WebAuthnName() string {
	accountm := NewAccount(qq.account)
	return accountm.Name()
}

func (qq *passkeyUser) WebAuthnDisplayName() string {
	accountm := NewAccount(qq.account)
	return accountm.Name()
}

func (qq *passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return qq.credentials
}
