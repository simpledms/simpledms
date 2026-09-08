package server

import (
	"context"
	"testing"

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

	for _, stage := range []string{"initialized", "no-pending-migrations", "missing-view-repaired"} {
		if stage == "missing-view-repaired" {
			_, err := tenantDB.ReadWriteConn.ExecContext(ctx, "DROP VIEW resolved_tag_assignments")
			if err != nil {
				t.Fatalf("drop resolved tag assignments view: %v", err)
			}
			if _, err := tenantDB.ReadOnlyConn.ResolvedTagAssignment.Query().All(ctx); err == nil {
				t.Fatal("resolved tag assignments query succeeded after dropping its view")
			}
		}
		if stage != "initialized" {
			if err := model.ExecuteDBMigrations(false, harness.metaPath, migrations, tenantDB); err != nil {
				t.Fatalf("%s: execute production migrations: %v", stage, err)
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
