package scheduler

import (
	"context"
	"log"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
)

func (qq *Scheduler) runTenantDataMigrationsOnce(ctx context.Context, runner *tenantdatamigration.Runner) {
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		tenantx, err := qq.mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.ID(tenantID)).Only(ctx)
		if err != nil {
			if entmain.IsNotFound(err) {
				log.Println("tenant not found", tenantID)
				return true
			}
			log.Println(err)
			return true
		}

		migrations := []tenantdatamigration.Migration{
			filesystem.NewZIPMIMERedetectionMigration(
				qq.infra.FileSystem(),
				tenantx.X25519IdentityEncrypted.Identity(),
			),
		}
		for _, migration := range migrations {
			completed, err := runner.Run(ctx, tenantDB, migration)
			if err != nil {
				log.Printf("tenant %d data migration %q failed: %v", tenantID, migration.Key(), err)
				continue
			}
			if completed {
				log.Printf("tenant %d data migration %q completed", tenantID, migration.Key())
			}
		}
		return true
	})
}
