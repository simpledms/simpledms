package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	"filippo.io/age"

	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	previewmodel "github.com/simpledms/simpledms/model/tenant/previewconversion"
)

func TestIdleSchedulerPassesDoNotLoadTenant(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	tenantDBs := tenantdbs.NewTenantDBs()
	tenantDBs.Store(1, tenantDB)
	scheduler := &Scheduler{tenantDBs: tenantDBs}

	// These passes have no eligible rows. A nil main DB makes any eager tenant lookup fail.
	scheduler.applyOCRx(ctx)
	scheduler.applyOCRMarkdownx(ctx)
	scheduler.copyTempFilesToFinalDest(ctx)
	scheduler.convertPreviewsOnce(ctx)
}

func TestPreviewIdleCandidatesDoNotLoadTenant(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	space := tenantDB.ReadWriteConn.Space.Create().SetID(1).SetName("Test").SaveX(ctx)
	source := createPreviewSource(t, ctx, tenantDB, space.ID, "source.md", "text/markdown")
	now := time.Now()
	preview := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename("source.pdf").
		SetSize(1).
		SetSizeInStorage(1).
		SetStorageType(storagetype.S3).
		SetStoragePath("final").
		SetStorageFilename("source.pdf").
		SetTemporaryStoragePath("tmp").
		SetTemporaryStorageFilename("source.pdf").
		SetUploadSucceededAt(now).
		SaveX(ctx)
	conversion := tenantDB.ReadWriteConn.PreviewConversion.Create().
		SetSourceID(source.ID).
		SetPreviewID(preview.ID).
		SetStatus(previewmodel.Processing).
		SaveX(ctx)
	futureSource := createPreviewSource(t, ctx, tenantDB, space.ID, "future.md", "text/markdown")
	future := tenantDB.ReadWriteConn.PreviewConversion.Create().
		SetSourceID(futureSource.ID).
		SetStatus(previewmodel.Pending).
		SetNextAttemptAt(now.Add(time.Hour)).
		SaveX(ctx)

	loads := 0
	loadTenant := func() (*entmain.Tenant, error) {
		loads++
		return nil, errors.New("tenant lookup should be deferred")
	}
	scheduler := &Scheduler{}
	scheduler.reconcilePreviewConversions(ctx, tenantDB, loadTenant)
	scheduler.processDuePreviewConversions(ctx, tenantDB, loadTenant)
	if loads != 0 {
		t.Fatalf("tenant lookup calls = %d, want 0", loads)
	}
	if got := tenantDB.ReadOnlyConn.PreviewConversion.GetX(ctx, conversion.ID).Status; got != previewmodel.Processing {
		t.Fatalf("non-final preview status = %q, want processing", got)
	}
	if got := tenantDB.ReadOnlyConn.PreviewConversion.GetX(ctx, future.ID).Status; got != previewmodel.Pending {
		t.Fatalf("future retry status = %q, want pending", got)
	}
}

func TestDuePreviewTenantLookupFailurePreventsClaim(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	space := tenantDB.ReadWriteConn.Space.Create().SetID(1).SetName("Test").SaveX(ctx)
	source := createPreviewSource(t, ctx, tenantDB, space.ID, "source.md", "text/markdown")
	conversion := tenantDB.ReadWriteConn.PreviewConversion.Create().
		SetSourceID(source.ID).SetStatus(previewmodel.Pending).SaveX(ctx)
	wantErr := errors.New("tenant unavailable")
	loads := 0
	(&Scheduler{}).processDuePreviewConversions(ctx, tenantDB, func() (*entmain.Tenant, error) {
		loads++
		return nil, wantErr
	})
	if loads != 1 {
		t.Fatalf("tenant lookup calls = %d, want 1", loads)
	}
	conversion = tenantDB.ReadOnlyConn.PreviewConversion.GetX(ctx, conversion.ID)
	if conversion.Status != previewmodel.Pending || conversion.ProcessingStartedAt != nil {
		t.Fatalf("conversion was claimed despite tenant lookup failure: %+v", conversion)
	}
}

func TestContentHashBackfillDoesNotLoadIdentityWithoutWork(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	service := filesystem.NewContentHashBackfillService(nil)
	for _, test := range []struct {
		name   string
		create func(*sqlx.TenantDB)
		config filesystem.ContentHashBackfillConfig
	}{
		{name: "empty", config: filesystem.ContentHashBackfillConfig{MaxFilesPerRun: 1}},
		{
			name: "over budget",
			create: func(tenantDB *sqlx.TenantDB) {
				createBackfillCandidate(t, ctx, tenantDB, 1, 2)
			},
			config: filesystem.ContentHashBackfillConfig{MaxFilesPerRun: 1, MaxBytesPerRun: 1},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tenantDB := newTestTenantDB(t)
			if test.create != nil {
				test.create(tenantDB)
			}
			loads := 0
			_, err := service.RunTenant(ctx, tenantDB.ReadOnlyConn, tenantDB.ReadWriteConn,
				func() (*age.X25519Identity, error) {
					loads++
					return nil, errors.New("identity should not be loaded")
				}, test.config)
			if err != nil {
				t.Fatalf("RunTenant() error = %v", err)
			}
			if loads != 0 {
				t.Fatalf("identity loader calls = %d, want 0", loads)
			}
		})
	}
}

func TestContentHashBackfillPropagatesIdentityLoadErrorWhenWorkExists(t *testing.T) {
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	createBackfillCandidate(t, ctx, tenantDB, 1, 2)
	wantErr := errors.New("identity unavailable")
	loads := 0
	_, err := filesystem.NewContentHashBackfillService(nil).RunTenant(
		ctx,
		tenantDB.ReadOnlyConn,
		tenantDB.ReadWriteConn,
		func() (*age.X25519Identity, error) {
			loads++
			return nil, wantErr
		},
		filesystem.ContentHashBackfillConfig{MaxFilesPerRun: 1},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunTenant() error = %v, want %v", err, wantErr)
	}
	if loads != 1 {
		t.Fatalf("identity loader calls = %d, want 1", loads)
	}
}

func createBackfillCandidate(t *testing.T, ctx context.Context, tenantDB *sqlx.TenantDB, id int, size int64) {
	t.Helper()
	tenantDB.ReadWriteConn.StoredFile.Create().
		SetID(int64(id)).
		SetFilename("candidate.pdf").
		SetSize(size).
		SetSizeInStorage(size).
		SetStorageType(storagetype.S3).
		SetStoragePath("final").
		SetStorageFilename("candidate.pdf").
		SetTemporaryStoragePath("tmp").
		SetTemporaryStorageFilename("candidate.pdf").
		SetUploadSucceededAt(time.Now()).
		SetCopiedToFinalDestinationAt(time.Now()).
		SaveX(ctx)
}
