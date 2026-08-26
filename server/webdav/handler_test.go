package webdav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/common/tenantdbs"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	_ "github.com/simpledms/simpledms/db/enttenant/runtime"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	enttenantwebdavresource "github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	credentialmodel "github.com/simpledms/simpledms/model/main/webdavcredential"
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
	"github.com/simpledms/simpledms/util/e"
)

func TestWebDAVQuotaHTTPStatusMapsTenantQuotaToInsufficientStorage(t *testing.T) {
	quotaErr := e.NewHTTPErrorf(
		http.StatusRequestEntityTooLarge,
		"Storage limit reached for this organization. Used: 1 B of 1 B.",
	)
	if got := webDAVHTTPStatus(quotaErr); got != http.StatusInsufficientStorage {
		t.Fatalf("expected quota status %d, got %d", http.StatusInsufficientStorage, got)
	}

	uploadErr := e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Upload is too large.")
	if got := webDAVHTTPStatus(uploadErr); got != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected upload status %d, got %d", http.StatusRequestEntityTooLarge, got)
	}
}

func TestWriteWebDAVTextIncludesAllowOnMethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()

	writeWebDAVText(rr, http.StatusMethodNotAllowed, "Method not allowed")

	if got := rr.Header().Get("Allow"); got != webDAVAllow {
		t.Fatalf("expected Allow %q, got %q", webDAVAllow, got)
	}
}

func TestWebDAVUploadWriteRefreshesResourceHeartbeat(t *testing.T) {
	tenantDB := newWebDAVCleanupTenantDB(t)
	dbs := tenantdbs.NewTenantDBs()
	dbs.Store(7, tenantDB)
	handler := &Handler{tenantDBs: dbs}
	credentialx := &credentialmodel.AuthRecord{
		TenantID: 7,
		PublicID: "cred-public-id",
	}
	ctx := webDAVCleanupTestContext()
	spacex := tenantDB.ReadWriteConn.Space.Create().SetName("A").SaveX(ctx)
	staleAt := time.Now().Add(-2 * time.Hour)
	resource := tenantDB.ReadWriteConn.WebDAVResource.Create().
		SetCredentialPublicID(entx.NewCIText(credentialx.PublicID)).
		SetSpaceID(spacex.ID).
		SetDavPath("/Inbox/scan.pdf").
		SetLastProgressAt(staleAt).
		SaveX(ctx)

	pipeReader, pipeWriter := io.Pipe()
	copyDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, pipeReader)
		copyDone <- err
	}()
	uploadFile := &webDAVUploadFile{
		requestCtx: &webDAVRequestContext{
			Context:       context.Background(),
			handler:       handler,
			credential:    credentialx,
			spacePublicID: spacex.PublicID.String(),
		},
		resourceID: resource.ID,
		pipeReader: pipeReader,
		pipeWriter: pipeWriter,
	}

	if _, err := uploadFile.Write([]byte("first")); err != nil {
		t.Fatalf("write first chunk: %v", err)
	}
	firstBeat := tenantDB.ReadOnlyConn.WebDAVResource.GetX(ctx, resource.ID).LastProgressAt
	if !firstBeat.After(staleAt) {
		t.Fatalf("resource heartbeat was not refreshed: %v", firstBeat)
	}
	if stale := tenantDB.ReadOnlyConn.WebDAVResource.Query().
		Where(
			enttenantwebdavresource.ID(resource.ID),
			enttenantwebdavresource.StateEQ(webdavresourcemodel.Uploading),
			enttenantwebdavresource.LastProgressAtLT(time.Now().Add(-time.Hour)),
		).
		ExistX(ctx); stale {
		t.Fatal("progressing upload remained eligible for scheduler recovery")
	}

	if _, err := uploadFile.Write([]byte("second")); err != nil {
		t.Fatalf("write second chunk: %v", err)
	}
	secondBeat := tenantDB.ReadOnlyConn.WebDAVResource.GetX(ctx, resource.ID).LastProgressAt
	if !secondBeat.Equal(firstBeat) {
		t.Fatalf("heartbeat was not throttled: first %v, second %v", firstBeat, secondBeat)
	}
	if err := pipeWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-copyDone; err != nil {
		t.Fatal(err)
	}
}

func TestCleanupFailedWebDAVUploadBypassesRevokedAuthorizationButStaysScoped(t *testing.T) {
	tenantDB := newWebDAVCleanupTenantDB(t)
	dbs := tenantdbs.NewTenantDBs()
	dbs.Store(7, tenantDB)
	handler := &Handler{tenantDBs: dbs}
	revokedAt := time.Now()
	credentialx := &credentialmodel.AuthRecord{
		TenantID:  7,
		PublicID:  "cred-public-id",
		RevokedAt: &revokedAt,
	}
	ctx := webDAVCleanupTestContext()
	spaceA := tenantDB.ReadWriteConn.Space.Create().SetName("A").SaveX(ctx)
	spaceB := tenantDB.ReadWriteConn.Space.Create().SetName("B").SaveX(ctx)
	storedFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename("scan.pdf").
		SetSizeInStorage(1).
		SetStorageType(storagetype.Local).
		SetStoragePath("final").
		SetStorageFilename("scan.pdf").
		SetTemporaryStoragePath("tmp").
		SetTemporaryStorageFilename("scan.tmp").
		SaveX(ctx)
	resource := tenantDB.ReadWriteConn.WebDAVResource.Create().
		SetCredentialPublicID(entx.NewCIText(credentialx.PublicID)).
		SetSpaceID(spaceA.ID).
		SetDavPath("/Inbox/scan.pdf").
		SetStoredFileID(storedFile.ID).
		SaveX(ctx)

	if err := handler.cleanupFailedWebDAVUpload(
		context.Background(),
		credentialx,
		spaceB.PublicID.String(),
		resource.ID,
		storedFile.ID,
	); err != nil {
		t.Fatalf("wrong-space cleanup failed: %v", err)
	}
	if count := tenantDB.ReadOnlyConn.WebDAVResource.Query().Where(enttenantwebdavresource.ID(resource.ID)).CountX(ctx); count != 1 {
		t.Fatalf("wrong-space cleanup deleted resource count %d", count)
	}

	if err := handler.cleanupFailedWebDAVUpload(
		context.Background(),
		credentialx,
		spaceA.PublicID.String(),
		resource.ID,
		storedFile.ID,
	); err != nil {
		t.Fatalf("scoped cleanup failed: %v", err)
	}
	if count := tenantDB.ReadOnlyConn.WebDAVResource.Query().Where(enttenantwebdavresource.ID(resource.ID)).CountX(ctx); count != 0 {
		t.Fatalf("resource was not deleted, count %d", count)
	}
	if _, err := tenantDB.ReadOnlyConn.StoredFile.Get(ctx, storedFile.ID); err == nil {
		t.Fatal("stored file was not deleted")
	}
}

func newWebDAVCleanupTenantDB(t *testing.T) *sqlx.TenantDB {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "tenant.sqlite3") + "?_fk=1"
	tenantDB := sqlx.NewTenantDB(dsn, dsn)
	t.Cleanup(func() {
		if err := tenantDB.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if err := tenantDB.ReadWriteConn.Schema.Create(webDAVCleanupTestContext()); err != nil {
		t.Fatalf("create tenant schema: %v", err)
	}
	return tenantDB
}

func webDAVCleanupTestContext() context.Context {
	return tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(context.Background()),
		tenantprivacy.Allow,
	)
}
