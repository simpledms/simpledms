package server

import (
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

const inboxMergeSourceNoteTitle = "Source history"

type inboxMergeCollisionFixture struct {
	targetID           int64
	sourceID           int64
	thirdID            int64
	targetPublicID     string
	sourcePublicID     string
	sourceStoredFileID int64
	tagID              int64
	propertyID         int64
	documentTypeID     int64
}

func newInboxMergeCollisionFixture(
	t *testing.T,
	h *actionTestHarness,
	f *mcpFixture,
) *inboxMergeCollisionFixture {
	t.Helper()
	fixture := &inboxMergeCollisionFixture{}
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		root := sc.SpaceRootDir()
		documentType := tc.TTx.DocumentType.Create().SetSpaceID(sp.ID).SetName("Target type").SaveX(sc)
		fixture.documentTypeID = documentType.ID
		target := createRegularFileForTest(sc, root.ID, "target.txt").Data
		target = target.Update().SetDocumentTypeID(documentType.ID).SaveX(sc)
		fixture.targetPublicID, fixture.targetID = target.PublicID.String(), target.ID
		if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{target}); err != nil {
			return err
		}
		third := createRegularFileForTest(sc, root.ID, "incoming.txt").Data
		fixture.thirdID = third.ID
		if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{third}); err != nil {
			return err
		}
		source := createRegularFileForTest(sc, root.ID, "inbox-source.tmp").Data.Update().
			SetName("incoming.txt").SetIsInInbox(true).SetOcrContent("incoming OCR").SaveX(sc)
		fixture.sourcePublicID, fixture.sourceID = source.PublicID.String(), source.ID
		if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{source}); err != nil {
			return err
		}
		fixture.sourceStoredFileID = latestFileVersion(sc, source.ID).StoredFileID
		tag := tc.TTx.Tag.Create().SetSpaceID(sp.ID).SetName("Source tag").
			SetType(tagtype.Simple).SaveX(sc)
		property := tc.TTx.Property.Create().SetSpaceID(sp.ID).SetName("Source field").
			SetType(fieldtype.Text).SaveX(sc)
		tc.TTx.TagAssignment.Create().SetSpaceID(sp.ID).SetFileID(source.ID).SetTagID(tag.ID).SaveX(sc)
		tc.TTx.TagAssignment.Create().SetSpaceID(sp.ID).SetFileID(target.ID).SetTagID(tag.ID).SaveX(sc)
		tc.TTx.FilePropertyAssignment.Create().SetSpaceID(sp.ID).SetFileID(source.ID).
			SetPropertyID(property.ID).SetTextValue("source value").SaveX(sc)
		tc.TTx.FilePropertyAssignment.Create().SetSpaceID(sp.ID).SetFileID(target.ID).
			SetPropertyID(property.ID).SetTextValue("target value").SaveX(sc)
		if _, err := filemodel.NewDocumentNotes().Create(
			sc, source.PublicID.String(), inboxMergeSourceNoteTitle, "transferred note",
		); err != nil {
			return err
		}
		fixture.tagID, fixture.propertyID = tag.ID, property.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func (fixture *inboxMergeCollisionFixture) assertPersisted(
	t *testing.T,
	h *actionTestHarness,
	f *mcpFixture,
) {
	t.Helper()
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		fixture.assertTargetRetained(t, sc)
		fixture.assertSourceCleanedUp(t, sc)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func (fixture *inboxMergeCollisionFixture) assertTargetRetained(
	t *testing.T,
	sc *ctxx.SpaceContext,
) {
	t.Helper()
	target := sc.TenantContext.TTx.File.GetX(sc, fixture.targetID)
	if target.Name != "target.txt" || target.ParentID != sc.SpaceRootDir().ID ||
		target.DocumentTypeID != fixture.documentTypeID {
		t.Fatalf("target identity/location changed: name=%q parent=%d", target.Name, target.ParentID)
	}
	latest := latestFileVersion(sc, fixture.targetID)
	if latest.StoredFileID != fixture.sourceStoredFileID {
		t.Fatalf("expected source stored file %d, got %d", fixture.sourceStoredFileID, latest.StoredFileID)
	}
	if sc.TenantContext.TTx.TagAssignment.Query().Where(
		tagassignment.FileID(fixture.targetID), tagassignment.TagID(fixture.tagID),
	).CountX(sc) != 1 {
		t.Fatal("target tag assignment was removed")
	}
	targetProperty := sc.TenantContext.TTx.FilePropertyAssignment.Query().Where(
		filepropertyassignment.FileID(fixture.targetID),
		filepropertyassignment.PropertyID(fixture.propertyID),
	).OnlyX(sc)
	if targetProperty.TextValue != "target value" {
		t.Fatalf("target field value changed to %q", targetProperty.TextValue)
	}
	if !sc.TenantContext.TTx.DocumentNote.Query().Where(
		documentnote.FileID(fixture.targetID), documentnote.Title(inboxMergeSourceNoteTitle),
	).ExistX(sc) {
		t.Fatal("source note history was not transferred")
	}
}

func (fixture *inboxMergeCollisionFixture) assertSourceCleanedUp(
	t *testing.T,
	sc *ctxx.SpaceContext,
) {
	t.Helper()
	if sc.TTx.File.Query().Where(file.ID(fixture.sourceID)).ExistX(schema.SkipSoftDelete(sc)) {
		t.Fatal("merged inbox source still exists")
	}
	if sc.TenantContext.TTx.File.Query().Where(file.ID(fixture.thirdID)).OnlyX(sc).Name != "incoming.txt" {
		t.Fatal("third filed document was changed")
	}
	if sc.TenantContext.TTx.FileVersion.Query().Where(
		fileversion.FileID(fixture.sourceID),
	).CountX(sc) != 0 {
		t.Fatal("source versions were not removed")
	}
	if sc.TenantContext.TTx.TagAssignment.Query().Where(
		tagassignment.FileID(fixture.sourceID), tagassignment.TagID(fixture.tagID),
	).CountX(sc) != 0 {
		t.Fatal("source tag assignment was not removed")
	}
	if sc.TenantContext.TTx.FilePropertyAssignment.Query().Where(
		filepropertyassignment.FileID(fixture.sourceID),
		filepropertyassignment.PropertyID(fixture.propertyID),
	).CountX(sc) != 0 {
		t.Fatal("source field assignment was not removed")
	}
}
