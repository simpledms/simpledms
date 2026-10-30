package scheduler

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	"filippo.io/age"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
)

// A pass initializes every pending tenant and reports the count, so that the scheduler can
// look for tenants registered during the pass right away instead of pausing first. An idle
// pass reports zero, which is when the scheduler pauses.
func TestInitializePendingTenantsReportsInitializedCountAndLeavesNothingPending(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	mainDB := newTestMainDB(t)
	now := time.Now()
	createTestTenant(t, mainDB, 1, now)
	createTestTenant(t, mainDB, 2, now)
	mainDB.ReadWriteConn.Tenant.UpdateOneID(2).SetInitializedAt(now).SaveX(ctx)
	createTestTenant(t, mainDB, 3, now)
	scheduler := newTenantInitializationScheduler(t, mainDB)
	metaPath := t.TempDir()
	migrationsTenantFS, err := migrate.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}

	initialized := scheduler.initializePendingTenants(ctx, true, metaPath, migrationsTenantFS)

	if initialized != 2 {
		t.Fatalf("initialized tenants = %d, want 2", initialized)
	}
	pending := mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.InitializedAtIsNil()).CountX(ctx)
	if pending != 0 {
		t.Fatalf("pending tenants after pass = %d, want 0", pending)
	}
	for _, tenantID := range []int64{1, 3} {
		if _, isOpen := scheduler.tenantDBs.Load(tenantID); !isOpen {
			t.Fatalf("tenant %d DB is not registered after initialization", tenantID)
		}
	}

	if initialized := scheduler.initializePendingTenants(ctx, true, metaPath, migrationsTenantFS); initialized != 0 {
		t.Fatalf("initialized tenants in idle pass = %d, want 0", initialized)
	}
}

func newTenantInitializationScheduler(t *testing.T, mainDB *sqlx.MainDB) *Scheduler {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	previousIdentity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = identity
	t.Cleanup(func() { encryptor.NilableX25519MainIdentity = previousIdentity })

	tenantDBs := tenantdbs.NewTenantDBs()
	t.Cleanup(func() {
		tenantDBs.Range(func(_ int64, tenantDB *sqlx.TenantDB) bool {
			if err := tenantDB.Close(); err != nil {
				t.Errorf("close tenant db: %v", err)
			}
			return true
		})
	})

	return &Scheduler{
		mainDB:    mainDB,
		tenantDBs: tenantDBs,
		// The data migrations of an empty tenant never open a file.
		infra: common.NewInfra(nil, t.TempDir(), nil, nil, nil, nil, nil),
	}
}
