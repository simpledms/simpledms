package scheduler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	"github.com/marcobeierer/go-tika"

	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/internal/xberg"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	"github.com/simpledms/simpledms/util/ocrutil"
)

const testOCRMarkdown = "# Products\n\n| Product | Units |\n| --- | --- |\n| Basic | 1204 |"

func TestSaveOCRMarkdownResultKeepsTikaSearchContentIfTikaIsConfigured(t *testing.T) {
	ctx, tenantDB := newOCRMarkdownTestDB(t)
	filex := createOCRMarkdownTestFile(t, ctx, tenantDB, 1, 128, func(create *enttenant.FileCreate) {
		create.SetOcrContent("tika text.").SetOcrSuccessAt(time.Now())
	})
	qq := &Scheduler{tikaClientNilable: tika.NewClient(nil, "http://tika.invalid")}

	if !qq.saveOCRMarkdownResult(ctx, tenantDB, filex, testOCRMarkdown, nil) {
		t.Fatal("expected processing to continue")
	}

	filex = tenantDB.ReadOnlyConn.File.GetX(ctx, filex.ID)
	if filex.OcrMarkdown != testOCRMarkdown || filex.OcrMarkdownSuccessAt == nil {
		t.Fatalf("expected markdown to be stored, got %q", filex.OcrMarkdown)
	}
	if filex.OcrContent != "tika text." {
		t.Fatalf("expected Tika search content to be kept, got %q", filex.OcrContent)
	}
}

func TestSaveOCRMarkdownResultFillsMissingSearchContentWithoutTika(t *testing.T) {
	ctx, tenantDB := newOCRMarkdownTestDB(t)
	withoutOCR := createOCRMarkdownTestFile(t, ctx, tenantDB, 1, 128, nil)
	withTikaOCR := createOCRMarkdownTestFile(t, ctx, tenantDB, 2, 128,
		func(create *enttenant.FileCreate) {
			create.SetOcrContent("earlier tika text.").SetOcrSuccessAt(time.Now())
		},
	)
	qq := &Scheduler{}

	qq.saveOCRMarkdownResult(ctx, tenantDB, withoutOCR, testOCRMarkdown, nil)
	qq.saveOCRMarkdownResult(ctx, tenantDB, withTikaOCR, testOCRMarkdown, nil)

	withoutOCR = tenantDB.ReadOnlyConn.File.GetX(ctx, withoutOCR.ID)
	wantContent := "# Products. | Product | Units | | --- | --- | | Basic | 1204 |"
	if withoutOCR.OcrContent != wantContent || withoutOCR.OcrSuccessAt == nil {
		t.Fatalf("expected flattened markdown as search content, got %q", withoutOCR.OcrContent)
	}
	if withoutOCR.OcrMarkdown != testOCRMarkdown {
		t.Fatalf("expected markdown with line breaks, got %q", withoutOCR.OcrMarkdown)
	}
	if !withoutOCR.IsOcrContentFromMarkdown {
		t.Fatal("expected search content to be marked as derived from markdown")
	}
	withTikaOCR = tenantDB.ReadOnlyConn.File.GetX(ctx, withTikaOCR.ID)
	if withTikaOCR.OcrContent != "earlier tika text." {
		t.Fatalf("expected earlier Tika content to be kept, got %q", withTikaOCR.OcrContent)
	}
}

func TestSaveOCRMarkdownResultMarksUnsupportedFormatsAsDone(t *testing.T) {
	ctx, tenantDB := newOCRMarkdownTestDB(t)
	filex := createOCRMarkdownTestFile(t, ctx, tenantDB, 1, 128, nil)
	unsupportedErr := fmt.Errorf("%w: chemical/x-xyz", xberg.ErrUnsupportedFormat)

	if !(&Scheduler{}).saveOCRMarkdownResult(ctx, tenantDB, filex, "", unsupportedErr) {
		t.Fatal("expected processing to continue")
	}

	filex = tenantDB.ReadOnlyConn.File.GetX(ctx, filex.ID)
	if filex.OcrMarkdownSuccessAt == nil || filex.OcrMarkdown != "" {
		t.Fatalf("expected empty markdown marked as done, got %+v", filex)
	}
}

func TestSaveOCRMarkdownResultRetriesOtherFailures(t *testing.T) {
	ctx, tenantDB := newOCRMarkdownTestDB(t)
	filex := createOCRMarkdownTestFile(t, ctx, tenantDB, 1, 128, nil)

	if (&Scheduler{}).saveOCRMarkdownResult(ctx, tenantDB, filex, "", errors.New("xberg down")) {
		t.Fatal("expected processing to stop after a failure")
	}

	filex = tenantDB.ReadOnlyConn.File.GetX(ctx, filex.ID)
	if filex.OcrMarkdownSuccessAt != nil || filex.OcrMarkdownRetryCount != 1 ||
		filex.OcrMarkdownLastTriedAt == nil {
		t.Fatalf("expected a counted retry, got %+v", filex)
	}
	if filex.OcrSuccessAt != nil {
		t.Fatal("expected search content to stay pending")
	}
}

func TestApplyOCRMarkdownxProcessesFilesCreatedBeforeMarkdownColumns(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	ctx, tenantDB := newOCRMarkdownTestDB(t)
	mainDB := newTestMainDB(t)
	tenantDBs := tenantdbs.NewTenantDBs()
	createTestTenant(t, mainDB, 1, time.Now())
	tenantDBs.Store(1, tenantDB)
	// the markdown columns are NULL, as for files created before they were added
	tooLarge := createOCRMarkdownTestFile(t, ctx, tenantDB, 1, (1024*1024)+1, nil)

	(&Scheduler{mainDB: mainDB, tenantDBs: tenantDBs}).applyOCRMarkdownx(ctx)

	tooLarge = tenantDB.ReadOnlyConn.File.GetX(ctx, tooLarge.ID)
	if tooLarge.OcrMarkdownRetryCount != 3 || tooLarge.OcrMarkdownLastTriedAt == nil {
		t.Fatalf("expected too-large file to be marked as done, got %+v", tooLarge)
	}
}

func TestApplyOCRxReprocessesSearchContentDerivedFromMarkdown(t *testing.T) {
	t.Setenv(ocrutil.MaxFileSizeMiBEnvVar, "1")

	ctx, tenantDB := newOCRMarkdownTestDB(t)
	mainDB := newTestMainDB(t)
	tenantDBs := tenantdbs.NewTenantDBs()
	createTestTenant(t, mainDB, 1, time.Now())
	tenantDBs.Store(1, tenantDB)
	// too large, so that processing a selected file only marks it as done
	fromMarkdown := createOCRMarkdownTestFile(t, ctx, tenantDB, 1, (1024*1024)+1,
		func(create *enttenant.FileCreate) {
			create.SetOcrContent("xberg text.").SetOcrSuccessAt(time.Now()).
				SetIsOcrContentFromMarkdown(true)
		},
	)
	fromTika := createOCRMarkdownTestFile(t, ctx, tenantDB, 2, (1024*1024)+1,
		func(create *enttenant.FileCreate) {
			create.SetOcrContent("tika text.").SetOcrSuccessAt(time.Now())
		},
	)

	(&Scheduler{mainDB: mainDB, tenantDBs: tenantDBs}).applyOCRx(ctx)

	if tenantDB.ReadOnlyConn.File.GetX(ctx, fromMarkdown.ID).OcrRetryCount != 3 {
		t.Fatal("expected Tika to process search content derived from markdown")
	}
	if tenantDB.ReadOnlyConn.File.GetX(ctx, fromTika.ID).OcrRetryCount != 0 {
		t.Fatal("expected Tika to skip files it already processed")
	}
}

func newOCRMarkdownTestDB(t *testing.T) (context.Context, *sqlx.TenantDB) {
	t.Helper()

	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	tenantDB := newTestTenantDB(t)
	tenantDB.ReadWriteConn.Space.Create().
		SetID(1).
		SetName("Test Space").
		SaveX(ctx)
	return ctx, tenantDB
}

func createOCRMarkdownTestFile(
	t *testing.T,
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	id int64,
	size int64,
	configure func(*enttenant.FileCreate),
) *enttenant.File {
	t.Helper()

	now := time.Now()
	name := fmt.Sprintf("file-%d.pdf", id)
	create := tenantDB.ReadWriteConn.File.Create().
		SetID(id).
		SetSpaceID(1).
		SetName(name).
		SetIsDirectory(false).
		SetIndexedAt(now)
	if configure != nil {
		configure(create)
	}
	filex := create.SaveX(ctx)

	storedFile := tenantDB.ReadWriteConn.StoredFile.Create().
		SetID(id).
		SetFilename(name).
		SetSize(size).
		SetSizeInStorage(size).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename(name).
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename(name).
		SetUploadSucceededAt(now).
		SetCopiedToFinalDestinationAt(now.Add(-1 * time.Minute)).
		SaveX(ctx)
	tenantDB.ReadWriteConn.FileVersion.Create().
		SetID(id).
		SetFileID(filex.ID).
		SetStoredFileID(storedFile.ID).
		SetVersionNumber(1).
		SaveX(ctx)
	return filex
}
