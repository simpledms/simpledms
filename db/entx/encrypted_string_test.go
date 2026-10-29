package entx

import (
	"fmt"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/simpledms/simpledms/encryptor"
)

func TestEncryptedStringScanNullAndEncryptedValue(t *testing.T) {
	var unset EncryptedString
	if err := unset.Scan(nil); err != nil {
		t.Fatalf("scan NULL: %v", err)
	}
	if unset != nil {
		t.Fatalf("scan NULL should leave secret unset, got %q", unset)
	}
	unset = EncryptedString("previous-secret")
	if err := unset.Scan(nil); err != nil {
		t.Fatalf("rescan NULL: %v", err)
	}
	if unset != nil {
		t.Fatalf("scan NULL should clear an existing secret, got %q", unset)
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	previousIdentity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = identity
	t.Cleanup(func() { encryptor.NilableX25519MainIdentity = previousIdentity })

	const plaintext = "turnstile-secret"
	ciphertext, err := encryptor.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}
	var got EncryptedString
	if err := got.Scan(ciphertext); err != nil {
		t.Fatalf("scan encrypted value: %v", err)
	}
	if got.String() != plaintext {
		t.Fatalf("decrypted value: want %q, got %q", plaintext, got.String())
	}
}

func TestEncryptedStringFormattingRedactsPlaintext(t *testing.T) {
	const secret = "turnstile-secret"
	value := EncryptedString(secret)
	for _, format := range []string{"%v", "%s", "%q", "%x"} {
		if got := fmt.Sprintf(format, value); strings.Contains(got, secret) || got != "[REDACTED]" {
			t.Errorf("fmt.Sprintf(%q) = %q, want redacted output", format, got)
		}
	}
	if got := fmt.Sprintf("%v", []EncryptedString{value}); strings.Contains(got, secret) {
		t.Errorf("nested formatting leaked plaintext: %q", got)
	}
	if value.String() != secret {
		t.Fatal("String should explicitly return plaintext")
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	previousIdentity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = identity
	t.Cleanup(func() { encryptor.NilableX25519MainIdentity = previousIdentity })
	ciphertext, err := value.Value()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext.([]byte)), secret) {
		t.Fatal("database value contains plaintext")
	}
	var scanned EncryptedString
	if err := scanned.Scan(ciphertext); err != nil {
		t.Fatal(err)
	}
	if scanned.String() != secret {
		t.Fatalf("roundtrip got %q", scanned.String())
	}
}
