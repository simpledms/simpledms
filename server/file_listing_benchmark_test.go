package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/action"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/model/common/storagetype"
	"github.com/simpledms/simpledms/util/httpx"
)

type listingBenchmarkFixture struct {
	actions         *action.Actions
	spaceCtx        *ctxx.SpaceContext
	spaceID         int64
	rootDirID       int64
	rootDirPublicID string
	cleanup         func()
}

func inboxEventSearchQuery() string {
	return "benchmark-file-000000.txt"
}

func BenchmarkFileListing(b *testing.B) {
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		log.SetOutput(originalLogOutput)
	})

	fileCounts := []int{0, 1000, 5000, 10000, 100000, 1000000}

	for _, fileCount := range fileCounts {
		fileCount := fileCount

		b.Run(fmt.Sprintf("inbox_%d", fileCount), func(b *testing.B) {
			fixture := newListingBenchmarkFixture(b, fileCount, true)
			defer fixture.cleanup()

			expectedRowCount := 0
			if fileCount > 0 {
				expectedRowCount = 1
			}

			rows, err := inboxListingQuery(fixture.spaceCtx, fixture.spaceID, inboxEventSearchQuery())
			if err != nil {
				b.Fatalf("initial inbox listing query failed: %v", err)
			}
			if len(rows) != expectedRowCount {
				b.Fatalf("expected %d inbox rows, got %d", expectedRowCount, len(rows))
			}

			b.Run("query", func(b *testing.B) {
				benchmarkInboxListingQuery(b, fixture, expectedRowCount)
			})
			b.Run("handler", func(b *testing.B) {
				benchmarkInboxListingHandler(b, fixture, fileCount)
			})
		})

		b.Run(fmt.Sprintf("browse_%d", fileCount), func(b *testing.B) {
			fixture := newListingBenchmarkFixture(b, fileCount, false)
			defer fixture.cleanup()

			rows, err := browseListingQuery(fixture.spaceCtx, fixture.spaceID, fixture.rootDirID)
			if err != nil {
				b.Fatalf("initial browse listing query failed: %v", err)
			}

			expectedRowCount := fileCount
			if expectedRowCount > 51 {
				expectedRowCount = 51
			}
			if len(rows) != expectedRowCount {
				b.Fatalf("expected %d browse rows, got %d", expectedRowCount, len(rows))
			}

			b.Run("query", func(b *testing.B) {
				benchmarkBrowseListingQuery(b, fixture, expectedRowCount)
			})
			b.Run("handler", func(b *testing.B) {
				benchmarkBrowseListingHandler(b, fixture, expectedRowCount)
			})
		})
	}
}

func benchmarkInboxListingQuery(b *testing.B, fixture *listingBenchmarkFixture, expectedRowCount int) {
	b.Helper()
	b.ReportAllocs()
	searchQuery := inboxEventSearchQuery()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rows, err := inboxListingQuery(fixture.spaceCtx, fixture.spaceID, searchQuery)
		if err != nil {
			b.Fatalf("inbox listing query failed: %v", err)
		}
		if len(rows) != expectedRowCount {
			b.Fatalf("expected %d inbox rows, got %d", expectedRowCount, len(rows))
		}
	}

	b.StopTimer()
	reportEventTime(b)
}

func benchmarkBrowseListingQuery(b *testing.B, fixture *listingBenchmarkFixture, expectedRowCount int) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rows, err := browseListingQuery(fixture.spaceCtx, fixture.spaceID, fixture.rootDirID)
		if err != nil {
			b.Fatalf("browse listing query failed: %v", err)
		}
		if len(rows) != expectedRowCount {
			b.Fatalf("expected %d browse rows, got %d", expectedRowCount, len(rows))
		}
	}

	b.StopTimer()
	reportEventTime(b)
}

func benchmarkInboxListingHandler(b *testing.B, fixture *listingBenchmarkFixture, fileCount int) {
	b.Helper()
	b.ReportAllocs()

	searchQuery := ""
	if fileCount > 0 {
		searchQuery = inboxEventSearchQuery()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		listEventSize, err := runInboxListEvent(fixture, searchQuery)
		if err != nil {
			b.Fatalf("inbox list event failed: %v", err)
		}
		if listEventSize == 0 {
			b.Fatal("inbox list event returned empty body")
		}
		if listEventSize < 100 {
			b.Fatalf("inbox list event body too small: %d", listEventSize)
		}
	}

	b.StopTimer()
	reportEventTime(b)
}

func benchmarkBrowseListingHandler(b *testing.B, fixture *listingBenchmarkFixture, expectedRowCount int) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		listEventSize, err := runBrowseListEvent(fixture)
		if err != nil {
			b.Fatalf("browse list event failed: %v", err)
		}
		if listEventSize == 0 {
			b.Fatal("browse list event returned empty body")
		}
		if expectedRowCount > 0 && listEventSize < 100 {
			b.Fatalf("browse list event body too small: %d", listEventSize)
		}
	}

	b.StopTimer()
	reportEventTime(b)
}

func runInboxListEvent(fixture *listingBenchmarkFixture, searchQuery string) (int, error) {
	form := url.Values{}
	form.Set("SelectedFileID", "")
	form.Set("SearchQuery", searchQuery)
	form.Set("ActiveSideSheet", "")
	form.Set("SortBy", "newestFirst")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/inbox/files-list-partial?hx-target=%23fileList",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	err := fixture.actions.Inbox.ListFilesPartial.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		fixture.spaceCtx,
	)
	if err != nil {
		return 0, err
	}
	if rr.Code != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", rr.Code)
	}

	return rr.Body.Len(), nil
}

func runBrowseListEvent(fixture *listingBenchmarkFixture) (int, error) {
	form := url.Values{}
	form.Set("CurrentDirID", fixture.rootDirPublicID)
	form.Set("SelectedFileID", "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/browse/list-dir-partial?hx-target=%23fileList",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	err := fixture.actions.Browse.ListDirPartial.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		fixture.spaceCtx,
	)
	if err != nil {
		return 0, err
	}
	if rr.Code != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", rr.Code)
	}

	return rr.Body.Len(), nil
}

func reportEventTime(b *testing.B) {
	b.Helper()

	if b.N == 0 {
		return
	}

	msPerEvent := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / float64(time.Millisecond)
	b.ReportMetric(msPerEvent, "ms/event")
}

func inboxListingQuery(spaceCtx *ctxx.SpaceContext, spaceID int64, searchQuery string) ([]*enttenant.File, error) {
	query := spaceCtx.TTx.File.Query().
		WithParent().
		WithChildren().
		Where(
			file.SpaceID(spaceID),
			file.IsInInbox(true),
		).
		Order(file.ByCreatedAt(sql.OrderDesc()))

	if searchQuery != "" {
		query = query.Where(file.NameContains(searchQuery))
	}

	return query.All(spaceCtx)
}

func browseListingQuery(spaceCtx *ctxx.SpaceContext, spaceID int64, rootDirID int64) ([]*enttenant.File, error) {
	return spaceCtx.TTx.File.Query().
		WithParent().
		WithChildren().
		Where(
			file.ParentID(rootDirID),
			file.SpaceID(spaceID),
			file.IsInInbox(false),
		).
		Order(file.ByIsDirectory(sql.OrderDesc()), file.ByName()).
		Limit(51).
		All(spaceCtx)
}

func newListingBenchmarkFixture(
	tb testing.TB,
	fileCount int,
	isInInbox bool,
) *listingBenchmarkFixture {
	tb.Helper()

	harness := newActionTestHarnessWithSaaS(tb, true)

	email := fmt.Sprintf("bench-listing-%d@example.com", time.Now().UnixNano())
	accountx, tenantx := signUpAccount(tb, harness, email)
	tenantDB := initTenantDB(tb, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	mainTx, tenantTx, tenantCtx := newTenantContext(tb, harness, accountx, tenantx, tenantDB)
	spaceName := fmt.Sprintf("Listing Benchmark %d", time.Now().UnixNano())
	createSpaceViaCmd(tb, harness.actions, tenantCtx, spaceName)

	spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	rootDir := spaceCtx.SpaceRootDir()

	versionSeedLimit := 0
	if !isInInbox {
		versionSeedLimit = 51
		if versionSeedLimit > fileCount {
			versionSeedLimit = fileCount
		}
	}

	err := seedListingBenchmarkFiles(
		spaceCtx,
		rootDir.ID,
		spacex.ID,
		fileCount,
		isInInbox,
		versionSeedLimit,
	)
	if err != nil {
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
		tb.Fatalf("seed benchmark files: %v", err)
	}

	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		tb.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		tb.Fatalf("commit tenant tx: %v", err)
	}

	mainReadOnlyTx, tenantReadOnlyTx, tenantReadOnlyCtx, err := newTenantContextForUpload(
		harness,
		accountx,
		tenantx,
		tenantDB,
	)
	if err != nil {
		tb.Fatalf("new read-only tenant context: %v", err)
	}

	readOnlySpace := tenantReadOnlyCtx.TTx.Space.Query().Where(space.ID(spacex.ID)).OnlyX(tenantReadOnlyCtx)
	readOnlySpaceCtx := ctxx.NewSpaceContext(tenantReadOnlyCtx, readOnlySpace)

	cleanup := func() {
		_ = tenantReadOnlyTx.Rollback()
		_ = mainReadOnlyTx.Rollback()
	}

	return &listingBenchmarkFixture{
		actions:         harness.actions,
		spaceCtx:        readOnlySpaceCtx,
		spaceID:         spacex.ID,
		rootDirID:       rootDir.ID,
		rootDirPublicID: rootDir.PublicID.String(),
		cleanup:         cleanup,
	}
}

func seedListingBenchmarkFiles(
	spaceCtx *ctxx.SpaceContext,
	rootDirID int64,
	spaceID int64,
	fileCount int,
	isInInbox bool,
	versionSeedLimit int,
) error {
	const chunkSize = 1000
	now := time.Now()

	filesWithVersions := make([]*enttenant.File, 0, versionSeedLimit)

	for start := 0; start < fileCount; start += chunkSize {
		end := start + chunkSize
		if end > fileCount {
			end = fileCount
		}

		builders := make([]*enttenant.FileCreate, 0, end-start)
		for i := start; i < end; i++ {
			builders = append(builders, spaceCtx.TTx.File.Create().
				SetName(fmt.Sprintf("benchmark-file-%06d.txt", i)).
				SetIsDirectory(false).
				SetIndexedAt(now).
				SetModifiedAt(now).
				SetParentID(rootDirID).
				SetSpaceID(spaceID).
				SetIsInInbox(isInInbox),
			)
		}

		createdFiles, err := spaceCtx.TTx.File.CreateBulk(builders...).Save(spaceCtx)
		if err != nil {
			return err
		}

		if len(filesWithVersions) < versionSeedLimit {
			remaining := versionSeedLimit - len(filesWithVersions)
			if remaining > len(createdFiles) {
				remaining = len(createdFiles)
			}
			filesWithVersions = append(filesWithVersions, createdFiles[:remaining]...)
		}
	}

	if len(filesWithVersions) > 0 {
		if err := seedStoredFilesForBenchmarkRows(spaceCtx, filesWithVersions); err != nil {
			return err
		}
	}

	return nil
}

func seedStoredFilesForBenchmarkRows(spaceCtx *ctxx.SpaceContext, files []*enttenant.File) error {
	now := time.Now()

	for _, filex := range files {
		storedFilex, err := spaceCtx.TTx.StoredFile.Create().
			SetFilename(filex.Name).
			SetSize(1).
			SetSizeInStorage(1).
			SetStorageType(storagetype.Local).
			SetStoragePath("benchmark/storage").
			SetStorageFilename(filex.PublicID.String() + ".txt").
			SetTemporaryStoragePath("benchmark/tmp").
			SetTemporaryStorageFilename(filex.PublicID.String() + ".tmp").
			SetUploadSucceededAt(now).
			Save(spaceCtx)
		if err != nil {
			return err
		}

		_, err = spaceCtx.TTx.FileVersion.Create().
			SetFileID(filex.ID).
			SetStoredFileID(storedFilex.ID).
			SetVersionNumber(1).
			Save(spaceCtx)
		if err != nil {
			return err
		}
	}

	return nil
}
