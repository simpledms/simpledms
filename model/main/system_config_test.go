package modelmain

import (
	"testing"

	"github.com/simpledms/simpledms/db/entmain"
)

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
