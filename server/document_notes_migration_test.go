package server

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	migratex "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/db/enttenant"
	migratetenant "github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
)

func TestDocumentNotesProductionMigrationFreshAndPopulated(t *testing.T) {
	const version = uint(20260907090139)
	migrations, err := migratetenant.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	script, err := fs.ReadFile(migrations, "20260907090139_document_notes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	// This migration must be strictly additive, not Atlas' SQLite table-copy strategy.
	for _, forbidden := range []string{
		"DROP TABLE", "ALTER TABLE", "INSERT INTO", "CREATE TABLE `new_",
	} {
		if strings.Contains(strings.ToUpper(string(script)), strings.ToUpper(forbidden)) {
			t.Fatalf("notes migration rebuilds/mutates existing tables: %s", forbidden)
		}
	}
	for _, populated := range []bool{false, true} {
		name := "fresh"
		if populated {
			name = "populated-pre-notes"
		}
		t.Run(name, func(t *testing.T) {
			source, err := iofs.New(migrations, ".")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "tenant.sqlite3")
			// Match production's embedded source and SQLite migration driver, not Schema.Create.
			migration, err := migratex.NewWithSourceInstance("migrationsTenantFS", source,
				"sqlite3://"+path+"?_foreign_keys=0")
			if err != nil {
				_ = source.Close()
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if sourceErr, dbErr := migration.Close(); sourceErr != nil || dbErr != nil {
					t.Errorf("close migrations: %v / %v", sourceErr, dbErr)
				}
			})
			client, err := enttenant.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := client.Close(); err != nil {
					t.Error(err)
				}
			})
			ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
			var file *enttenant.File
			var before string
			if populated {
				previous, err := source.Prev(version)
				if err != nil {
					t.Fatal(err)
				}
				if err := migration.Migrate(previous); err != nil {
					t.Fatalf("replay pre-notes migrations: %v", err)
				}
				spacex := client.Space.Create().SetName("Existing Space").SaveX(ctx)
				file = client.File.Create().SetSpaceID(spacex.ID).SetName("existing.pdf").
					SetIsDirectory(false).SetIsInInbox(true).SetIndexedAt(time.Now()).
					SetNotes("legacy\n  <plain text> & preserved").SetOcrContent("existing OCR").SaveX(ctx)
				before = client.File.GetX(ctx, file.ID).String()
			}
			if err := migration.Migrate(version); err != nil {
				t.Fatalf("apply production notes migration: %v", err)
			}
			gotVersion, dirty, err := migration.Version()
			if err != nil || dirty || gotVersion != version {
				t.Fatalf("migration state: %d, dirty=%t, %v", gotVersion, dirty, err)
			}
			if count := client.DocumentNote.Query().CountX(ctx); count != 0 {
				t.Fatalf("migration fabricated %d note rows", count)
			}
			if populated && client.File.GetX(ctx, file.ID).String() != before {
				t.Fatal("migration changed existing file/legacy fields")
			}
		})
	}
}

func TestDocumentNoteTitlesProductionMigration(t *testing.T) {
	const version = uint(20260907210921)
	migrations, err := migratetenant.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	script, err := fs.ReadFile(migrations, "20260907210921_document_note_titles.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(script)) != "-- add column \"title\" to table: \"document_notes\"\n"+
		"ALTER TABLE `document_notes` ADD COLUMN `title` text NULL;" {
		t.Fatalf("title migration must only add its column: %s", script)
	}
	source, err := iofs.New(migrations, ".")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "tenant.sqlite3")
	migration, err := migratex.NewWithSourceInstance("migrationsTenantFS", source,
		"sqlite3://"+path+"?_foreign_keys=0")
	if err != nil {
		_ = source.Close()
		t.Fatal(err)
	}
	defer migration.Close()
	previous, err := source.Prev(version)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Migrate(previous); err != nil {
		t.Fatal(err)
	}
	client, err := enttenant.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	spacex := client.Space.Create().SetName("Existing notes").SaveX(ctx)
	file := client.File.Create().SetSpaceID(spacex.ID).SetName("existing.pdf").
		SetIsDirectory(false).SetIndexedAt(time.Now()).SaveX(ctx)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// SQL is intentional: the previous schema cannot use the newly generated title column.
	_, err = db.Exec(`INSERT INTO document_notes
		(id, public_id, body, space_id, file_id, replaced_by_id) VALUES
		(1, 'old', 'original body', ?, ?, 2), (2, 'new', 'successor body', ?, ?, NULL)`,
		spacex.ID, file.ID, spacex.ID, file.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Migrate(version); err != nil {
		t.Fatal(err)
	}
	old := client.DocumentNote.GetX(ctx, 1)
	next := client.DocumentNote.GetX(ctx, 2)
	if old.Title != "" || next.Title != "" || old.Body != "original body" ||
		next.Body != "successor body" || old.ReplacedByID != next.ID ||
		old.AuthorID != 0 || old.AuthoredAt != nil || client.DocumentNote.Query().CountX(ctx) != 2 {
		t.Fatalf("title migration changed legacy content/history: %v / %v", old, next)
	}
}
