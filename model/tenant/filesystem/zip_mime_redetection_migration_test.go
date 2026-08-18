package filesystem

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	"filippo.io/age"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/enttest"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
)

func TestZIPMIMERedetectionMigrationProcessesLegacyFiles(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	client := enttest.Open(t, "sqlite3", "file:zip-migration?mode=memory&cache=shared&_fk=1")
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	tenantDB := &sqlx.TenantDB{DB: &sqlx.DB[*enttenant.Client, *enttenant.Tx]{
		ReadOnlyConn: client, ReadWriteConn: client,
	}}
	firstStart := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	ods := createZIPMigrationFile(t, ctx, client, "sheet.ods", firstStart.Add(-time.Hour))
	archive := createZIPMigrationFile(t, ctx, client, "archive.zip", firstStart.Add(-time.Minute))
	missing := createZIPMigrationFile(t, ctx, client, "missing.zip", firstStart.Add(-time.Minute))
	skipped := createZIPMigrationFile(t, ctx, client, "new.ods", firstStart)
	contents := map[int64][]byte{
		ods.ID:     zipMigrationContent(t, map[string]string{"mimetype": "application/vnd.oasis.opendocument.spreadsheet"}),
		archive.ID: zipMigrationContent(t, map[string]string{"notes.txt": "hello"}),
	}
	openCount := 0
	migration := newZIPMIMERedetectionMigration(func(
		_ context.Context,
		_ *age.X25519Identity,
		filex *storedfilemodel.StoredFile,
	) (io.ReadCloser, error) {
		if filex.Data.ID == missing.ID {
			return nil, errors.New("The specified key does not exist.")
		}
		openCount++
		return io.NopCloser(bytes.NewReader(contents[filex.Data.ID])), nil
	}, nil)
	runner := tenantdatamigration.NewRunnerWithClock(func() time.Time { return firstStart })

	completed, err := runner.Run(ctx, tenantDB, migration)
	if err != nil || !completed {
		t.Fatalf("completed=%t, err=%v", completed, err)
	}
	if openCount != 2 {
		t.Fatalf("opened files %d times, want 2", openCount)
	}
	ods = client.StoredFile.GetX(ctx, ods.ID)
	archive = client.StoredFile.GetX(ctx, archive.ID)
	missing = client.StoredFile.GetX(ctx, missing.ID)
	skipped = client.StoredFile.GetX(ctx, skipped.ID)
	if ods.MimeType != "application/vnd.oasis.opendocument.spreadsheet" {
		t.Fatalf("ODS MIME = %q", ods.MimeType)
	}
	if archive.MimeType != "application/zip" || missing.MimeType != "application/zip" || skipped.MimeType != "application/zip" {
		t.Fatal("genuine or post-cutoff ZIP was changed")
	}
	state := client.TenantDataMigration.Query().OnlyX(ctx)
	if state.CompletedAt == nil || !state.FirstStartedAt.Equal(firstStart) || state.Cursor != missing.ID {
		t.Fatalf("unexpected migration state: %#v", state)
	}
}

func createZIPMigrationFile(
	t *testing.T,
	ctx context.Context,
	client *enttenant.Client,
	filename string,
	createdAt time.Time,
) *enttenant.StoredFile {
	t.Helper()
	now := time.Now()
	filex := client.StoredFile.Create().
		SetCreatedAt(createdAt).
		SetFilename(filename).
		SetSize(10).
		SetSizeInStorage(10).
		SetMimeType("application/zip").
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename(filename).
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename(filename).
		SetCopiedToFinalDestinationAt(now).
		SaveX(ctx)
	client.StoredFile.UpdateOneID(filex.ID).
		ClearUploadStartedAt().
		ExecX(enttenantschema.WithUnfinishedUploads(ctx))
	return filex
}

func zipMigrationContent(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var content bytes.Buffer
	archive := zip.NewWriter(&content)
	for name, value := range entries {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}
