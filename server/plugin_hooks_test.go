package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/pluginx"
	"github.com/simpledms/simpledms/util/httpx"
)

type captureSignUpPlugin struct {
	err   error
	calls int
	last  pluginx.SignUpEvent
}

func (qq *captureSignUpPlugin) Name() string {
	return "capture-sign-up"
}

func (qq *captureSignUpPlugin) OnSignUp(_ ctxx.Context, event pluginx.SignUpEvent) error {
	qq.calls++
	qq.last = event
	return qq.err
}

func TestSignUpCmdRollsBackWhenPluginHookFails(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	plugin := &captureSignUpPlugin{err: errors.New("plugin sign-up hook failed")}
	harness.infra.PluginRegistry().SetPlugins(plugin)
	t.Cleanup(func() {
		harness.infra.PluginRegistry().SetPlugins()
	})

	err := executeSignUpCmd(t, harness, "signup-hook-failure@example.com")
	if err == nil {
		t.Fatal("expected signup error")
	}

	if plugin.calls != 1 {
		t.Fatalf("expected plugin hook call count 1, got %d", plugin.calls)
	}

	accountCount := harness.mainDB.ReadOnlyConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText("signup-hook-failure@example.com"))).
		CountX(context.Background())
	if accountCount != 0 {
		t.Fatalf("expected no account persisted, got %d", accountCount)
	}

	tenantCount := harness.mainDB.ReadOnlyConn.Tenant.Query().CountX(context.Background())
	if tenantCount != 0 {
		t.Fatalf("expected no tenant persisted, got %d", tenantCount)
	}
}

func TestSignUpCmdEmitsPluginHookEvent(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	plugin := &captureSignUpPlugin{}
	harness.infra.PluginRegistry().SetPlugins(plugin)
	t.Cleanup(func() {
		harness.infra.PluginRegistry().SetPlugins()
	})

	err := executeSignUpCmd(t, harness, "signup-hook-success@example.com")
	if err != nil {
		t.Fatalf("signup command: %v", err)
	}

	if plugin.calls != 1 {
		t.Fatalf("expected plugin hook call count 1, got %d", plugin.calls)
	}
	if plugin.last.AccountEmail != "signup-hook-success@example.com" {
		t.Fatalf("expected event email signup-hook-success@example.com, got %q", plugin.last.AccountEmail)
	}
	if plugin.last.TenantName != "Test User" {
		t.Fatalf("expected tenant name Test User, got %q", plugin.last.TenantName)
	}
}

func executeSignUpCmd(t *testing.T, harness *actionTestHarness, email string) error {
	t.Helper()

	mainTx, err := harness.mainDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}

	ctx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)

	form := url.Values{}
	form.Set("Email", email)
	form.Set("FirstName", "Test")
	form.Set("LastName", "User")
	form.Set("Country", country.Switzerland.String())
	form.Set("Language", language.English.String())
	form.Set("SubscribeToNewsletter", "true")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-up-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	err = harness.actions.Auth.SignUpCmd.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		ctx,
	)
	if err != nil {
		_ = mainTx.Rollback()
		return err
	}

	if err := mainTx.Commit(); err != nil {
		return err
	}

	return nil
}
