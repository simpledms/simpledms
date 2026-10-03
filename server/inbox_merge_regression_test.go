package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestInboxMergeRegressionKeepsTargetNameWhenIncomingNameCollides(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "inbox-merge-regression")
	var targetID, sourceID, thirdID int64
	var targetPublicID, sourcePublicID string
	var sourceStoredFileID, tagID, propertyID int64
	var documentTypeID int64
	const sourceNoteTitle = "Source history"
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		root := sc.SpaceRootDir()
		documentType := tc.TTx.DocumentType.Create().SetSpaceID(sp.ID).SetName("Target type").SaveX(sc)
		documentTypeID = documentType.ID
		target := createRegularFileForTest(sc, root.ID, "target.txt").Data
		target = target.Update().SetDocumentTypeID(documentType.ID).SaveX(sc)
		targetPublicID, targetID = target.PublicID.String(), target.ID
		if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{target}); err != nil {
			return err
		}
		third := createRegularFileForTest(sc, root.ID, "incoming.txt").Data
		thirdID = third.ID
		if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{third}); err != nil {
			return err
		}
		source := createRegularFileForTest(sc, root.ID, "inbox-source.tmp").Data.Update().
			SetName("incoming.txt").SetIsInInbox(true).SetOcrContent("incoming OCR").SaveX(sc)
		sourcePublicID, sourceID = source.PublicID.String(), source.ID
		if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{source}); err != nil {
			return err
		}
		sourceStoredFileID = latestFileVersion(sc, source.ID).StoredFileID
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
			sc, source.PublicID.String(), sourceNoteTitle, "transferred note",
		); err != nil {
			return err
		}
		tagID, propertyID = tag.ID, property.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"TargetFileID":   {targetPublicID},
		"SourceFileID":   {sourcePublicID},
		"ConfirmWarning": {"true"},
	}
	rr := f.browserAt(route.Inbox(f.tenant.PublicID.String(), f.spaceID, sourcePublicID),
		h.actions.Browse.FileVersionFromInboxCmd.Endpoint(), form)
	if rr.Code != http.StatusOK {
		t.Fatalf("merge status %d", rr.Code)
	}
	wantTrigger := fmt.Sprintf("%s, %s", event.FileVersionMerged.String(), event.CloseDialog.String())
	if rr.Header().Get("HX-Trigger") != wantTrigger || rr.Header().Get("HX-Reswap") != "none" {
		t.Fatalf("unexpected merge response headers: trigger=%q reswap=%q",
			rr.Header().Get("HX-Trigger"), rr.Header().Get("HX-Reswap"))
	}

	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sp := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, sp)
		target := tc.TTx.File.GetX(sc, targetID)
		if target.Name != "target.txt" || target.ParentID != sc.SpaceRootDir().ID ||
			target.DocumentTypeID != documentTypeID {
			t.Fatalf("target identity/location changed: name=%q parent=%d", target.Name, target.ParentID)
		}
		latest := latestFileVersion(sc, targetID)
		if latest.StoredFileID != sourceStoredFileID {
			t.Fatalf("expected source stored file %d, got %d", sourceStoredFileID, latest.StoredFileID)
		}
		if tc.TTx.File.Query().Where(file.ID(sourceID)).ExistX(schema.SkipSoftDelete(sc)) {
			t.Fatal("merged inbox source still exists")
		}
		if tc.TTx.File.Query().Where(file.ID(thirdID)).OnlyX(sc).Name != "incoming.txt" {
			t.Fatal("third filed document was changed")
		}
		if tc.TTx.FileVersion.Query().Where(fileversion.FileID(sourceID)).CountX(sc) != 0 {
			t.Fatal("source versions were not removed")
		}
		if tc.TTx.TagAssignment.Query().Where(
			tagassignment.FileID(sourceID), tagassignment.TagID(tagID),
		).CountX(sc) != 0 {
			t.Fatal("source tag assignment was not removed")
		}
		if tc.TTx.FilePropertyAssignment.Query().Where(
			filepropertyassignment.FileID(sourceID),
			filepropertyassignment.PropertyID(propertyID),
		).CountX(sc) != 0 {
			t.Fatal("source field assignment was not removed")
		}
		if tc.TTx.TagAssignment.Query().Where(
			tagassignment.FileID(targetID), tagassignment.TagID(tagID),
		).CountX(sc) != 1 {
			t.Fatal("target tag assignment was removed")
		}
		targetProperty := tc.TTx.FilePropertyAssignment.Query().Where(
			filepropertyassignment.FileID(targetID),
			filepropertyassignment.PropertyID(propertyID),
		).OnlyX(sc)
		if targetProperty.TextValue != "target value" {
			t.Fatalf("target field value changed to %q", targetProperty.TextValue)
		}
		if !tc.TTx.DocumentNote.Query().Where(
			documentnote.FileID(targetID), documentnote.Title(sourceNoteTitle),
		).ExistX(sc) {
			t.Fatal("source note history was not transferred")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	list := f.browserAt(
		route.BrowseFile(f.tenant.PublicID.String(), f.spaceID, f.rootID, targetPublicID),
		h.actions.Browse.ListDirPartial.Endpoint(), url.Values{
			"CurrentDirID": {f.rootID}, "SelectedFileID": {targetPublicID},
		})
	if list.Code != http.StatusOK {
		t.Fatalf("directory query status %d", list.Code)
	}
	if !strings.Contains(list.Body.String(), `id="listDirWrapper"`) ||
		!strings.Contains(list.Body.String(), targetPublicID) {
		t.Fatal("directory refresh did not render the selected target in its wrapper")
	}
}
