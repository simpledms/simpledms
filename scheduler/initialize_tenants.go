package scheduler

import (
	"context"
	"io/fs"
	"log"
	"runtime/debug"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/entmain/tenant"
	"github.com/simpledms/simpledms/model/modelmain"
)

func (qq *Scheduler) initializeTenants(devMode bool, metaPath string, migrationsTenantFS fs.FS) {
	defer func() {
		// tested and works
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")

			// TODO what is a good interval
			time.Sleep(1 * time.Minute)

			// tested and works, automatically restarts loop
			qq.initializeTenants(devMode, metaPath, migrationsTenantFS)
		}
	}()
	for {
		// TODO in transaction or not? if so, don't forget rollback in recovery logic

		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		tenants := qq.mainDB.ReadWriteConn.Tenant.
			Query().
			Where(tenant.InitializedAtIsNil()).
			Order(tenant.ByCreatedAt(sql.OrderDesc())).
			AllX(ctx)

		for _, tenantx := range tenants {
			tenantm := modelmain.NewTenant(tenantx)

			// TODO implement more robust error handling with pause between retries
			tenantDB, err := tenantm.Init(devMode, metaPath, migrationsTenantFS)
			if err != nil {
				log.Println(err)
				continue
			}

			qq.tenantDBs.Store(tenantm.Data.ID, tenantDB)
		}

		// TODO what is a good interval? user shouldn't have to wait before he/she can login...
		time.Sleep(15 * time.Second)
	}
}
