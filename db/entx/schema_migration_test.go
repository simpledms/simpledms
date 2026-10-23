package entx_test

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	migratetenant "github.com/simpledms/simpledms/db/enttenant/migrate"
)

func TestSchemaIntegrityMigrationsPreserveExistingData(t *testing.T) {
	mainFS, err := migratemain.NewMigrationsMainFS()
	if err != nil {
		t.Fatal(err)
	}
	tenantFS, err := migratetenant.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		migrations fs.FS
		seed       string
		counts     map[string]int
	}{
		{
			name:       "main",
			migrations: mainFS,
			seed: `INSERT INTO accounts
(id, public_id, created_at, updated_at, email, first_name, last_name, language,
two_factor_auth_recovery_code_hashes, role)
VALUES (1, 'account', '2026-01-01', '2026-01-01', 'member@example.com', 'Member', '', 'English', '[]', 'User');
INSERT INTO sessions
(value, is_temporary_session, expires_at, deletable_at, created_at, updated_at, account_id)
VALUES ('session', false, '2027-01-01', '2027-01-01', '2026-01-01', '2026-01-01', 1);`,
			counts: map[string]int{"accounts": 1, "sessions": 1},
		},
		{
			name:       "tenant",
			migrations: tenantFS,
			seed: `INSERT INTO spaces (id, public_id, name) VALUES (1, 'space', 'Space');
INSERT INTO users (id, public_id, created_at, updated_at, account_id, role, email, first_name)
VALUES (1, 'user', '2026-01-01', '2026-01-01', 1, 'User', 'member@example.com', 'Member');
INSERT INTO files
(id, public_id, created_at, updated_at, indexed_at, ocr_last_tried_at, name, is_directory, space_id)
VALUES (1, 'file', '2026-01-01', '2026-01-01', '2026-01-01', '2026-01-01', 'invoice.pdf', false, 1);
INSERT INTO document_types (id, name, space_id) VALUES (1, 'Invoice', 1);
INSERT INTO tags (id, name, type, space_id) VALUES (1, 'Status', 'Group', 1);
INSERT INTO properties (id, name, type, space_id) VALUES (1, 'Amount', 'Money', 1);
INSERT INTO attributes (id, type, tag_id, document_type_id, space_id) VALUES (1, 'Tag', 1, 1, 1);
INSERT INTO attributes (id, type, property_id, document_type_id, space_id) VALUES (2, 'Field', 1, 1, 1);
INSERT INTO file_property_assignments (file_id, property_id, space_id, number_value)
VALUES (1, 1, 1, 0);`,
			counts: map[string]int{
				"spaces": 1, "users": 1, "files": 1, "document_types": 1,
				"tags": 1, "properties": 1, "attributes": 2, "file_property_assignments": 1,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, migrationName := databaseBeforeIntegrityMigration(t, test.migrations)
			if _, err := db.Exec(test.seed); err != nil {
				t.Fatal(err)
			}
			for _, direction := range []string{"up", "down", "up"} {
				script, err := fs.ReadFile(test.migrations, migrationName+"."+direction+".sql")
				if err != nil {
					t.Fatal(err)
				}
				if err := executeSchemaMigration(db, string(script)); err != nil {
					t.Fatalf("%s migration: %v", direction, err)
				}
				for table, want := range test.counts {
					var got int
					if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&got); err != nil {
						t.Fatal(err)
					}
					if got != want {
						t.Fatalf("%s: %s has %d rows, want %d", direction, table, got, want)
					}
				}
				var violations int
				if err := db.QueryRow("SELECT count(*) FROM pragma_foreign_key_check").Scan(&violations); err != nil || violations != 0 {
					t.Fatalf("%s: foreign-key violations=%d, error=%v", direction, violations, err)
				}
			}
			if test.name == "tenant" {
				if _, err := db.Exec("UPDATE attributes SET tag_id = NULL WHERE id = 1"); err == nil {
					t.Fatal("upgraded database did not enforce the attribute target constraint")
				}
				if _, err := db.Exec("UPDATE file_property_assignments SET text_value = ''"); err == nil {
					t.Fatal("upgraded database did not enforce the single-value constraint")
				}
				if _, err := db.Exec(`UPDATE users SET deleted_at = '2026-10-23' WHERE id = 1;
INSERT INTO users (public_id, created_at, updated_at, account_id, role, email, first_name)
VALUES ('replacement', '2026-10-23', '2026-10-23', 2, 'User', 'member@example.com', 'Replacement');`); err != nil {
					t.Fatalf("upgraded database did not release the deleted email: %v", err)
				}
			}
		})
	}
}

func TestSchemaIntegrityMigrationRejectsInvalidLegacyDataWithoutDeletingIt(t *testing.T) {
	migrations, err := migratetenant.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	db, name := databaseBeforeIntegrityMigration(t, migrations)
	if _, err := db.Exec(`INSERT INTO spaces (id, public_id, name) VALUES (1, 'space', 'Space');
INSERT INTO document_types (id, name, space_id) VALUES (1, 'Invoice', 1);
INSERT INTO attributes (id, type, document_type_id, space_id) VALUES (1, 'Tag', 1, 1);`); err != nil {
		t.Fatal(err)
	}
	script, err := fs.ReadFile(migrations, name+".up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := executeSchemaMigration(db, string(script)); err == nil {
		t.Fatal("migration accepted a legacy attribute without its target")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM attributes WHERE id = 1").Scan(&count); err != nil || count != 1 {
		t.Fatalf("failed migration lost original data: count=%d error=%v", count, err)
	}
}

func databaseBeforeIntegrityMigration(t *testing.T, migrations fs.FS) (*sql.DB, string) {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "schema.sqlite3")+"?_foreign_keys=0")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_schema_integrity_and_lookup_indexes.up.sql") {
			return db, strings.TrimSuffix(entry.Name(), ".up.sql")
		}
		script, err := fs.ReadFile(migrations, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if err := executeSchemaMigration(db, string(script)); err != nil {
			t.Fatalf("replay %s: %v", entry.Name(), err)
		}
	}
	t.Fatal("schema integrity migration is missing")
	return nil, ""
}

func executeSchemaMigration(db *sql.DB, query string) error {
	// Match golang-migrate's transaction wrapping and FK-disabled migration connection.
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(query); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}
