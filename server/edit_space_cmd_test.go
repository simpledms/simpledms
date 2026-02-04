package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/action"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	tenant2 "github.com/simpledms/simpledms/model/tenant"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/httpx"

	_ "github.com/simpledms/simpledms/db/entmain/runtime"
	_ "github.com/simpledms/simpledms/db/enttenant/runtime"
)

func TestSignUpCmdCreatesTenantAndAccount(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	accountx, tenantx := signUpAccount(t, harness, "user@example.com")

	assignmentCount := harness.mainDB.ReadWriteConn.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(tenantx.ID),
			tenantaccountassignment.AccountID(accountx.ID),
			tenantaccountassignment.RoleEQ(tenantrole.Owner),
		).
		CountX(context.Background())
	if assignmentCount != 1 {
		t.Fatalf("expected 1 tenant assignment, got %d", assignmentCount)
	}
}

func TestEditSpaceCmdUpdatesSpaceAndRootDir(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		spaceName := "Operations"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)

		updatedName := "Operations & Logistics"
		updatedDescription := "Updated description"

		form := url.Values{}
		form.Set("SpaceID", spacex.PublicID.String())
		form.Set("Name", updatedName)
		form.Set("Description", updatedDescription)

		req := httptest.NewRequest(http.MethodPost, "/-/spaces/edit-space-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		err := harness.actions.Spaces.EditSpaceCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			tenantCtx,
		)
		if err != nil {
			return fmt.Errorf("edit space command: %w", err)
		}

		if header := rr.Header().Get("HX-Trigger"); header != event.SpaceUpdated.String() {
			return fmt.Errorf("expected HX-Trigger %q, got %q", event.SpaceUpdated.String(), header)
		}

		updatedSpace := tenantCtx.TTx.Space.Query().Where(space.ID(spacex.ID)).OnlyX(tenantCtx)
		if updatedSpace.Name != updatedName {
			return fmt.Errorf("expected space name %q, got %q", updatedName, updatedSpace.Name)
		}
		if updatedSpace.Description != updatedDescription {
			return fmt.Errorf("expected space description %q, got %q", updatedDescription, updatedSpace.Description)
		}

		rootDir := tenantCtx.TTx.File.Query().Where(
			file.SpaceID(spacex.ID),
			file.IsDirectory(true),
			file.IsRootDir(true),
		).OnlyX(tenantCtx)
		if rootDir.Name != updatedName {
			return fmt.Errorf("expected root dir name %q, got %q", updatedName, rootDir.Name)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("edit space command: %v", err)
	}
}

func signUpAccount(t *testing.T, harness *actionTestHarness, email string) (*entmain.Account, *entmain.Tenant) {
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
	form.Set("SubscribeToNewsletter", "false")

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
		t.Fatalf("sign up command: %v", err)
	}

	if err := mainTx.Commit(); err != nil {
		t.Fatalf("commit main tx: %v", err)
	}

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())
	tenantx := accountx.QueryTenants().OnlyX(context.Background())

	return accountx, tenantx
}

func initTenantDB(t *testing.T, harness *actionTestHarness, tenantx *entmain.Tenant) *sqlx.TenantDB {
	t.Helper()

	migrationsTenantFS, err := migrate.NewMigrationsTenantFS()
	if err != nil {
		t.Fatalf("load tenant migrations: %v", err)
	}

	tenantm := tenant2.NewTenant(tenantx)
	tenantDB, err := tenantm.Init(true, harness.metaPath, migrationsTenantFS)
	if err != nil {
		t.Fatalf("init tenant db: %v", err)
	}
	harness.tenantDBs.Store(tenantx.ID, tenantDB)

	t.Cleanup(func() {
		_ = tenantDB.Close()
	})

	return tenantDB
}

func newTenantContext(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	tenantDB *sqlx.TenantDB,
) (*entmain.Tx, *enttenant.Tx, *ctxx.TenantContext) {
	t.Helper()

	mainTx, err := harness.mainDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}

	visitorCtx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(visitorCtx, accountx, harness.i18n, harness.tenantDBs)

	tenantTx, err := tenantDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		_ = mainTx.Rollback()
		t.Fatalf("start tenant tx: %v", err)
	}

	tenantCtx := ctxx.NewTenantContext(mainCtx, tenantTx, tenantx)

	return mainTx, tenantTx, tenantCtx
}

func createSpaceViaCmd(t *testing.T, actions *action.Actions, tenantCtx *ctxx.TenantContext, name string) {
	t.Helper()

	form := url.Values{}
	form.Set("Name", name)
	form.Set("Description", "Initial description")
	form.Set("AddMeAsSpaceOwner", "true")

	req := httptest.NewRequest(http.MethodPost, "/-/spaces/create-space-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	err := actions.Spaces.CreateSpaceCmd.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		tenantCtx,
	)
	if err != nil {
		t.Fatalf("create space command: %v", err)
	}
}
