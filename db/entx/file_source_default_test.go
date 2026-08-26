package entx

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
	_ "github.com/mattn/go-sqlite3"
)

func TestFileSourceSchemaDefaultMigratesExistingRows(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "tenant.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := db.Exec("CREATE TABLE files (id integer NOT NULL PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO files (id) VALUES (1)"); err != nil {
		t.Fatal(err)
	}

	id := &schema.Column{
		Name: "id",
		Type: field.TypeInt,
	}
	source := &schema.Column{
		Name:    "source",
		Type:    field.TypeEnum,
		Enums:   []string{"UnknownLegacy", "WebDAV"},
		Default: schema.Expr("'UnknownLegacy'"),
	}
	files := schema.NewTable("files").
		AddPrimary(id).
		AddColumn(source).
		AddIndex("files_source", false, []string{"source"})
	migrate, err := schema.NewMigrate(
		entsql.OpenDB(dialect.SQLite, db),
		schema.WithDropIndex(true),
		schema.WithDropColumn(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Create(context.Background(), files); err != nil {
		t.Fatal(err)
	}

	var got string
	if err := db.QueryRow("SELECT source FROM files WHERE id = 1").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "UnknownLegacy" {
		t.Fatalf("expected UnknownLegacy source, got %q", got)
	}
}
