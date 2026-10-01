package server

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	migratex "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/enttenant"
	documenttypex "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/file"
	migratetenant "github.com/simpledms/simpledms/db/enttenant/migrate"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	propertyx "github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
)

func TestMCPMetadataProductionMigrationFreshAndPopulated(t *testing.T) {
	const columnsVersion uint = 20260929123641
	const indexesVersion uint = 20260929123746

	migrations, err := migratetenant.NewMigrationsTenantFS()
	if err != nil {
		t.Fatal(err)
	}
	columns, err := fs.ReadFile(migrations, "20260929123641_metadata_public_ids.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(string(columns)), "DROP TABLE") ||
		strings.Contains(strings.ToUpper(string(columns)), "CREATE TABLE `NEW_") {
		t.Fatal("metadata public-ID column migration rebuilds existing tables")
	}
	indexes, err := fs.ReadFile(migrations, "20260929123746_metadata_public_id_indexes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(string(indexes)), "DROP TABLE") ||
		strings.Contains(strings.ToUpper(string(indexes)), "INSERT INTO") {
		t.Fatal("metadata public-ID index migration rewrites existing tables")
	}

	for _, populated := range []bool{false, true} {
		name := "fresh"
		if populated {
			name = "populated"
		}
		t.Run(name, func(t *testing.T) {
			source, err := iofs.New(migrations, ".")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "tenant.sqlite3")
			migration, err := migratex.NewWithSourceInstance("migrationsTenantFS", source,
				"sqlite3://"+path+"?_foreign_keys=0&_journal_mode=wal")
			if err != nil {
				_ = source.Close()
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = migration.Close() })
			ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
			previous, err := source.Prev(columnsVersion)
			if err != nil {
				t.Fatal(err)
			}
			if err := migration.Migrate(previous); err != nil {
				t.Fatal(err)
			}

			if populated {
				seedDB, err := sql.Open("sqlite3", path)
				if err != nil {
					t.Fatal(err)
				}
				if err := seedMetadataBeforePublicIDColumns(seedDB); err != nil {
					_ = seedDB.Close()
					t.Fatal(err)
				}
				if err := seedDB.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := migration.Migrate(columnsVersion); err != nil {
				t.Fatal(err)
			}
			client, err := enttenant.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}

			var before struct {
				tags, properties, documentTypes, attributes, tagAssignments, propertyAssignments int
			}
			if populated {
				before.tags = client.Tag.Query().CountX(ctx)
				before.properties = client.Property.Query().CountX(ctx)
				before.documentTypes = client.DocumentType.Query().CountX(ctx)
				before.attributes = client.Attribute.Query().CountX(ctx)
				before.tagAssignments = client.TagAssignment.Query().CountX(ctx)
				before.propertyAssignments = client.FilePropertyAssignment.Query().CountX(ctx)
				if !client.Tag.Query().Where(tag.PublicIDIsNil()).ExistX(ctx) ||
					!client.Property.Query().Where(propertyx.PublicIDIsNil()).ExistX(ctx) ||
					!client.DocumentType.Query().Where(documenttypex.PublicIDIsNil()).ExistX(ctx) {
					t.Fatal("pre-column metadata fixture unexpectedly had public IDs")
				}
			}
			if err := migration.Migrate(indexesVersion); err != nil {
				_ = client.Close()
				t.Fatal(err)
			}
			if err := client.Close(); err != nil {
				t.Fatal(err)
			}

			tenantDB := sqlx.NewTenantDB("file:"+path+"?"+sqlx.SQLiteQueryParamsReadOnly,
				"file:"+path+"?"+sqlx.SQLiteQueryParamsReadWrite)
			t.Cleanup(func() { _ = tenantDB.Close() })
			runner := tenantdatamigration.NewRunner()
			for _, metadataMigration := range tenantdatamigration.NewMetadataPublicIDMigrations() {
				completed, err := runner.RunToCompletion(ctx, tenantDB, metadataMigration)
				if err != nil || !completed {
					t.Fatalf("metadata migration %s completed=%t, err=%v",
						metadataMigration.Key(), completed, err)
				}
			}
			if !populated {
				if tenantDB.ReadOnlyConn.Tag.Query().CountX(ctx) != 0 ||
					tenantDB.ReadOnlyConn.Property.Query().CountX(ctx) != 0 ||
					tenantDB.ReadOnlyConn.DocumentType.Query().CountX(ctx) != 0 {
					t.Fatal("fresh migration fabricated metadata definitions")
				}
				return
			}
			if tenantDB.ReadOnlyConn.Tag.Query().CountX(ctx) != before.tags ||
				tenantDB.ReadOnlyConn.Property.Query().CountX(ctx) != before.properties ||
				tenantDB.ReadOnlyConn.DocumentType.Query().CountX(ctx) != before.documentTypes ||
				tenantDB.ReadOnlyConn.Attribute.Query().CountX(ctx) != before.attributes ||
				tenantDB.ReadOnlyConn.TagAssignment.Query().CountX(ctx) != before.tagAssignments ||
				tenantDB.ReadOnlyConn.FilePropertyAssignment.Query().CountX(ctx) != before.propertyAssignments {
				t.Fatal("metadata migration changed populated definitions")
			}
			if missing, err := tenantDB.ReadOnlyConn.Tag.Query().Where(tag.PublicIDIsNil()).Exist(ctx); err != nil {
				t.Fatal(err)
			} else if missing {
				t.Fatal("populated Tag did not receive a public ID")
			}
			property := tenantDB.ReadOnlyConn.Property.Query().OnlyX(ctx)
			if property.Name != "Amount" || property.Unit != "CHF" {
				t.Fatalf("property definition changed during migration: %v", property)
			}
			documentType := tenantDB.ReadOnlyConn.DocumentType.Query().OnlyX(ctx)
			if documentType.Name != "Invoice" {
				t.Fatalf("document type changed during migration: %v", documentType)
			}
			filex := tenantDB.ReadOnlyConn.File.Query().Where(file.Name("invoice.pdf")).OnlyX(ctx)
			if filex.DocumentTypeID != documentType.ID {
				t.Fatalf("document type assignment changed during migration: %v", filex)
			}
			attribute := tenantDB.ReadOnlyConn.Attribute.Query().OnlyX(ctx)
			if attribute.Name != "Total" || attribute.PropertyID != property.ID {
				t.Fatalf("document type attribute changed during migration: %v", attribute)
			}
			assignment := tenantDB.ReadOnlyConn.FilePropertyAssignment.Query().OnlyX(ctx)
			if assignment.NumberValue != 12345 {
				t.Fatalf("minor-unit assignment changed during migration: %v", assignment)
			}
			group := tenantDB.ReadOnlyConn.Tag.Query().Where(tag.NameEQ("Group")).OnlyX(ctx)
			composed := tenantDB.ReadOnlyConn.Tag.Query().Where(tag.NameEQ("Composed")).OnlyX(ctx)
			super := tenantDB.ReadOnlyConn.Tag.Query().Where(tag.NameEQ("Super")).OnlyX(ctx)
			if composed.GroupID != group.ID || super.QuerySubTags().CountX(ctx) != 1 {
				t.Fatal("grouped/composed tag relationships changed during migration")
			}
			stableTagID := tenantDB.ReadOnlyConn.Tag.Query().Where(tag.NameEQ("Composed")).OnlyX(ctx).PublicID.String()
			stablePropertyID := property.PublicID.String()
			stableDocumentTypeID := documentType.PublicID.String()
			for _, metadataMigration := range tenantdatamigration.NewMetadataPublicIDMigrations() {
				completed, err := runner.RunToCompletion(ctx, tenantDB, metadataMigration)
				if err != nil || !completed {
					t.Fatalf("repeated metadata migration %s completed=%t, err=%v",
						metadataMigration.Key(), completed, err)
				}
			}
			if tenantDB.ReadOnlyConn.Tag.Query().Where(tag.NameEQ("Composed")).OnlyX(ctx).PublicID.String() != stableTagID ||
				tenantDB.ReadOnlyConn.Property.Query().OnlyX(ctx).PublicID.String() != stablePropertyID ||
				tenantDB.ReadOnlyConn.DocumentType.Query().OnlyX(ctx).PublicID.String() != stableDocumentTypeID {
				t.Fatal("repeated metadata migration changed public IDs")
			}
		})
	}
}

func seedMetadataBeforePublicIDColumns(db *sql.DB) error {
	statements := []string{
		`INSERT INTO spaces (id, public_id, name, is_folder_mode) VALUES
			(1, 'space-old', 'Existing metadata', 0)`,
		`INSERT INTO tags (id, name, type, space_id, group_id) VALUES
			(1, 'Group', 'Group', 1, NULL),
			(2, 'Super', 'Super', 1, 1),
			(3, 'Composed', 'Simple', 1, 1)`,
		`INSERT INTO tag_sub_tags (tag_id, super_tag_id) VALUES (2, 3)`,
		`INSERT INTO properties (id, name, type, unit, space_id) VALUES
			(1, 'Amount', 'Money', 'CHF', 1)`,
		`INSERT INTO document_types (id, name, is_protected, is_disabled, space_id) VALUES
			(1, 'Invoice', 0, 0, 1)`,
		`INSERT INTO attributes (id, name, is_name_giving, is_protected, is_disabled,
			is_required, type, space_id, property_id, document_type_id) VALUES
			(1, 'Total', 0, 0, 0, 1, 'Field', 1, 1, 1)`,
		`INSERT INTO files (id, public_id, created_at, updated_at, name, source, is_directory,
			indexed_at, is_in_inbox, is_root_dir, ocr_content, ocr_retry_count,
			ocr_last_tried_at, space_id, document_type_id) VALUES
			(1, 'file-old', datetime('now'), datetime('now'), 'invoice.pdf', 'UnknownLegacy',
			0, datetime('now'), 1, 0, '', 0, datetime('now'), 1, 1)`,
		`INSERT INTO tag_assignments (id, space_id, tag_id, file_id) VALUES (1, 1, 3, 1)`,
		`INSERT INTO file_property_assignments
			(id, number_value, space_id, file_id, property_id) VALUES (1, 12345, 1, 1, 1)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}
