package scheduler

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/core/db/entmain"
	entmaintest "github.com/simpledms/simpledms/core/db/entmain/enttest"
	"github.com/simpledms/simpledms/core/db/entx"

	"github.com/simpledms/simpledms/common/tenantdbs"
	sqlx2 "github.com/simpledms/simpledms/core/db/sqlx"
	"github.com/simpledms/simpledms/core/model/common/language"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	"github.com/simpledms/simpledms/core/model/common/storagetype"
	"github.com/simpledms/simpledms/core/util/accountutil"
	"github.com/simpledms/simpledms/db/enttenant"
	enttenanttest "github.com/simpledms/simpledms/db/enttenant/enttest"
	"github.com/simpledms/simpledms/db/sqlx"
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
	filesToDelete := qq.processedTempFilesToDelete(ctx, tenantDB, deletionThreshold)

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

	filesToDelete := qq.tempAccountFilesToDelete(ctx, time.Now())
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

func newTestMainDB(t *testing.T) *sqlx2.MainDB {
	t.Helper()

	client := entmaintest.Open(t, "sqlite3", "file:scheduler-main-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close main db: %v", err)
		}
	})

	return &sqlx2.MainDB{
		DB: &sqlx2.DB[*entmain.Client, *entmain.Tx]{
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
		DB: &sqlx2.DB[*enttenant.Client, *enttenant.Tx]{
			ReadOnlyConn:  client,
			ReadWriteConn: client,
		},
	}
}

func createTestAccount(t *testing.T, mainDB *sqlx2.MainDB) *entmain.Account {
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
