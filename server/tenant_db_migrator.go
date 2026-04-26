package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"entgo.io/ent/dialect"
	migratex "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/sqlx"
)

type tenantDBMigrator struct {
	devMode            bool
	migrationsTenantFS fs.FS
}

func newTenantDBMigrator(devMode bool, migrationsTenantFS fs.FS) *tenantDBMigrator {
	return &tenantDBMigrator{
		devMode:            devMode,
		migrationsTenantFS: migrationsTenantFS,
	}
}

func (qq *tenantDBMigrator) execute(tenantDB *sqlx.TenantDB) error {
	ctx := context.Background()

	// necessary because they depend on other tables and are not managed by ent;
	// for example on updates to files table, migration failes if file_infos still exists because
	// data gets transferred in temporary table and files table then gets deleted by ent...
	//
	// FIXME find a more efficient way, could be very expensive doing this on every startup; only
	// 		necessary if migration is necessary
	for _, query := range []string{
		"drop view if exists file_infos;",
		"drop view if exists resolved_tag_assignments;",
		"DROP TABLE IF EXISTS file_searches;",
		"drop trigger if exists files_ai;",
		"drop trigger if exists files_ad;",
		"drop trigger if exists files_au;",
	} {
		_, err := tenantDB.ReadWriteConn.ExecContext(ctx, query)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	if qq.devMode {
		log.Println("Running in development mode")

		// TODO
		// fails for tables that reference themselves (O2M with same type), for example files
		// (parent_id) because in the migration, a copy of the table files is created, but
		// parent_id still references the `files` table instead of the copied table;
		// ent disables foreign keys, but it seems it has no effect
		//
		// hint from docs that might help fixing:
		// It is not possible to enable or disable foreign key constraints in the middle of a
		// multi-statement transaction (when SQLite is not in autocommit mode).
		// Attempting to do so does not return an error; it simply has no effect.
		// https://www.sqlite.org/foreignkeys.html
		//
		// if auto migration fails because of foreign key constraint violation,
		// just create migration scripts and execute them manually
		if err := tenantDB.ReadWriteConn.Schema.Create(
			ctx,
			migrate.WithDropIndex(true),
			migrate.WithDropColumn(true),
		); err != nil {
			// fatal only in dev mode
			log.Fatalf("failed creating schema resources: %v", err)
			return err
		}
	} else {
		err := qq.migrateTenant(tenantDB)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	// TODO find a better way
	for _, query := range []string{
		schema.ResolvedTagAssignment{}.SQL(),
		schema.FileSearch{}.SQL(),
	} {
		_, err := tenantDB.ReadWriteConn.ExecContext(ctx, query)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func (qq *tenantDBMigrator) migrateTenant(tenantDB *sqlx.TenantDB) error {
	drv, err := iofs.New(qq.migrationsTenantFS, ".")
	if err != nil {
		log.Printf("failed reading migration filesystem: %v", err)
		return err
	}
	defer func() {
		if err = drv.Close(); err != nil {
			log.Println(err)
		}
	}()

	readWriteTenantDataSourceURL := tenantDB.ReadWriteDataSourceURL()
	// necessary because `PRAGMA foreign_keys = off` doesn't work in many
	// circumstances
	readWriteTenantDataSourceURL = strings.Replace(
		readWriteTenantDataSourceURL,
		"_foreign_keys=1",
		"_foreign_keys=0",
		1,
	)

	// sqlite is pure go driver; sqlite3 is with cgo
	// TODO can it lead to conflicts if db is opened a second time?
	// TODO disable foreign keys?
	migx, err := migratex.NewWithSourceInstance(
		"migrationsTenantFS",
		drv,
		dialect.SQLite+"://"+readWriteTenantDataSourceURL,
	)
	if err != nil {
		log.Printf("failed loading migration instance: %v", err)
		return err
	}
	err = migx.Up()
	if err != nil && !errors.Is(err, migratex.ErrNoChange) {
		log.Printf("failed running migrations up: %v", err)
		return err
	}

	srcErr, dbErr := migx.Close()
	if srcErr != nil || dbErr != nil {
		log.Println(srcErr, dbErr)
		return fmt.Errorf("failed closing migration instance: %v, %v", srcErr, dbErr)
	}

	return nil
}
