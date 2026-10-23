package server

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/model/main/tenant"
)

func TestTenantProductionMigrationsRestoreViews(t *testing.T) {
	harness := newActionTestHarness(t)
	_, tenantx := signUpAccount(t, harness, "tenant-views@example.com")
	migrations, err := migrate.NewMigrationsTenantFS()
	if err != nil {
		t.Fatalf("load tenant migrations: %v", err)
	}
	model := tenant.NewTenant(tenantx)
	// Do not use initTenantDB: its development mode uses Schema.Create.
	tenantDB, err := model.Init(false, harness.metaPath, migrations)
	if err != nil {
		t.Fatalf("initialize tenant with production migrations: %v", err)
	}
	t.Cleanup(func() {
		if err := tenantDB.Close(); err != nil {
			t.Errorf("close tenant database: %v", err)
		}
	})
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	space := tenantDB.ReadWriteConn.Space.Query().OnlyX(ctx)
	doc := tenantDB.ReadWriteConn.File.Create().SetSpaceID(space.ID).SetName("regression.pdf").
		SetIsDirectory(false).SetIndexedAt(time.Now()).SetOcrContent("originalword").SaveX(ctx)
	if _, err := tenantDB.ReadWriteConn.ExecContext(ctx,
		"INSERT INTO file_searches(file_searches, rank) VALUES ('automerge', 7)"); err != nil {
		t.Fatal(err)
	}

	for _, stage := range []string{
		"initialized", "no-pending-migrations", "missing-view-repaired",
		"legacy-trigger-upgraded", "missing-trigger-repaired", "missing-index-repaired",
	} {
		if stage == "missing-view-repaired" {
			_, err := tenantDB.ReadWriteConn.ExecContext(ctx, "DROP VIEW resolved_tag_assignments")
			if err != nil {
				t.Fatalf("drop resolved tag assignments view: %v", err)
			}
			if _, err := tenantDB.ReadOnlyConn.ResolvedTagAssignment.Query().All(ctx); err == nil {
				t.Fatal("resolved tag assignments query succeeded after dropping its view")
			}
		}
		if stage == "missing-trigger-repaired" {
			if _, err := tenantDB.ReadWriteConn.ExecContext(ctx, "DROP TRIGGER files_au"); err != nil {
				t.Fatal(err)
			}
			doc.Update().SetOcrContent("repairedword").SaveX(ctx)
		}
		if stage == "legacy-trigger-upgraded" {
			// Recreate the previous broad trigger independently of production SQL.
			if _, err := tenantDB.ReadWriteConn.ExecContext(ctx, `DROP TRIGGER files_au;
CREATE TRIGGER files_au AFTER UPDATE ON files BEGIN
  INSERT INTO file_searches(file_searches, rowid, space_id, is_directory, is_in_inbox, name, ocr_content)
  VALUES('delete', old.id, old.space_id, old.is_directory, old.is_in_inbox, old.name, old.ocr_content);
  INSERT INTO file_searches(rowid, space_id, is_directory, is_in_inbox, name, ocr_content)
  VALUES(new.id, new.space_id, new.is_directory, new.is_in_inbox, new.name, new.ocr_content);
END;`); err != nil {
				t.Fatalf("install legacy files update trigger: %v", err)
			}
			doc.Update().SetName("regression-upgraded.pdf").SetOcrContent("upgradedcontent").SaveX(ctx)
			if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
				"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'upgradedcontent'"); got != 1 {
				t.Fatalf("legacy trigger did not index changed content before migration: %d", got)
			}
		}
		if stage == "missing-index-repaired" {
			if _, err := tenantDB.ReadWriteConn.ExecContext(ctx, "DROP TABLE file_searches"); err != nil {
				t.Fatal(err)
			}
		}
		if stage != "initialized" {
			if err := model.ExecuteDBMigrations(false, harness.metaPath, migrations, tenantDB); err != nil {
				t.Fatalf("%s: execute production migrations: %v", stage, err)
			}
		}
		switch stage {
		case "no-pending-migrations", "missing-view-repaired", "legacy-trigger-upgraded":
			if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
				"SELECT v FROM file_searches_config WHERE k = 'automerge'"); got != 7 {
				t.Fatalf("%s rebuilt a healthy FTS index (automerge=%d)", stage, got)
			}
		}
		if stage == "legacy-trigger-upgraded" {
			if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
				"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'originalword'"); got != 0 {
				t.Fatalf("%s retained old indexed text: %d", stage, got)
			}
			if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
				"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'upgradedcontent'"); got != 1 {
				t.Fatalf("%s lost new indexed text: %d", stage, got)
			}
			before := tenantSQLInt(t, tenantDB.ReadWriteConn, "SELECT total_changes()")
			if _, err := tenantDB.ReadWriteConn.ExecContext(ctx,
				"UPDATE files SET notes = 'metadata only' WHERE id = ?", doc.ID); err != nil {
				t.Fatalf("%s metadata-only update: %v", stage, err)
			}
			delta := tenantSQLInt(t, tenantDB.ReadWriteConn, "SELECT total_changes()") - before
			if delta != 1 {
				t.Fatalf("%s metadata update caused %d database changes, want 1", stage, delta)
			}
		}
		if strings.HasPrefix(stage, "missing-trigger") || strings.HasPrefix(stage, "missing-index") {
			if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
				"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'repairedword'"); got != 1 {
				t.Fatalf("%s did not recover writes missed by the trigger", stage)
			}
		}
		if _, err := tenantDB.ReadOnlyConn.ResolvedTagAssignment.Query().All(ctx); err != nil {
			t.Fatalf("%s: query resolved tag assignments through read-only connection: %v", stage, err)
		}
		rows, err := tenantDB.ReadOnlyConn.QueryContext(ctx,
			"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'regression'")
		if err != nil {
			t.Fatalf("%s: query FTS table: %v", stage, err)
		}
		var count int
		if !rows.Next() {
			t.Fatalf("%s: missing FTS count row: %v", stage, rows.Err())
		}
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("%s: scan FTS count: %v", stage, err)
		}
		if err := rows.Close(); err != nil {
			t.Fatalf("%s: close FTS rows: %v", stage, err)
		}
		if count != 1 {
			t.Fatalf("%s: expected one regression document in FTS, got %d", stage, count)
		}
		rows, err = tenantDB.ReadOnlyConn.QueryContext(ctx, `SELECT count(*) FROM sqlite_master
			WHERE type = 'trigger' AND tbl_name = 'files'
			AND name IN ('files_ai', 'files_ad', 'files_au')`)
		if err != nil {
			t.Fatalf("%s: query FTS triggers: %v", stage, err)
		}
		if !rows.Next() {
			t.Fatalf("%s: missing trigger count row: %v", stage, rows.Err())
		}
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("%s: scan trigger count: %v", stage, err)
		}
		if err := rows.Close(); err != nil {
			t.Fatalf("%s: close trigger rows: %v", stage, err)
		}
		if count != 3 {
			t.Fatalf("%s: expected 3 FTS triggers, got %d", stage, count)
		}
		t.Logf("%s: resolved tag assignments, FTS query, and all 3 FTS triggers verified", stage)
	}
}

func TestFileSearchUpdatesOnlyForChangedIndexedContent(t *testing.T) {
	harness := newActionTestHarness(t)
	_, tenantx := signUpAccount(t, harness, "search-triggers@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	spaces := tenantDB.ReadWriteConn.Space.Query().AllX(ctx)
	space := spaces[0]
	destination := tenantDB.ReadWriteConn.Space.Create().SetName("Destination").SaveX(ctx)
	doc := tenantDB.ReadWriteConn.File.Create().SetSpaceID(space.ID).SetName("oldfilename.pdf").
		SetIsDirectory(false).SetIndexedAt(time.Now()).SetOcrContent("oldcontentword").SaveX(ctx)
	client := tenantDB.ReadWriteConn
	assertMatch := func(term string, count int) {
		t.Helper()
		if got := tenantSQLInt(t, client,
			"SELECT count(*) FROM file_searches WHERE file_searches MATCH ?", term); got != count {
			t.Fatalf("search %q: got %d matches, want %d", term, got, count)
		}
	}
	assertMatch("oldcontentword", 1)
	for _, update := range []string{
		"UPDATE files SET notes = 'metadata', ocr_retry_count = 1 WHERE id = ?",
		"UPDATE files SET name = name, ocr_content = ocr_content WHERE id = ?",
	} {
		before := tenantSQLInt(t, client, "SELECT total_changes()")
		if _, err := client.ExecContext(ctx, update, doc.ID); err != nil {
			t.Fatal(err)
		}
		if delta := tenantSQLInt(t, client, "SELECT total_changes()") - before; delta != 1 {
			t.Fatalf("metadata/unchanged text rewrote FTS data: %d database changes", delta)
		}
	}
	doc.Update().SetSpaceID(destination.ID).SetIsInInbox(true).SaveX(ctx)
	if got := tenantSQLInt(t, client, `SELECT count(*) FROM file_searches
WHERE file_searches MATCH 'oldcontentword' AND space_id = ? AND is_in_inbox = true`,
		destination.ID); got != 1 {
		t.Fatal("external-content search did not follow the document's Space and Inbox")
	}
	if got := tenantSQLInt(t, client, `SELECT count(*) FROM file_searches
WHERE file_searches MATCH 'oldcontentword' AND space_id = ?`, space.ID); got != 0 {
		t.Fatal("search still exposed the transferred document in its old Space")
	}
	doc.Update().SetName("newfilename.pdf").SetOcrContent("newcontentword").SaveX(ctx)
	assertMatch("oldfilename", 0)
	assertMatch("oldcontentword", 0)
	assertMatch("newfilename", 1)
	assertMatch("newcontentword", 1)
	if _, err := client.ExecContext(ctx, "DELETE FROM files WHERE id = ?", doc.ID); err != nil {
		t.Fatal(err)
	}
	assertMatch("newcontentword", 0)
}

func TestTenantMigrationsPreserveSearchUnlessFilesAreRebuilt(t *testing.T) {
	harness := newActionTestHarness(t)
	_, tenantx := signUpAccount(t, harness, "search-migrations@example.com")
	migrations, err := migrate.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	model := tenant.NewTenant(tenantx)
	tenantDB, err := model.Init(false, harness.metaPath, migrations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tenantDB.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	space := tenantDB.ReadWriteConn.Space.Query().OnlyX(ctx)
	tenantDB.ReadWriteConn.File.Create().SetSpaceID(space.ID).SetName("migrationword.pdf").
		SetIsDirectory(false).SetIndexedAt(time.Now()).SaveX(ctx)
	if _, err := tenantDB.ReadWriteConn.ExecContext(ctx,
		"INSERT INTO file_searches(file_searches, rank) VALUES ('automerge', 7)"); err != nil {
		t.Fatal(err)
	}
	upgrades := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(migrations, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		upgrades[entry.Name()] = &fstest.MapFile{Data: data}
	}
	upgrades["20990101000000_index_only.up.sql"] = &fstest.MapFile{
		Data: []byte("CREATE INDEX test_file_name ON files(name);"),
	}
	if err := model.ExecuteDBMigrations(false, harness.metaPath, upgrades, tenantDB); err != nil {
		t.Fatal(err)
	}
	if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
		"SELECT v FROM file_searches_config WHERE k = 'automerge'"); got != 7 {
		t.Fatal("index-only migration rebuilt healthy FTS data")
	}
	rows, err := tenantDB.ReadOnlyConn.QueryContext(ctx,
		"SELECT sql FROM sqlite_master WHERE name = 'files' AND type = 'table'")
	if err != nil {
		t.Fatal(err)
	}
	var definition string
	if !rows.Next() {
		t.Fatal("files table is missing")
	}
	if err := rows.Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	// Exercise Atlas's table replacement shape, which removes the old table's triggers.
	upgrades["20990101000001_rebuild_files.up.sql"] = &fstest.MapFile{
		Data: []byte("CREATE TABLE new_files " + definition[strings.Index(definition, "("):] + ";" +
			"INSERT INTO new_files SELECT * FROM files; DROP TABLE files;" +
			"ALTER TABLE new_files RENAME TO files;"),
	}
	if err := model.ExecuteDBMigrations(false, harness.metaPath, upgrades, tenantDB); err != nil {
		t.Fatal(err)
	}
	if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
		"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'migrationword'"); got != 1 {
		t.Fatal("table replacement failed to restore searchable existing data")
	}
	tenantDB.ReadWriteConn.File.Create().SetSpaceID(space.ID).SetName("postmigrationword.pdf").
		SetIsDirectory(false).SetIndexedAt(time.Now()).SaveX(ctx)
	if got := tenantSQLInt(t, tenantDB.ReadWriteConn,
		"SELECT count(*) FROM file_searches WHERE file_searches MATCH 'postmigrationword'"); got != 1 {
		t.Fatal("table replacement failed to restore the insert trigger")
	}
}

func tenantSQLInt(t *testing.T, client *enttenant.Client, query string, args ...any) int {
	t.Helper()
	rows, err := client.QueryContext(context.Background(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()
	if !rows.Next() {
		t.Fatal("query returned no row", rows.Err())
	}
	var result int
	if err := rows.Scan(&result); err != nil {
		t.Fatal(err)
	}
	return result
}
