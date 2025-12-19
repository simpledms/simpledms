package encryptor

import (
	"filippo.io/age"
)

// global var because otherwise it is not usable in entx.EncryptedString, etc.;
// is nil before the app is unlocked if passphrase protected or before the identity
// is loaded if not passphrase protected
//
// as of 07.04.2025 this is practically no longer nilable because var is set very early
// in app startup, because because it is so sensitive if something goes wrong, we still should
// treat it as nilable because it is nil for a short period at startup
var NilableX25519MainIdentity *age.X25519Identity
