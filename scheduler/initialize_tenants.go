package scheduler

import (
	"context"
	"io/fs"
	"log"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	tenant2 "github.com/simpledms/simpledms/model/main/tenant"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
)

// initializeTenantsIdleInterval is the pause between passes when there are no tenants to
// initialize or none could be initialized. It also spaces out the retries of failing tenants.
const initializeTenantsIdleInterval = 15 * time.Second

// initializeTenants initializes registered tenants in the background. A pass that initialized
// tenants is followed by the next one right away, because tenants registered during the pass
// would otherwise wait the idle interval on top of the pass; the owner of a new organization
// should not have to wait longer than necessary before signing in.
func (qq *Scheduler) initializeTenants(devMode bool, metaPath string, migrationsTenantFS fs.FS) {
	for {
		// TODO in transaction or not? if so, don't forget rollback in recovery logic
		ctx := privacy.DecisionContext(context.Background(), privacy.Allow)

		if qq.initializePendingTenants(ctx, devMode, metaPath, migrationsTenantFS) == 0 {
			time.Sleep(initializeTenantsIdleInterval)
		}
	}
}

// initializePendingTenants initializes one batch of tenants that are not initialized yet and
// returns how many it initialized. Tenants whose initialization fails stay pending and are
// retried in a later pass.
func (qq *Scheduler) initializePendingTenants(
	ctx context.Context,
	devMode bool,
	metaPath string,
	migrationsTenantFS fs.FS,
) int {
	tenants := qq.mainDB.ReadOnlyConn.Tenant.
		Query().
		Where(tenant.InitializedAtIsNil()).
		Order(tenant.ByCreatedAt(sql.OrderDesc())).
		Limit(defaultSchedulerBatchSize).
		AllX(ctx)

	initialized := 0
	for _, tenantCandidate := range tenants {
		tenantx, err := qq.mainDB.ReadWriteConn.Tenant.Get(ctx, tenantCandidate.ID)
		if err != nil {
			if entmain.IsNotFound(err) {
				continue
			}
			log.Println(err)
			continue
		}

		tenantm := tenant2.NewTenant(tenantx)

		// TODO implement more robust error handling with pause between retries
		tenantDB, err := tenantm.Init(devMode, metaPath, migrationsTenantFS)
		if err != nil {
			log.Println(err)
			continue
		}
		initialized++

		// A request may have opened the freshly initialized DB in the meantime.
		tenantDB = qq.tenantDBs.StoreOrClose(tenantm.Data.ID, tenantDB)
		qq.runTenantDataMigrations(ctx, tenantdatamigration.NewRunner(), tenantDB, tenantm.Data)
	}

	return initialized
}
