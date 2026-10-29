package systemstatus

import (
	"strings"
	"testing"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entx"
	appmodel "github.com/simpledms/simpledms/model/main/app"
	systemconfigmodel "github.com/simpledms/simpledms/model/main/systemconfig"
)

func TestTurnstileStatusStoredConfigurationIsEffectiveAndRedacted(t *testing.T) {
	t.Setenv("SIMPLEDMS_TURNSTILE_SITE_KEY", "environment-site-key")
	t.Setenv("SIMPLEDMS_TURNSTILE_SECRET", "environment-secret")
	checker := turnstileChecker(true, false, "site-key-stored", "secret-stored", "https://signup.example.test")
	status := checker.turnstileStatus()

	if status.Level() != StatusLevelOK {
		t.Fatalf("expected OK, got %v", status.Level())
	}
	assertTurnstileSetting(t, status, "Mode", "Stored configuration")
	assertTurnstileSetting(t, status, "Signup bot protection", "Enabled")
	assertTurnstileSetting(t, status, "Site key", "site-key-stored")
	assertTurnstileSetting(t, status, "Secret key", "Set")
	assertTurnstileSetting(t, status, "Expected hostname", "signup.example.test")
	assertTurnstileNoSecret(t, status, "secret-stored", "environment-secret")
	if hasTurnstileFinding(status, "validated during signup") == false {
		t.Fatal("expected signup-time validation finding")
	}
}

func TestTurnstileStatusDevelopmentUsesPublicTestKeysWithoutClaimingProtection(t *testing.T) {
	checker := turnstileChecker(true, true, "production-site-key", "production-secret", "https://signup.example.test")
	status := checker.turnstileStatus()

	if status.Level() != StatusLevelWarning {
		t.Fatalf("expected warning, got %v", status.Level())
	}
	assertTurnstileSetting(t, status, "Mode", "Development test keys")
	assertTurnstileSetting(t, status, "Signup bot protection", "Enabled")
	assertTurnstileSetting(t, status, "Site key", appmodel.TurnstileTestSiteKey)
	assertTurnstileSetting(t, status, "Secret key", "Set")
	assertTurnstileNoSecret(t, status, "production-secret")
	if !hasTurnstileFinding(status, "not real bot protection") {
		t.Fatal("expected warning that development test keys are not actual protection")
	}
}

func TestTurnstileStatusMissingAndPartialConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		site    string
		secret  string
		level   StatusLevel
		setting string
	}{
		{name: "both absent", level: StatusLevelDisabled, setting: "Disabled"},
		{name: "site only", site: "site", level: StatusLevelWarning, setting: "Disabled"},
		{name: "secret only", secret: "secret", level: StatusLevelWarning, setting: "Disabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := turnstileChecker(true, false, tc.site, tc.secret, "https://signup.example.test").turnstileStatus()
			if status.Level() != tc.level {
				t.Fatalf("expected %v, got %v", tc.level, status.Level())
			}
			assertTurnstileSetting(t, status, "Signup bot protection", tc.setting)
		})
	}
}

func TestTurnstileStatusRejectsProductionTestKeysAndLocalHostname(t *testing.T) {
	for _, tc := range []struct {
		name   string
		site   string
		secret string
		origin string
	}{
		{name: "test site key", site: "1x-test-site", secret: "real-secret", origin: "https://signup.example.test"},
		{name: "test secret", site: "real-site", secret: "2x-test-secret", origin: "https://signup.example.test"},
		{name: "missing origin", site: "real-site", secret: "real-secret"},
		{name: "local origin", site: "real-site", secret: "real-secret", origin: "http://localhost"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := turnstileChecker(true, false, tc.site, tc.secret, tc.origin).turnstileStatus()
			if status.Level() != StatusLevelError {
				t.Fatalf("expected error, got %v", status.Level())
			}
		})
	}
}

func TestTurnstileStatusDisabledOutsideSaaS(t *testing.T) {
	status := turnstileChecker(false, false, "site", "secret", "https://signup.example.test").turnstileStatus()
	if status.Level() != StatusLevelDisabled {
		t.Fatalf("expected disabled, got %v", status.Level())
	}
	assertTurnstileSetting(t, status, "Signup bot protection", "Disabled")
}

func turnstileChecker(isSaaS, isDev bool, site, secret, origin string) *SystemStatusChecker {
	config := systemconfigmodel.NewSystemConfig(
		&entmain.SystemConfig{
			TurnstileSiteKey: site,
			TurnstileSecret:  entx.NewEncryptedString(secret),
		},
		isSaaS,
		false,
		false,
		origin,
		"example.test",
		"",
	)
	return NewSystemStatusChecker(config, nil, isDev, false, false, nil)
}

func assertTurnstileSetting(t *testing.T, status *ComponentStatus, label, want string) {
	t.Helper()
	for _, setting := range status.Settings() {
		if setting.Label().StringUntranslated() == label {
			if got := setting.Value().StringUntranslated(); got != want {
				t.Fatalf("%s: want %q, got %q", label, want, got)
			}
			return
		}
	}
	t.Fatalf("setting %q not found", label)
}

func assertTurnstileNoSecret(t *testing.T, status *ComponentStatus, secrets ...string) {
	t.Helper()
	for _, setting := range status.Settings() {
		value := setting.Value().StringUntranslated()
		for _, secret := range secrets {
			if secret != "" && strings.Contains(value, secret) {
				t.Fatalf("secret leaked in %q: %q", setting.Label().StringUntranslated(), value)
			}
		}
	}
}

func hasTurnstileFinding(status *ComponentStatus, substring string) bool {
	for _, finding := range status.Findings() {
		if strings.Contains(finding.Message().StringUntranslated(), substring) {
			return true
		}
	}
	return false
}
