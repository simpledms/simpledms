package webdavcredential

import (
	"strings"
	"testing"
)

func TestRandomSecretLength(t *testing.T) {
	for _, compatibilityMode := range []bool{false, true} {
		for _, length := range []int{0, 12, 13, 20, 43} {
			secret, err := randomSecret(length, compatibilityMode)
			if err != nil {
				t.Fatal(err)
			}
			expectedLength := length
			if expectedLength == 0 {
				expectedLength = DefaultSecretLength
			}
			if len(secret) != expectedLength {
				t.Fatalf("expected %d-character secret, got %d", expectedLength, len(secret))
			}
			if compatibilityMode {
				for _, char := range secret {
					if !strings.ContainsRune(
						"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_",
						char,
					) {
						t.Fatalf("expected compatibility secret, got %q", secret)
					}
				}
				continue
			}
			hasSpecialCharacter := false
			for index := range len(secret) {
				char := secret[index]
				if char < '!' || char > '~' {
					t.Fatalf("expected printable ASCII secret, got %q", secret)
				}
				hasSpecialCharacter = hasSpecialCharacter || !isASCIIAlphaNumeric(char)
			}
			if !hasSpecialCharacter {
				t.Fatalf("expected a special character, got %q", secret)
			}
		}
	}

	for _, length := range []int{1, MinimumSecretLength - 1, DefaultSecretLength + 1} {
		if _, err := randomSecret(length, false); err == nil {
			t.Fatalf("expected length %d to fail", length)
		}
	}
}
