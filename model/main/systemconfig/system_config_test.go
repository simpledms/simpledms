package systemconfig

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/encryptor"
)

func TestDecryptMainIdentityRejectsMalformedDecryptedIdentity(t *testing.T) {
	const passphrase = "malformed-identity-passphrase"
	encryptedIdentity := mustEncryptPlaintext(t, passphrase, "not an X25519 identity")

	identity, err := DecryptMainIdentity(encryptedIdentity, passphrase)
	if err == nil {
		t.Fatal("expected malformed decrypted identity to return an error")
	}
	if identity != nil {
		t.Fatal("expected malformed decrypted identity to return no identity")
	}

	previousIdentity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = nil
	t.Cleanup(func() {
		encryptor.NilableX25519MainIdentity = previousIdentity
	})

	systemConfig := NewSystemConfig(
		&entmain.SystemConfig{
			X25519Identity:                    encryptedIdentity,
			IsIdentityEncryptedWithPassphrase: true,
		},
		false,
		false,
		false,
		"",
		"",
		"",
	)
	if err := systemConfig.Unlock(passphrase); err == nil {
		t.Fatal("expected unlock with malformed decrypted identity to return an error")
	}
	if encryptor.NilableX25519MainIdentity != nil {
		t.Fatal("expected malformed decrypted identity to leave the identity unset")
	}
}

func TestSystemConfigWebAuthnRPIDDerivesFromPublicOrigin(t *testing.T) {
	systemConfig := NewSystemConfig(
		&entmain.SystemConfig{},
		false,
		false,
		false,
		"https://app.simpledms.eu",
		"",
		"",
	)

	rpID := systemConfig.WebAuthnRPID()
	if rpID != "app.simpledms.eu" {
		t.Fatalf("expected rp id %q, got %q", "app.simpledms.eu", rpID)
	}
}

func TestSystemConfigWebAuthnRPIDUsesOverride(t *testing.T) {
	systemConfig := NewSystemConfig(
		&entmain.SystemConfig{},
		false,
		false,
		false,
		"https://app.simpledms.eu",
		"auth.simpledms.eu",
		"",
	)

	rpID := systemConfig.WebAuthnRPID()
	if rpID != "auth.simpledms.eu" {
		t.Fatalf("expected rp id %q, got %q", "auth.simpledms.eu", rpID)
	}
}

func TestSystemConfigAbsoluteURLUsesPublicOrigin(t *testing.T) {
	systemConfig := NewSystemConfig(
		&entmain.SystemConfig{},
		false,
		false,
		false,
		"https://app.simpledms.eu",
		"",
		"",
	)

	absURL := systemConfig.AbsoluteURL("/-/auth/sign-in-cmd")
	if absURL != "https://app.simpledms.eu/-/auth/sign-in-cmd" {
		t.Fatalf("expected absolute url %q, got %q", "https://app.simpledms.eu/-/auth/sign-in-cmd", absURL)
	}
}

func mustEncryptPlaintext(t *testing.T, passphrase, plaintext string) []byte {
	t.Helper()

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		t.Fatalf("create scrypt recipient: %v", err)
	}
	recipient.SetWorkFactor(1)

	ciphertext := bytes.NewBuffer(nil)
	writer, err := age.Encrypt(ciphertext, recipient)
	if err != nil {
		t.Fatalf("create encryptor: %v", err)
	}
	if _, err := io.Copy(writer, strings.NewReader(plaintext)); err != nil {
		t.Fatalf("encrypt plaintext: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close encryptor: %v", err)
	}

	return ciphertext.Bytes()
}
