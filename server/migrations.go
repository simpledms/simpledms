package server

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"strings"

	"entgo.io/ent/dialect"
	securejoin "github.com/cyphar/filepath-securejoin"
	migratex "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/common/tenantdbs"
	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/sqlx"
	tenant2 "github.com/simpledms/simpledms/model/tenant"
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

		// if auto migration fails because of foreign key constraint violation,
		// just create migration scripts and execute them manually
		//
		// see longer comment in model.Tenant.ExecuteDBMigrations

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
		defer func() {
			if err = mainDrv.Close(); err != nil {
				log.Println(err)
			}
		}()

		readWriteDataSourceURL := mainDB.ReadWriteDataSourceURL()
		// necessary because `PRAGMA foreign_keys = off` doesn't work in many
		// circumstances
		readWriteDataSourceURL = strings.Replace(
			readWriteDataSourceURL,
			"_foreign_keys=1",
			"_foreign_keys=0",
			1,
		)

		mainMigx, err := migratex.NewWithSourceInstance(
			"migrationsMainFS",
			mainDrv,
			dialect.SQLite+"://"+readWriteDataSourceURL,
		)
		if err != nil {
			log.Fatalf("failed loading migration instance: %v", err)
		}

		err = mainMigx.Up()
		if err != nil && !errors.Is(err, migratex.ErrNoChange) {
			log.Fatalf("failed running migrations up: %v", err)
		}

		srcErr, dbErr := mainMigx.Close()
		if srcErr != nil || dbErr != nil {
			log.Fatalf("failed closing migration instance: %v, %v", srcErr, dbErr)
		}
	}

	return mainDB
}

func dbMigrationsTenantDBs(mainDB *sqlx.MainDB, isDevMode bool, metaPath string) *tenantdbs.TenantDBs {
	// TODO Where query shouldn't be necessary because of mixin, but it seems it is...
	tenants := mainDB.ReadWriteConn.Tenant.Query().Where(tenant.DeletedAtIsNil()).AllX(context.Background())
	tenantDBs := tenantdbs.NewTenantDBs()

	for _, tenant := range tenants {
		tenantm := tenant2.NewTenant(tenant)
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
