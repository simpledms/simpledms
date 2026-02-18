package server

import (
	"context"
	"errors"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/pluginx"
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

func TestSignUpFlowRollsBackWhenPluginHookFails(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	plugin := &captureSignUpPlugin{err: errors.New("plugin sign-up hook failed")}
	harness.infra.PluginRegistry().SetPlugins(plugin)
	t.Cleanup(func() {
		harness.infra.PluginRegistry().SetPlugins()
	})

	err := executeSignUpFlow(t, harness, "signup-hook-failure@example.com")
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

func TestSignUpFlowEmitsPluginHookEvent(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	plugin := &captureSignUpPlugin{}
	harness.infra.PluginRegistry().SetPlugins(plugin)
	t.Cleanup(func() {
		harness.infra.PluginRegistry().SetPlugins()
	})

	err := executeSignUpFlow(t, harness, "signup-hook-success@example.com")
	if err != nil {
		t.Fatalf("signup flow: %v", err)
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

func executeSignUpFlow(t *testing.T, harness *actionTestHarness, email string) error {
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

	accountm, err := modelmain.NewSignUpService().SignUp(
		ctx,
		email,
		"Test User",
		"Test",
		"User",
		country.Switzerland,
		language.English,
		true,
		true,
	)
	if err != nil {
		_ = mainTx.Rollback()
		return err
	}

	tenantx, err := accountm.Data.QueryTenants().Only(ctx)
	if err != nil {
		_ = mainTx.Rollback()
		return err
	}

	err = harness.infra.PluginRegistry().EmitSignUp(ctx, pluginx.SignUpEvent{
		AccountID:             accountm.Data.ID,
		AccountPublicID:       accountm.Data.PublicID.String(),
		AccountEmail:          accountm.Data.Email.String(),
		TenantID:              tenantx.ID,
		TenantPublicID:        tenantx.PublicID.String(),
		TenantName:            tenantx.Name,
		SubscribeToNewsletter: true,
	})
	if err != nil {
		_ = mainTx.Rollback()
		return err
	}

	if err := mainTx.Commit(); err != nil {
		return err
	}

	return nil
}
