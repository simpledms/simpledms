package tenantdatamigration

import (
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

func TestMetadataPublicIDMigrationBackfillsInRestartSafeBatches(t *testing.T) {
	ctx, tenantDB := newMigrationTestDB(t)
	space := tenantDB.ReadWriteConn.Space.Create().SetName("Backfill").SaveX(ctx)
	for index := range metadataPublicIDBatchSize + 1 {
		tenantDB.ReadWriteConn.Tag.Create().
			SetSpaceID(space.ID).
			SetName(fmt.Sprintf("Tag %d", index)).
			SetType(tagtype.Simple).
			SaveX(ctx)
	}
	var result sql.Result
	if err := tenantDB.ReadWriteConn.Driver().Exec(
		ctx,
		"UPDATE `tags` SET `public_id` = NULL",
		[]any{},
		&result,
	); err != nil {
		t.Fatal(err)
	}
	createdDuringUpgrade := tenantDB.ReadWriteConn.Tag.Create().
		SetSpaceID(space.ID).
		SetName("created during upgrade").
		SetType(tagtype.Simple).
		SaveX(ctx)
	originalPublicID := createdDuringUpgrade.PublicID

	runner := NewRunner()
	tagMigration := NewMetadataPublicIDMigrations()[0]
	completed, err := runner.Run(ctx, tenantDB, tagMigration)
	if err != nil || completed {
		t.Fatalf("first batch completed=%t, err=%v", completed, err)
	}
	completed, err = runner.RunToCompletion(ctx, tenantDB, tagMigration)
	if err != nil || !completed {
		t.Fatalf("resumed backfill completed=%t, err=%v", completed, err)
	}
	for _, migration := range NewMetadataPublicIDMigrations()[1:] {
		if completed, err := runner.RunToCompletion(ctx, tenantDB, migration); err != nil || !completed {
			t.Fatalf("migration %s completed=%t, err=%v", migration.Key(), completed, err)
		}
	}

	if count := tenantDB.ReadOnlyConn.Tag.Query().Where(tag.PublicIDIsNil()).CountX(ctx); count != 0 {
		t.Fatalf("%d Tags remain without public IDs", count)
	}
	createdDuringUpgrade = tenantDB.ReadOnlyConn.Tag.GetX(ctx, createdDuringUpgrade.ID)
	if createdDuringUpgrade.PublicID != originalPublicID {
		t.Fatal("backfill changed a public ID assigned during upgrade")
	}
	if ready, err := MetadataPublicIDsReady(ctx, tenantDB.ReadOnlyConn); err != nil || !ready {
		t.Fatalf("metadata readiness=%t, err=%v", ready, err)
	}
}
