package server

import (
	"context"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/plan"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
)

func TestMainMenuShowsOnlySetupEntriesWhenPasskeyEnrollmentRequired(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "tenant-passkey-main-menu@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		OnlyX(context.Background())

	now := time.Now()
	tenantRequired := harness.mainDB.ReadWriteConn.Tenant.Create().
		SetName("Required Tenant").
		SetCountry(country.Unknown).
		SetPlan(plan.Unknown).
		SetTermsOfServiceAccepted(now).
		SetPrivacyPolicyAccepted(now).
		SetPasskeyAuthEnforced(true).
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantRequired.ID).
		SetAccountID(accountx.ID).
		SetRole(tenantrole.Owner).
		SetIsDefault(true).
		SaveX(context.Background())

	mainTx, err := harness.mainDB.ReadOnlyConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}
	defer func() {
		_ = mainTx.Rollback()
	}()

	accountInTx := mainTx.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())

	visitorCtx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(visitorCtx, accountInTx, harness.i18n, harness.mainDB, harness.tenantDBs, true)

	mainMenu := partial2.NewMainMenu(mainCtx, harness.infra)
	menu, ok := mainMenu.Children.(*wx.Menu)
	if !ok {
		t.Fatalf("expected main menu children to be *wx.Menu, got %T", mainMenu.Children)
	}

	var labels []string
	for _, item := range menu.Items {
		if item.IsDivider || item.Label == nil {
			continue
		}
		labels = append(labels, item.Label.String(mainCtx))
	}

	if len(labels) != 2 {
		t.Fatalf("expected exactly 2 setup menu entries, got %d (%v)", len(labels), labels)
	}
	if labels[0] != "Dashboard" {
		t.Fatalf("expected first menu entry to be Dashboard, got %q", labels[0])
	}
	if labels[1] != "Sign out" {
		t.Fatalf("expected second menu entry to be Sign out, got %q", labels[1])
	}
}
