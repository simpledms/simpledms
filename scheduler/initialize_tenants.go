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

func (qq *Scheduler) initializeTenants(devMode bool, metaPath string, migrationsTenantFS fs.FS) {
	for {
		// TODO in transaction or not? if so, don't forget rollback in recovery logic

		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		tenants := qq.mainDB.ReadOnlyConn.Tenant.
			Query().
			Where(tenant.InitializedAtIsNil()).
			Order(tenant.ByCreatedAt(sql.OrderDesc())).
			Limit(defaultSchedulerBatchSize).
			AllX(ctx)

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

			// A request may have opened the freshly initialized DB in the meantime.
			tenantDB = qq.tenantDBs.StoreOrClose(tenantm.Data.ID, tenantDB)
			qq.runTenantDataMigrations(ctx, tenantdatamigration.NewRunner(), tenantDB, tenantm.Data)
		}

		// TODO what is a good interval? user shouldn't have to wait before he/she can login...
		time.Sleep(15 * time.Second)
	}
}
