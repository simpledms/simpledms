package server

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/space"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestDocumentNotesUploadFileVersionPreservesNotes(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
		account, tenant := signUpAccount(t, harness, "notes-versions@example.com")
		tenantDB := initTenantDB(t, harness, tenant)
		tenant = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)
		notes := filemodel.NewDocumentNotes()
		var doc *enttenant.File
		var spacex *enttenant.Space
		var before []*enttenant.DocumentNote
		var versionCount, versionNumber int
		const legacy = "Untouched legacy\n  attribution unknown"

		err := withTenantContext(t, harness, account, tenant, tenantDB, func(
			_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
		) error {
			createSpaceViaCmd(t, harness.actions, tenantCtx, "Notes versions")
			spacex = tenantCtx.TTx.Space.Query().Where(space.Name("Notes versions")).OnlyX(tenantCtx)
			ctx := ctxx.NewSpaceContext(tenantCtx, spacex)
			doc = uploadSpaceFile(
				t, harness, ctx, ctx.SpaceRootDir().ID, "original.pdf", []byte("original"), false,
			)
			doc = doc.Update().SetNotes(legacy).SaveX(ctx)
			current, err := notes.Create(ctx, doc.PublicID.String(), "Current title", "current")
			if err != nil {
				return err
			}
			if _, err := notes.Edit(ctx, doc.PublicID.String(), current.PublicID.String(),
				"Edited title", "edited current\nsecond line"); err != nil {
				return err
			}
			deleted, err := notes.Create(ctx, doc.PublicID.String(), "Deleted title", "deleted text")
			if err != nil {
				return err
			}
			if _, err := notes.Delete(ctx, doc.PublicID.String(), deleted.PublicID.String()); err != nil {
				return err
			}
			original, err := notes.Create(ctx, doc.PublicID.String(), "Original title", "replaced text")
			if err != nil {
				return err
			}
			if _, err := notes.Replace(ctx, doc.PublicID.String(), original.PublicID.String(),
				"Replacement title", "replacement text"); err != nil {
				return err
			}
			_, before, err = notes.List(ctx, doc.PublicID.String(), true)
			if err != nil {
				return err
			}
			if len(before) != 4 {
				t.Fatalf("expected four persisted note fixtures, got %d", len(before))
			}
			versionCount = ctx.TTx.FileVersion.Query().
				Where(fileversion.FileID(doc.ID)).CountX(ctx)
			versionNumber = latestFileVersion(ctx, doc.ID).VersionNumber
			return nil
		})
		if err != nil {
			t.Fatalf("prepare notes and original version: %v", err)
		}

		// This command owns its write transactions, so fixtures must be committed and
		// its request context read-only, just as for ordinary browser uploads.
		mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(
			harness, account, tenant, tenantDB,
		)
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer mainTx.Rollback()
			defer tenantTx.Rollback()
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if err := writer.WriteField("FileID", doc.PublicID.String()); err != nil {
				t.Fatal(err)
			}
			part, err := writer.CreateFormFile("File", "new-version.pdf")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write([]byte("new version content")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/-/browse/upload-file-version-cmd", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			err = harness.actions.Browse.UploadFileVersionCmd.Handler(
				httpx.NewResponseWriter(httptest.NewRecorder()),
				httpx.NewRequest(req),
				ctxx.NewSpaceContext(tenantCtx, spacex),
			)
		}()
		if err != nil {
			t.Fatalf("upload ordinary file version: %v", err)
		}

		err = withTenantContext(t, harness, account, tenant, tenantDB, func(
			_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
		) error {
			ctx := ctxx.NewSpaceContext(tenantCtx, spacex)
			afterDoc, after, err := notes.List(ctx, doc.PublicID.String(), true)
			if err != nil {
				return err
			}
			if afterDoc.ID != doc.ID || afterDoc.Notes != legacy {
				t.Fatalf("document identity or legacy notes changed: %v", afterDoc)
			}
			count := ctx.TTx.FileVersion.Query().Where(fileversion.FileID(doc.ID)).CountX(ctx)
			if count != versionCount+1 {
				t.Fatalf("expected %d versions, got %d", versionCount+1, count)
			}
			if latest := latestFileVersion(ctx, doc.ID); latest.VersionNumber != versionNumber+1 {
				t.Fatalf("expected version number %d, got %d", versionNumber+1, latest.VersionNumber)
			}
			if len(after) != len(before) {
				t.Fatalf("note count changed: %d -> %d", len(before), len(after))
			}
			for i, old := range before {
				got := after[i]
				if got.ID != old.ID || got.PublicID != old.PublicID || got.FileID != old.FileID ||
					got.SpaceID != old.SpaceID || got.Body != old.Body || got.AuthorID != old.AuthorID ||
					got.EditorID != old.EditorID || got.ReplacedByID != old.ReplacedByID ||
					got.Title != old.Title ||
					!reflect.DeepEqual(got.AuthoredAt, old.AuthoredAt) ||
					!reflect.DeepEqual(got.EditedAt, old.EditedAt) ||
					!reflect.DeepEqual(got.DeletedAt, old.DeletedAt) {
					t.Errorf("note identity/content/attribution/history changed: %v -> %v", old, got)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("verify committed notes and new version: %v", err)
		}
	})
}
