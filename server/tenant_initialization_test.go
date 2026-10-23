package server

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/hook"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	tenant2 "github.com/simpledms/simpledms/model/main/tenant"
	"github.com/simpledms/simpledms/pathx"
)

func TestTenantInitCreatesUsersBeforeMarkingTenantInitialized(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "init-order@example.com")

	usersCountWhenMarkedInitialized := -1
	spaceWasReadyWhenMarkedInitialized := false
	harness.mainDB.ReadWriteConn.Tenant.Use(func(next entmain.Mutator) entmain.Mutator {
		return hook.TenantFunc(func(ctx context.Context, m *entmain.TenantMutation) (entmain.Value, error) {
			if _, isInitialized := m.InitializedAt(); isInitialized {
				usersCountWhenMarkedInitialized = countTenantUsersOfAccount(t, harness, tenantx, accountx.ID)
				spaceWasReadyWhenMarkedInitialized = initializedSpaceIsReady(
					t, harness, tenantx, accountx.ID, tenantx.Name,
				)
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
	if !spaceWasReadyWhenMarkedInitialized {
		t.Fatal("expected the initial space, root, and owner assignment before tenant initialization")
	}
}

func TestTenantInitCreatesNamedFolderSpaceWithOwner(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "space-init@example.com")
	name := "Example GmbH"
	tenantx = harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
		SetName(name).
		SaveX(context.Background())
	tenantDB := initTenantDB(t, harness, tenantx)

	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	spacex := tenantDB.ReadOnlyConn.Space.Query().OnlyX(ctx)
	if spacex.Name != name || !spacex.IsFolderMode {
		t.Fatalf("space = (%q, folder mode %v), want (%q, true)", spacex.Name, spacex.IsFolderMode, name)
	}
	root := tenantDB.ReadOnlyConn.File.Query().Where(
		file.SpaceID(spacex.ID), file.IsRootDir(true),
	).OnlyX(ctx)
	if !root.IsDirectory || root.Name != name {
		t.Fatalf("root = (%q, directory %v), want directory named %q", root.Name, root.IsDirectory, name)
	}
	owner := tenantDB.ReadOnlyConn.User.Query().Where(user.AccountID(accountx.ID)).OnlyX(ctx)
	assignment := tenantDB.ReadOnlyConn.SpaceUserAssignment.Query().Where(
		spaceuserassignment.SpaceID(spacex.ID), spaceuserassignment.UserID(owner.ID),
	).OnlyX(ctx)
	if assignment.Role != spacerole.Owner {
		t.Fatalf("space role = %v, want owner", assignment.Role)
	}
}

func TestTenantInitCanBeRetriedWhenTenantWasNotMarkedInitialized(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "init-retry@example.com")
	initialTenantDB := initTenantDB(t, harness, tenantx)
	initialCtx := privacy.DecisionContext(context.Background(), privacy.Allow)
	initialSpace := initialTenantDB.ReadOnlyConn.Space.Query().OnlyX(initialCtx)
	initialRoot := initialTenantDB.ReadOnlyConn.File.Query().Where(
		file.SpaceID(initialSpace.ID), file.IsRootDir(true),
	).OnlyX(initialCtx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
		ClearInitializedAt().
		SaveX(context.Background())

	migrationsTenantFS, err := migrate.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	retryDB, err := tenant2.NewTenant(tenantx).Init(true, harness.metaPath, migrationsTenantFS)
	if err != nil {
		t.Fatalf("retry tenant init: %v", err)
	}
	if err := retryDB.Close(); err != nil {
		t.Fatal(err)
	}

	if count := countTenantUsersOfAccount(t, harness, tenantx, accountx.ID); count != 1 {
		t.Fatalf("expected 1 user after the retry, got %d", count)
	}
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	spaceCount := initialTenantDB.ReadOnlyConn.Space.Query().CountX(ctx)
	if spaceCount != 1 {
		t.Fatalf("expected one initialized space after retry, got %d", spaceCount)
	}
	rootCount := initialTenantDB.ReadOnlyConn.File.Query().Where(file.IsRootDir(true)).CountX(ctx)
	if rootCount != 1 {
		t.Fatalf("expected one space root after retry, got %d", rootCount)
	}
	spacex := initialTenantDB.ReadOnlyConn.Space.Query().OnlyX(ctx)
	if spacex.ID != initialSpace.ID {
		t.Fatalf("space ID after retry = %d, want original ID %d", spacex.ID, initialSpace.ID)
	}
	root := initialTenantDB.ReadOnlyConn.File.Query().Where(file.SpaceID(spacex.ID), file.IsRootDir(true)).OnlyX(ctx)
	if root.ID != initialRoot.ID {
		t.Fatalf("root ID after retry = %d, want original ID %d", root.ID, initialRoot.ID)
	}
	owner := initialTenantDB.ReadOnlyConn.User.Query().Where(user.AccountID(accountx.ID)).OnlyX(ctx)
	if count := initialTenantDB.ReadOnlyConn.SpaceUserAssignment.Query().Where(
		spaceuserassignment.SpaceID(spacex.ID),
		spaceuserassignment.UserID(owner.ID),
	).CountX(ctx); count != 1 {
		t.Fatalf("expected one owner assignment after retry, got %d", count)
	}
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)
	if tenantx.InitializedAt == nil {
		t.Fatal("expected the retried tenant to be marked initialized")
	}
}

func TestTenantInitUsesOwnerNameWhenTenantNameIsBlank(t *testing.T) {
	for _, name := range []string{"", " \t "} {
		t.Run(fmt.Sprintf("name-%q", name), func(t *testing.T) {
			harness := newActionTestHarness(t)
			_, tenantx := signUpAccount(t, harness, "blank-name@example.com")
			tenantx = harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
				SetName(name).
				SaveX(context.Background())
			tenantDB := initTenantDB(t, harness, tenantx)
			ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
			spacex := tenantDB.ReadOnlyConn.Space.Query().OnlyX(ctx)
			if spacex.Name != "Test User" {
				t.Fatalf("space name = %q, want %q", spacex.Name, "Test User")
			}
		})
	}
}

func initializedSpaceIsReady(
	t testing.TB,
	harness *actionTestHarness,
	tenantx *entmain.Tenant,
	accountID int64,
	name string,
) bool {
	t.Helper()
	db := openTenantDBByPath(t, harness, tenantx)
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	spacex, err := db.ReadOnlyConn.Space.Query().Only(ctx)
	if err != nil || spacex.Name != name || !spacex.IsFolderMode {
		return false
	}
	root, err := db.ReadOnlyConn.File.Query().Where(
		file.SpaceID(spacex.ID), file.IsRootDir(true), file.IsDirectory(true),
	).Only(ctx)
	if err != nil || root.Name != name {
		return false
	}
	owner, err := db.ReadOnlyConn.User.Query().Where(user.AccountID(accountID)).Only(ctx)
	if err != nil {
		return false
	}
	assignment, err := db.ReadOnlyConn.SpaceUserAssignment.Query().Where(
		spaceuserassignment.SpaceID(spacex.ID), spaceuserassignment.UserID(owner.ID),
	).Only(ctx)
	return err == nil && assignment.Role == spacerole.Owner
}

func openTenantDBByPath(t testing.TB, harness *actionTestHarness, tenantx *entmain.Tenant) *sqlx.TenantDB {
	t.Helper()
	tenantDBDir, err := pathx.TenantDBPath(harness.metaPath, tenantx.PublicID.String())
	if err != nil {
		t.Fatal(err)
	}
	tenantDBPath := filepath.Join(tenantDBDir, "tenant.sqlite3")
	return sqlx.NewTenantDB(
		fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadOnly),
		fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadWrite),
	)
}

func countTenantUsersOfAccount(
	t testing.TB,
	harness *actionTestHarness,
	tenantx *entmain.Tenant,
	accountID int64,
) int {
	t.Helper()

	// opened by path because the tenant may not be marked initialized yet
	tenantDB := openTenantDBByPath(t, harness, tenantx)
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
