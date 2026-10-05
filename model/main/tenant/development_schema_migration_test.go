package tenant

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	migratex "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	migratetenant "github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
)

func TestDevelopmentSchemaCreateAddsIndexedColumnsWithoutRebuildingTables(t *testing.T) {
	const firstVersion = uint(20260929123641)
	migrations, err := migratetenant.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	source, err := iofs.New(migrations, ".")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "tenant.sqlite3")
	migration, err := migratex.NewWithSourceInstance(
		"migrationsTenantFS",
		source,
		"sqlite3://"+path+"?_foreign_keys=0",
	)
	if err != nil {
		_ = source.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = migration.Close() })
	previous, err := source.Prev(firstVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Migrate(previous); err != nil {
		t.Fatalf("replay pre-public-ID migrations: %v", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO spaces (id, public_id, name) VALUES (1, 'space', 'Existing Space')`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO document_types (id, name, space_id) VALUES (1, 'Existing Type', 1)`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO attributes (name, type, space_id, document_type_id) ` +
			`VALUES ('Tag', 'Tag', 1, 1)`,
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	readOnlyURL := fmt.Sprintf("file:%s?%s", path, sqlx.SQLiteQueryParamsReadOnly)
	readWriteURL := fmt.Sprintf("file:%s?%s", path, sqlx.SQLiteQueryParamsReadWrite)
	tenantDB := sqlx.NewTenantDB(readOnlyURL, readWriteURL)
	t.Cleanup(func() { _ = tenantDB.Close() })
	ctx := context.Background()
	if err := addMetadataPublicIDColumns(ctx, tenantDB); err != nil {
		t.Fatalf("add metadata public-ID columns: %v", err)
	}
	if err := tenantDB.ReadWriteConn.Schema.Create(
		ctx,
		migratetenant.WithDropIndex(true),
		migratetenant.WithDropColumn(true),
		entx.WithFileSourceDefault(),
		entx.WithInboxTransferDefault(),
	); err != nil {
		t.Fatalf("add development schema indexes: %v", err)
	}
	rows, err := tenantDB.ReadOnlyConn.QueryContext(
		ctx,
		`SELECT COUNT(*) FROM attributes WHERE document_type_id = 1`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()
	var attributeCount int
	if !rows.Next() {
		t.Fatal("attribute count query returned no result")
	}
	if err := rows.Scan(&attributeCount); err != nil {
		t.Fatal(err)
	}
	if attributeCount != 1 {
		t.Fatalf("development schema migration lost referenced data: got %d", attributeCount)
	}
}
