package scheduler

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/country"
	"github.com/simpledms/simpledms/model/main/common/plan"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	"github.com/simpledms/simpledms/model/tenant"
	"github.com/simpledms/simpledms/util/ocrutil"
)

func TestApplyOCROneFileSkipsTooLargeFile(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	now := time.Now()
	currentVersion := model.NewStoredFile(&enttenant.StoredFile{
		Size:                       (1024 * 1024) + 1,
		CopiedToFinalDestinationAt: &now,
	})

	content, fileNotReady, fileTooLarge, err := (&Scheduler{}).applyOCROneFile(
		context.Background(),
		nil,
		currentVersion,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fileNotReady {
		t.Fatalf("expected fileNotReady to be false")
	}
	if !fileTooLarge {
		t.Fatalf("expected fileTooLarge to be true")
	}
	if content != "" {
		t.Fatalf("expected empty OCR content for too-large file")
	}
}

func TestApplyOCROneFileReturnsNotReadyForUnmovedFile(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	currentVersion := model.NewStoredFile(&enttenant.StoredFile{
		Size: 1024,
	})

	content, fileNotReady, fileTooLarge, err := (&Scheduler{}).applyOCROneFile(
		context.Background(),
		nil,
		currentVersion,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !fileNotReady {
		t.Fatalf("expected fileNotReady to be true")
	}
	if fileTooLarge {
		t.Fatalf("expected fileTooLarge to be false")
	}
	if content != "" {
		t.Fatalf("expected empty OCR content for not-ready file")
	}
}

func TestApplyOCRxContinuesAfterNotReadyCurrentVersion(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	mainDB := newTestMainDB(t)
	tenantDB := newTestTenantDB(t)
	tenantDBs := tenantdbs.NewTenantDBs()

	now := time.Now()
	tenantID := int64(1)
	createTestTenant(t, mainDB, tenantID, now)
	tenantDBs.Store(tenantID, tenantDB)

	space := tenantDB.ReadWriteConn.Space.Create().
		SetID(1).
		SetName("Test Space").
		SaveX(ctx)

	notReadyFile := tenantDB.ReadWriteConn.File.Create().
		SetID(1).
		SetSpaceID(space.ID).
		SetName("not-ready.pdf").
		SetIsDirectory(false).
		SetIndexedAt(now).
		SaveX(ctx)

	oldCopiedVersion := tenantDB.ReadWriteConn.StoredFile.Create().
		SetID(1).
		SetFilename("old-ready.pdf").
		SetSize(128).
		SetSizeInStorage(128).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("old-ready.pdf").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("old-ready.pdf").
		SetUploadSucceededAt(now).
		SetCopiedToFinalDestinationAt(now.Add(-1 * time.Minute)).
		SaveX(ctx)

	currentNotReadyVersion := tenantDB.ReadWriteConn.StoredFile.Create().
		SetID(2).
		SetFilename("current-not-ready.pdf").
		SetSize(128).
		SetSizeInStorage(128).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("current-not-ready.pdf").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("current-not-ready.pdf").
		SetUploadSucceededAt(now).
		SaveX(ctx)

	tenantDB.ReadWriteConn.FileVersion.Create().
		SetID(1).
		SetFileID(notReadyFile.ID).
		SetStoredFileID(oldCopiedVersion.ID).
		SetVersionNumber(1).
		SaveX(ctx)
	tenantDB.ReadWriteConn.FileVersion.Create().
		SetID(2).
		SetFileID(notReadyFile.ID).
		SetStoredFileID(currentNotReadyVersion.ID).
		SetVersionNumber(2).
		SaveX(ctx)

	tooLargeFile := tenantDB.ReadWriteConn.File.Create().
		SetID(2).
		SetSpaceID(space.ID).
		SetName("too-large.pdf").
		SetIsDirectory(false).
		SetIndexedAt(now).
		SaveX(ctx)

	tooLargeVersion := tenantDB.ReadWriteConn.StoredFile.Create().
		SetID(3).
		SetFilename("too-large.pdf").
		SetSize((1024 * 1024) + 1).
		SetSizeInStorage((1024 * 1024) + 1).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("too-large.pdf").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("too-large.pdf").
		SetUploadSucceededAt(now).
		SetCopiedToFinalDestinationAt(now.Add(-1 * time.Minute)).
		SaveX(ctx)

	tenantDB.ReadWriteConn.FileVersion.Create().
		SetID(3).
		SetFileID(tooLargeFile.ID).
		SetStoredFileID(tooLargeVersion.ID).
		SetVersionNumber(1).
		SaveX(ctx)

	qq := &Scheduler{
		mainDB:    mainDB,
		tenantDBs: tenantDBs,
	}

	qq.applyOCRx(ctx)

	notReadyFile = tenantDB.ReadWriteConn.File.GetX(ctx, notReadyFile.ID)
	if notReadyFile.OcrRetryCount != 0 {
		t.Fatalf("expected not-ready file retry count to stay 0, got %d", notReadyFile.OcrRetryCount)
	}
	if !notReadyFile.OcrLastTriedAt.IsZero() {
		t.Fatal("expected not-ready file last tried time to stay zero")
	}

	tooLargeFile = tenantDB.ReadWriteConn.File.GetX(ctx, tooLargeFile.ID)
	if tooLargeFile.OcrRetryCount != 3 {
		t.Fatalf("expected too-large file retry count to be 3, got %d", tooLargeFile.OcrRetryCount)
	}
	if tooLargeFile.OcrLastTriedAt.IsZero() {
		t.Fatal("expected too-large file last tried time to be set")
	}
	if tooLargeFile.OcrContent != "" {
		t.Fatal("expected too-large file OCR content to stay empty")
	}
}

func createTestTenant(t *testing.T, mainDB *sqlx.MainDB, tenantID int64, now time.Time) {
	t.Helper()

	mainDB.ReadWriteConn.Tenant.Create().
		SetID(tenantID).
		SetName("Test Tenant").
		SetCountry(country.Unknown).
		SetPlan(plan.Unknown).
		SetTermsOfServiceAccepted(now).
		SetPrivacyPolicyAccepted(now).
		SaveX(context.Background())
}
