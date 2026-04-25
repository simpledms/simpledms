package server

import (
	"context"
	"io/fs"
	"log"

	"github.com/marcobeierer/go-core/db/entmain/tenant"

	"github.com/marcobeierer/go-core/db/sqlx"
	tenant2 "github.com/marcobeierer/go-core/model/tenant"
	server2 "github.com/marcobeierer/go-core/server"
	"github.com/simpledms/simpledms/common/tenantdbs"
)

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

func dbMigrationsMainDB(isDevMode bool, metaPath string, migrationsMainFS fs.FS) *sqlx.MainDB {
	return server2.DBMigrationsMainDB(isDevMode, metaPath, migrationsMainFS)
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
