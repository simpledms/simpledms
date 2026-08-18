package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/entmain"
	entmaintest "github.com/simpledms/simpledms/db/entmain/enttest"
	"github.com/simpledms/simpledms/db/enttenant"
	enttenanttest "github.com/simpledms/simpledms/db/enttenant/enttest"
	privacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	previewmodel "github.com/simpledms/simpledms/model/tenant/previewconversion"
	"github.com/simpledms/simpledms/util/accountutil"
)

func TestDeleteProcessedTempFilesDeletesOnlyAfterGracePeriod(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	tenantDBs := tenantdbs.NewTenantDBs()
	tenantDBs.Store(1, tenantDB)

	now := time.Now()
	oldFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename("old.pdf").
		SetSize(10).
		SetSizeInStorage(10).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("old.pdf").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("old.pdf").
		SetUploadSucceededAt(now).
		SetCopiedToFinalDestinationAt(now.Add(-6 * time.Minute)).
		SaveX(ctx)

	recentFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename("recent.pdf").
		SetSize(10).
		SetSizeInStorage(10).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("recent.pdf").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("recent.pdf").
		SetUploadSucceededAt(now).
		SetCopiedToFinalDestinationAt(now.Add(-4 * time.Minute)).
		SaveX(ctx)

	alreadyDeletedAt := now.Add(-2 * time.Minute)
	alreadyDeletedFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename("already-deleted.pdf").
		SetSize(10).
		SetSizeInStorage(10).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("already-deleted.pdf").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("already-deleted.pdf").
		SetUploadSucceededAt(now).
		SetCopiedToFinalDestinationAt(now.Add(-6 * time.Minute)).
		SetDeletedTemporaryFileAt(alreadyDeletedAt).
		SaveX(ctx)

	qq := &Scheduler{
		tenantDBs: tenantDBs,
	}

	deletionThreshold := time.Now().Add(-5 * time.Minute)
	filesToDelete := qq.processedTempFilesToDelete(
		ctx,
		tenantDB,
		deletionThreshold,
		defaultSchedulerBatchSize,
	)

	if len(filesToDelete) != 1 {
		t.Fatalf("expected 1 stored file to be eligible for temp deletion, got %d", len(filesToDelete))
	}
	if filesToDelete[0].ID != oldFile.ID {
		t.Fatalf("expected old file %d to be eligible, got %d", oldFile.ID, filesToDelete[0].ID)
	}

	recentFile = tenantDB.ReadWriteConn.StoredFile.GetX(ctx, recentFile.ID)
	if recentFile.DeletedTemporaryFileAt != nil {
		t.Fatal("expected recent file temp object to stay marked as not deleted")
	}

	alreadyDeletedFile = tenantDB.ReadWriteConn.StoredFile.GetX(ctx, alreadyDeletedFile.ID)
	if alreadyDeletedFile.DeletedTemporaryFileAt == nil || !alreadyDeletedFile.DeletedTemporaryFileAt.Equal(alreadyDeletedAt) {
		t.Fatal("expected already deleted file timestamp to stay unchanged")
	}
}

func TestDiscoverPreviewConversionsIncludesLegacyFinalFiles(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	space := tenantDB.ReadWriteConn.Space.Create().
		SetID(1).
		SetName("Test Space").
		SaveX(ctx)

	filex := tenantDB.ReadWriteConn.File.Create().
		SetSpaceID(space.ID).
		SetName("legacy.html").
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SaveX(ctx)

	storedFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename("legacy.html").
		SetSize(10).
		SetSizeInStorage(10).
		SetMimeType("text/html").
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("legacy.html").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("legacy.html").
		SetCopiedToFinalDestinationAt(time.Now()).
		SaveX(ctx)

	_, err := tenantDB.ReadWriteConn.ExecContext(
		ctx,
		"UPDATE stored_files SET upload_started_at = NULL, upload_succeeded_at = NULL WHERE id = ?",
		storedFile.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tenantDB.ReadWriteConn.FileVersion.Create().
		SetFileID(filex.ID).
		SetStoredFileID(storedFile.ID).
		SetVersionNumber(1).
		Save(ctx)
	if err != nil {
		t.Fatal(err)
	}

	(&Scheduler{}).discoverPreviewConversions(ctx, tenantDB)

	conversion, err := tenantDB.ReadOnlyConn.PreviewConversion.Query().Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if conversion.SourceStoredFileID != storedFile.ID {
		t.Fatalf("expected conversion for stored file %d, got %d", storedFile.ID, conversion.SourceStoredFileID)
	}
}

func TestDiscoverPreviewConversionsSkipsPastIneligibleBatch(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	space := tenantDB.ReadWriteConn.Space.Create().SetID(1).SetName("Test Space").SaveX(ctx)
	filex := tenantDB.ReadWriteConn.File.Create().
		SetSpaceID(space.ID).
		SetName("versions").
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SaveX(ctx)

	for i := range defaultSchedulerBatchSize {
		createPreviewVersion(t, ctx, tenantDB, filex.ID, i+1, fmt.Sprintf("image-%d.png", i), "image/png")
	}
	eligible := createPreviewVersion(
		t,
		ctx,
		tenantDB,
		filex.ID,
		defaultSchedulerBatchSize+1,
		"README.md",
		"text/plain; charset=utf-8",
	)
	if _, isEligible := previewmodel.Classify(eligible.MimeType, eligible.Filename, false); !isEligible {
		t.Fatal("test source should be eligible")
	}
	scheduler := &Scheduler{}
	scheduler.discoverPreviewConversions(ctx, tenantDB)
	scheduler.discoverPreviewConversions(ctx, tenantDB)

	conversion := tenantDB.ReadOnlyConn.PreviewConversion.Query().OnlyX(ctx)
	if conversion.SourceStoredFileID != eligible.ID {
		t.Fatalf("conversion source = %d, want %d", conversion.SourceStoredFileID, eligible.ID)
	}
}

func TestRecoverInvalidReadyPreviewConversions(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	space := tenantDB.ReadWriteConn.Space.Create().SetID(1).SetName("Test Space").SaveX(ctx)
	source := createPreviewSource(t, ctx, tenantDB, space.ID, "README.md", "text/markdown")

	t.Run("missing artifact is queued again", func(t *testing.T) {
		conversion := tenantDB.ReadWriteConn.PreviewConversion.Create().
			SetSourceID(source.ID).
			SetStatus(previewmodel.Ready).
			SetRetryCount(3).
			SetLastAttemptedAt(time.Now()).
			SaveX(ctx)

		(&Scheduler{}).recoverInvalidReadyPreviewConversions(ctx, tenantDB)

		conversion = tenantDB.ReadOnlyConn.PreviewConversion.GetX(ctx, conversion.ID)
		if conversion.Status != previewmodel.Pending || conversion.PreviewStoredFileID != nil || conversion.RetryCount != 0 {
			t.Fatalf("invalid ready conversion was not queued again: %#v", conversion)
		}
		tenantDB.ReadWriteConn.PreviewConversion.DeleteOneID(conversion.ID).ExecX(ctx)
	})

	t.Run("non-final artifact returns to processing", func(t *testing.T) {
		preview := tenantDB.ReadWriteConn.StoredFile.Create().
			SetFilename("README.pdf").
			SetSize(10).
			SetSizeInStorage(10).
			SetMimeType("application/pdf").
			SetStorageType(storagetype.S3).
			SetStoragePath("tenant/final").
			SetStorageFilename("README.pdf").
			SetTemporaryStoragePath("tenant/tmp").
			SetTemporaryStorageFilename("README.pdf").
			SetUploadSucceededAt(time.Now()).
			SaveX(ctx)
		conversion := tenantDB.ReadWriteConn.PreviewConversion.Create().
			SetSourceID(source.ID).
			SetPreviewID(preview.ID).
			SetStatus(previewmodel.Ready).
			SaveX(ctx)
		if conversion.PreviewStoredFileID == nil {
			t.Fatal("preview conversion was created without its artifact")
		}
		if _, err := tenantDB.ReadOnlyConn.StoredFile.Get(ctx, preview.ID); err != nil {
			t.Fatalf("preview artifact is not readable: %v", err)
		}

		(&Scheduler{}).recoverInvalidReadyPreviewConversions(ctx, tenantDB)

		conversion = tenantDB.ReadOnlyConn.PreviewConversion.GetX(ctx, conversion.ID)
		if conversion.Status != previewmodel.Processing || conversion.PreviewStoredFileID == nil || conversion.ProcessingStartedAt == nil {
			t.Fatalf("invalid ready conversion did not return to processing: %#v", conversion)
		}
	})
}

func createPreviewSource(
	t *testing.T,
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	spaceID int64,
	filename string,
	mimeType string,
) *enttenant.StoredFile {
	t.Helper()
	filex := tenantDB.ReadWriteConn.File.Create().
		SetSpaceID(spaceID).
		SetName(filename).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SaveX(ctx)
	return createPreviewVersion(t, ctx, tenantDB, filex.ID, 1, filename, mimeType)
}

func createPreviewVersion(
	t *testing.T,
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	fileID int64,
	versionNumber int,
	filename string,
	mimeType string,
) *enttenant.StoredFile {
	t.Helper()
	storedFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename(filename).
		SetSize(10).
		SetSizeInStorage(10).
		SetMimeType(mimeType).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename(filename).
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename(filename).
		SetUploadSucceededAt(time.Now()).
		SetCopiedToFinalDestinationAt(time.Now()).
		SaveX(ctx)
	tenantDB.ReadWriteConn.FileVersion.Create().
		SetFileID(fileID).
		SetStoredFileID(storedFile.ID).
		SetVersionNumber(versionNumber).
		SaveX(ctx)
	return storedFile
}

func TestDeleteTempAccountFilesDeletesOnlyExpiredUnconvertedFiles(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	mainDB := newTestMainDB(t)

	owner := createTestAccount(t, mainDB)
	now := time.Now()

	expiredFile := mainDB.ReadWriteConn.TemporaryFile.Create().
		SetOwnerID(owner.ID).
		SetFilename("expired.pdf").
		SetSize(10).
		SetSizeInStorage(10).
		SetStorageType(storagetype.S3).
		SetStoragePath("account/tmp").
		SetStorageFilename("expired.pdf").
		SetUploadToken("expired-token").
		SetUploadSucceededAt(now).
		SetExpiresAt(now.Add(-1 * time.Minute)).
		SaveX(ctx)

	activeFile := mainDB.ReadWriteConn.TemporaryFile.Create().
		SetOwnerID(owner.ID).
		SetFilename("active.pdf").
		SetSize(10).
		SetSizeInStorage(10).
		SetStorageType(storagetype.S3).
		SetStoragePath("account/tmp").
		SetStorageFilename("active.pdf").
		SetUploadToken("active-token").
		SetUploadSucceededAt(now).
		SetExpiresAt(now.Add(1 * time.Minute)).
		SaveX(ctx)

	convertedFile := mainDB.ReadWriteConn.TemporaryFile.Create().
		SetOwnerID(owner.ID).
		SetFilename("converted.pdf").
		SetSize(10).
		SetSizeInStorage(10).
		SetStorageType(storagetype.S3).
		SetStoragePath("account/tmp").
		SetStorageFilename("converted.pdf").
		SetUploadToken("converted-token").
		SetUploadSucceededAt(now).
		SetExpiresAt(now.Add(-1 * time.Minute)).
		SetConvertedToStoredFileAt(now).
		SaveX(ctx)

	qq := &Scheduler{
		mainDB: mainDB,
	}

	filesToDelete := qq.tempAccountFilesToDelete(ctx, time.Now(), defaultSchedulerBatchSize)
	if len(filesToDelete) != 1 {
		t.Fatalf("expected 1 temporary file to be eligible for deletion, got %d", len(filesToDelete))
	}
	if filesToDelete[0].ID != expiredFile.ID {
		t.Fatalf("expected expired temporary file %d to be eligible, got %d", expiredFile.ID, filesToDelete[0].ID)
	}

	activeFile = mainDB.ReadWriteConn.TemporaryFile.GetX(ctx, activeFile.ID)
	if !activeFile.DeletedAt.IsZero() {
		t.Fatal("expected active temporary file to remain undeleted")
	}

	convertedFile = mainDB.ReadWriteConn.TemporaryFile.GetX(ctx, convertedFile.ID)
	if !convertedFile.DeletedAt.IsZero() {
		t.Fatal("expected converted temporary file to remain undeleted")
	}
}

func newTestMainDB(t *testing.T) *sqlx.MainDB {
	t.Helper()

	client := entmaintest.Open(t, "sqlite3", "file:scheduler-main-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close main db: %v", err)
		}
	})

	return &sqlx.MainDB{
		DB: &sqlx.DB[*entmain.Client, *entmain.Tx]{
			ReadOnlyConn:  client,
			ReadWriteConn: client,
		},
	}
}

func newTestTenantDB(t *testing.T) *sqlx.TenantDB {
	t.Helper()

	client := enttenanttest.Open(t, "sqlite3", "file:scheduler-tenant-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close tenant db: %v", err)
		}
	})

	return &sqlx.TenantDB{
		DB: &sqlx.DB[*enttenant.Client, *enttenant.Tx]{
			ReadOnlyConn:  client,
			ReadWriteConn: client,
		},
	}
}

func createTestAccount(t *testing.T, mainDB *sqlx.MainDB) *entmain.Account {
	t.Helper()

	salt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate salt")
	}

	return mainDB.ReadWriteConn.Account.Create().
		SetEmail(entx.NewCIText("scheduler@example.com")).
		SetFirstName("Scheduler").
		SetLastName("Test").
		SetLanguage(language.Unknown).
		SetRole(mainrole.User).
		SetPasswordSalt(salt).
		SetPasswordHash(accountutil.PasswordHash("secret", salt)).
		SaveX(context.Background())
}
