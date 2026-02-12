package tenant

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"filippo.io/age"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	accountm "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/pathx"
	"github.com/simpledms/simpledms/util/e"

	"entgo.io/ent/dialect"
	securejoin "github.com/cyphar/filepath-securejoin"
	migratex "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/schema"
)

type Tenant struct {
	Data *entmain.Tenant
}

func NewTenant(data *entmain.Tenant) *Tenant {
	return &Tenant{
		Data: data,
	}
}

// TODO should be other way around account.IsOwner()?
func (qq *Tenant) IsOwner(account *accountm.Account) bool {
	return qq.Data.QueryAccountAssignment().Where(
		tenantaccountassignment.AccountID(account.Data.ID),
		tenantaccountassignment.RoleEQ(tenantrole.Owner),
	).ExistX(context.Background()) // TODO pass in ctx?
}

func (qq *Tenant) IsInitialized() bool {
	return qq.Data.InitializedAt != nil && !qq.Data.InitializedAt.IsZero()
}

// caller has to close tenantDB, usually done in defer function in main.go
func (qq *Tenant) Init(
	devMode bool,
	metaPath string,
	migrationsTenantFS fs.FS,
) (*sqlx.TenantDB, error) {
	// create directories
	storagePath := pathx.StoragePath(metaPath, qq.Data.PublicID.String())
	err := os.MkdirAll(storagePath, 0777)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if encryptor.NilableX25519MainIdentity == nil {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "App not unlocked yet. Please try again later.")
	}

	tenantIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// added to clients list on demand
	tenantDB, err := qq.initDB(devMode, metaPath, migrationsTenantFS, tenantIdentity)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return tenantDB, nil
}

// caller has to close db
func (qq *Tenant) OpenDB(devMode bool, metaPath string) (*sqlx.TenantDB, error) {
	if !qq.IsInitialized() {
		// TODO StatusAccepted or StatusServiceUnavailable? the latter may not process nackbar...
		return nil, e.NewHTTPErrorf(http.StatusAccepted, "Tenant not initialized yet. Please try again later.")
	}

	client, err := qq.openDB(devMode, metaPath, false)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Error opening tenant database. Please try again later.")
	}

	return client, err
}

func (qq *Tenant) tenantDataSourceURL(metaPath string, shouldCreateDirs bool) (string, string, error) {
	tenantDBPath, err := qq.dbPath(metaPath, shouldCreateDirs)
	if err != nil {
		log.Println(err)
		return "", "", err
	}

	return fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadOnly),
		fmt.Sprintf("file:%s?%s", tenantDBPath, sqlx.SQLiteQueryParamsReadWrite),
		nil
}

// caller has to close connection
func (qq *Tenant) openDB(devMode bool, metaPath string, shouldCreateDirs bool) (*sqlx.TenantDB, error) {
	readOnlyTenantDataSourceURL, readWriteTenantDataSourceURL, err := qq.tenantDataSourceURL(metaPath, shouldCreateDirs)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	tenantClient := sqlx.NewTenantDB(readOnlyTenantDataSourceURL, readWriteTenantDataSourceURL)

	if devMode {
		tenantClient.Debug()
	}

	return tenantClient, nil
}

func (qq *Tenant) dbPath(metaPath string, shouldCreateDirs bool) (string, error) {
	tenantDBPath, err := pathx.TenantDBPath(metaPath, qq.Data.PublicID.String())
	if err != nil {
		log.Println(err)
		return "", err
	}
	if shouldCreateDirs {
		err = os.MkdirAll(tenantDBPath, 0777)
		if err != nil {
			log.Println(err)
			return "", err
		}
	}
	tenantDBPath, err = securejoin.SecureJoin(tenantDBPath, "tenant.sqlite3")
	if err != nil {
		log.Println(err)
		return "", err
	}
	return tenantDBPath, nil
}

func (qq *Tenant) ExecuteDBMigrations(
	devMode bool,
	metaPath string,
	migrationsTenantFS fs.FS,
	tenantDB *sqlx.TenantDB, // TODO?
) error {
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

	if devMode {
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
		drv, err := iofs.New(migrationsTenantFS, ".")
		if err != nil {
			log.Printf("failed reading migration filesystem: %v", err)
			return err
		}

		// TODO shouldCreateDirs okay? should already exists...
		_, readWriteTenantDataSourceURL, err := qq.tenantDataSourceURL(metaPath, false)
		if err != nil {
			log.Println(err)
			return err
		}

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

// caller has to close tenantDB, usually done in main.go defer if added to tenantDBs map
func (qq *Tenant) initDB(
	devMode bool,
	metaPath string,
	migrationsTenantFS fs.FS,
	tenantIdentity *age.X25519Identity,
) (*sqlx.TenantDB, error) {
	ctx := context.Background()

	// TODO not very efficient also opened in ExecuteDBMigrations
	tenantClient, err := qq.openDB(devMode, metaPath, true)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// defer tenantClient.Close() // closed in main.go defer

	err = qq.ExecuteDBMigrations(devMode, metaPath, migrationsTenantFS, tenantClient)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	qq.Data.Update().
		SetInitializedAt(time.Now()).
		SetX25519IdentityEncrypted(entx.NewEncryptedX25519Identity(tenantIdentity)).
		SaveX(ctx)

	// create users in tenant db
	accounts := qq.Data.QueryAccounts().WithTenantAssignment().AllX(ctx)
	for _, accountx := range accounts {
		if len(accountx.Edges.TenantAssignment) == 0 || len(accountx.Edges.TenantAssignment) > 1 {
			// should never happen, should always be one
			log.Printf("account %d has %d tenant assignments", accountx.ID, len(accountx.Edges.TenantAssignment))
			continue
		}

		tenantClient.ReadWriteConn.User.Create().
			SetAccountID(accountx.ID).
			SetRole(accountx.Edges.TenantAssignment[0].Role).
			SetEmail(accountx.Email).
			SetFirstName(accountx.FirstName).
			SetLastName(accountx.LastName).
			SaveX(ctx)
	}

	return tenantClient, nil
}

func (qq *Tenant) Name() string {
	return qq.Data.Name
}

func (qq *Tenant) HasAccount(ctx ctxx.Context, accountm *accountm.Account) bool {
	return qq.Data.QueryAccountAssignment().Where(
		tenantaccountassignment.AccountID(accountm.Data.ID),
	).ExistX(ctx)
}
