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

		/*if err := normalizeMainDBBeforeDevSchemaCreate(mainDB); err != nil {
			log.Fatalf("failed normalizing main db before schema migration: %v", err)
		}*/

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

/*
func normalizeMainDBBeforeDevSchemaCreate(mainDB *sqlx.MainDB) error {
	ctx := context.Background()

	hasAccountsTable, err := sqliteTableExists(ctx, mainDB.ReadWriteConn, "accounts")
	if err != nil {
		return err
	}
	if !hasAccountsTable {
		return nil
	}

	normalizations := []struct {
		column string
		addSQL string
		query  string
	}{
		{
			column: "passkey_login_enabled",
			addSQL: "ALTER TABLE accounts ADD COLUMN passkey_login_enabled bool NOT NULL DEFAULT (false)",
			query:  "UPDATE accounts SET passkey_login_enabled = 0 WHERE passkey_login_enabled IS NULL",
		},
		{
			column: "passkey_recovery_code_salt",
			addSQL: "ALTER TABLE accounts ADD COLUMN passkey_recovery_code_salt text NOT NULL DEFAULT ('')",
			query:  "UPDATE accounts SET passkey_recovery_code_salt = '' WHERE passkey_recovery_code_salt IS NULL",
		},
		{
			column: "passkey_recovery_code_hashes",
			addSQL: "ALTER TABLE accounts ADD COLUMN passkey_recovery_code_hashes json NOT NULL DEFAULT ('[]')",
			query:  "UPDATE accounts SET passkey_recovery_code_hashes = '[]' WHERE passkey_recovery_code_hashes IS NULL OR TRIM(passkey_recovery_code_hashes) = '' OR passkey_recovery_code_hashes = 'null'",
		},
	}

	for _, normalization := range normalizations {
		hasColumn, err := sqliteTableHasColumn(ctx, mainDB.ReadWriteConn, "accounts", normalization.column)
		if err != nil {
			return err
		}
		if !hasColumn {
			_, err = mainDB.ReadWriteConn.ExecContext(ctx, normalization.addSQL)
			if err != nil {
				return err
			}
		}

		_, err = mainDB.ReadWriteConn.ExecContext(ctx, normalization.query)
		if err != nil {
			return err
		}
	}

	return nil
}

func sqliteTableExists(
	ctx context.Context,
	queryer interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	},
	tableName string,
) (bool, error) {
	rows, err := queryer.QueryContext(
		ctx,
		"SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1",
		tableName,
	)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return rows.Next(), rows.Err()
}

func sqliteTableHasColumn(
	ctx context.Context,
	queryer interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	},
	tableName string,
	columnName string,
) (bool, error) {
	rows, err := queryer.QueryContext(
		ctx,
		"SELECT 1 FROM pragma_table_info('"+tableName+"') WHERE name = ? LIMIT 1",
		columnName,
	)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return rows.Next(), rows.Err()
}
*/

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
