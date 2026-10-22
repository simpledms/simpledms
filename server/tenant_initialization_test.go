package server

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/hook"
	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/sqlx"
	tenant2 "github.com/simpledms/simpledms/model/main/tenant"
	"github.com/simpledms/simpledms/pathx"
)

func TestTenantInitCreatesUsersBeforeMarkingTenantInitialized(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "init-order@example.com")

	usersCountWhenMarkedInitialized := -1
	harness.mainDB.ReadWriteConn.Tenant.Use(func(next entmain.Mutator) entmain.Mutator {
		return hook.TenantFunc(func(ctx context.Context, m *entmain.TenantMutation) (entmain.Value, error) {
			if _, isInitialized := m.InitializedAt(); isInitialized {
				usersCountWhenMarkedInitialized = countTenantUsersOfAccount(t, harness, tenantx, accountx.ID)
			}
			return next.Mutate(ctx, m)
		})
	})

	initTenantDB(t, harness, tenantx)

	if usersCountWhenMarkedInitialized != 1 {
		t.Fatalf(
			"expected the user to exist when the tenant is marked initialized, got %d users",
			usersCountWhenMarkedInitialized,
		)
	}
}

func TestTenantInitCanBeRetriedWhenTenantWasNotMarkedInitialized(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "init-retry@example.com")
	initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
		ClearInitializedAt().
		SaveX(context.Background())

	migrationsTenantFS, err := migrate.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	tenantDB, err := tenant2.NewTenant(tenantx).Init(true, harness.metaPath, migrationsTenantFS)
	if err != nil {
		t.Fatalf("retry tenant init: %v", err)
	}
	if err := tenantDB.Close(); err != nil {
		t.Fatal(err)
	}

	if count := countTenantUsersOfAccount(t, harness, tenantx, accountx.ID); count != 1 {
		t.Fatalf("expected 1 user after the retry, got %d", count)
	}
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)
	if tenantx.InitializedAt == nil {
		t.Fatal("expected the retried tenant to be marked initialized")
	}
}

func countTenantUsersOfAccount(
	t testing.TB,
	harness *actionTestHarness,
	tenantx *entmain.Tenant,
	accountID int64,
) int {
	t.Helper()

	// opened by path because the tenant may not be marked initialized yet
	tenantDBDir, err := pathx.TenantDBPath(harness.metaPath, tenantx.PublicID.String())
	if err != nil {
		t.Fatal(err)
	}
	tenantDBPath := filepath.Join(tenantDBDir, "tenant.sqlite3")
	tenantDB := sqlx.NewTenantDB(
		fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadOnly),
		fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadWrite),
	)
	defer func() {
		if err := tenantDB.Close(); err != nil {
			t.Error(err)
		}
	}()

	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	count, err := tenantDB.ReadOnlyConn.User.Query().Where(user.AccountID(accountID)).Count(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return count
}
