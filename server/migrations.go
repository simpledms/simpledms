package server

import (
	"context"
	"errors"
	"io/fs"
	"log"

	"entgo.io/ent/dialect"
	securejoin "github.com/cyphar/filepath-securejoin"
	migratex "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/common/tenantdbs"
	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/modelmain"
)

// caller has to close db
func dbMigrationsMainDB(isDevMode bool, metaPath string, migrationsMainFS fs.FS) *sqlx.MainDB {
	mainDBPath, err := securejoin.SecureJoin(metaPath, "main.sqlite3")
	if err != nil {
		log.Fatalln(err)
	}
	mainDB := sqlx.NewMainDB(mainDBPath)

	if isDevMode {
		mainDB.Debug()
		log.Println("Running in development mode")

		if err := mainDB.ReadWriteConn.Schema.Create(
			context.Background(),
			migratemain.WithDropIndex(true),
			migratemain.WithDropColumn(true),
		); err != nil {
			log.Fatalf("failed creating schema resources: %v", err)
		}

	} else {
		mainDrv, err := iofs.New(migrationsMainFS, ".")
		if err != nil {
			log.Fatalf("failed reading migration filesystem: %v", err)
		}
		mainMigx, err := migratex.NewWithSourceInstance("migrationsMainFS", mainDrv, dialect.SQLite+"://"+mainDB.ReadWriteDataSourceURL())
		if err != nil {
			log.Fatalf("failed loading migration instance: %v", err)
		}
		err = mainMigx.Up()
		if err != nil && !errors.Is(err, migratex.ErrNoChange) {
			log.Fatalf("failed running migrations up: %v", err)
		}
	}

	return mainDB
}

func dbMigrationsTenantDBs(mainDB *sqlx.MainDB, isDevMode bool, metaPath string) *tenantdbs.TenantDBs {
	// TODO Where query shouldn't be necessary because of mixin, but it seems it is...
	tenants := mainDB.ReadWriteConn.Tenant.Query().Where(tenant.DeletedAtIsNil()).AllX(context.Background())
	tenantDBs := tenantdbs.NewTenantDBs()

	for _, tenant := range tenants {
		tenantm := modelmain.NewTenant(tenant)
		tenantClient, err := tenantm.OpenDB(isDevMode, metaPath)
		if err != nil {
			log.Println(err)
			// TODO continue or fail?
			continue
		}
		tenantDBs.Store(tenant.ID, tenantClient)
	}

	return tenantDBs
}
