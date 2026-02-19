package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
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
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/model/common/storagetype"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/sqlutil"
)

type listingBenchmarkFixture struct {
	actions         *action.Actions
	router          *Router
	spaceCtx        *ctxx.SpaceContext
	spaceID         int64
	tenantPublicID  string
	spacePublicID   string
	rootDirID       int64
	rootDirPublicID string
	sessionValue    string
	cleanup         func()
}

func inboxEventSearchQuery() string {
	return "benchmark-file-000000.txt"
}

func browseFTSBenchmarkSearchQuery() string {
	return sqlutil.FTSSafeAndQuery(inboxEventSearchQuery(), 300)
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
			b.Run("router", func(b *testing.B) {
				benchmarkInboxListingRouter(b, fixture, fileCount)
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
			b.Run("router", func(b *testing.B) {
				benchmarkBrowseListingRouter(b, fixture, expectedRowCount)
			})
		})
	}
}

func BenchmarkFileListingAcrossTenSpaces(b *testing.B) {
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		log.SetOutput(originalLogOutput)
	})

	fileCounts := []int{0, 1000, 5000, 10000, 100000, 1000000}
	const spaceCount = 10

	for _, fileCount := range fileCounts {
		fileCount := fileCount
		targetSpaceFileCount := distributedFileCount(fileCount, spaceCount, 0)

		b.Run(fmt.Sprintf("inbox_%d", fileCount), func(b *testing.B) {
			fixture := newListingBenchmarkFixtureAcrossSpaces(b, fileCount, true, spaceCount)
			defer fixture.cleanup()

			expectedRowCount := 0
			if targetSpaceFileCount > 0 {
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
				benchmarkInboxListingHandler(b, fixture, targetSpaceFileCount)
			})
			b.Run("router", func(b *testing.B) {
				benchmarkInboxListingRouter(b, fixture, targetSpaceFileCount)
			})
		})

		b.Run(fmt.Sprintf("browse_%d", fileCount), func(b *testing.B) {
			fixture := newListingBenchmarkFixtureAcrossSpaces(b, fileCount, false, spaceCount)
			defer fixture.cleanup()

			rows, err := browseListingQuery(fixture.spaceCtx, fixture.spaceID, fixture.rootDirID)
			if err != nil {
				b.Fatalf("initial browse listing query failed: %v", err)
			}

			expectedRowCount := targetSpaceFileCount
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
			b.Run("router", func(b *testing.B) {
				benchmarkBrowseListingRouter(b, fixture, expectedRowCount)
			})
		})
	}
}

func BenchmarkBrowseFTSQuery(b *testing.B) {
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		log.SetOutput(originalLogOutput)
	})

	fileCounts := []int{0, 1000, 5000, 10000, 100000, 1000000}
	searchQuery := browseFTSBenchmarkSearchQuery()

	for _, fileCount := range fileCounts {
		fileCount := fileCount

		b.Run(fmt.Sprintf("single_space_%d", fileCount), func(b *testing.B) {
			fixture := newListingBenchmarkFixture(b, fileCount, false)
			defer fixture.cleanup()

			expectedRowCount := 0
			if fileCount > 0 {
				expectedRowCount = 1
			}

			rows, err := browseFTSListingQuery(fixture.spaceCtx, fixture.spaceID, searchQuery)
			if err != nil {
				b.Fatalf("initial browse fts query failed: %v", err)
			}
			if len(rows) != expectedRowCount {
				b.Fatalf("expected %d browse fts rows, got %d", expectedRowCount, len(rows))
			}

			benchmarkBrowseFTSQuery(b, fixture, searchQuery, expectedRowCount)
		})

		b.Run(fmt.Sprintf("ten_spaces_%d", fileCount), func(b *testing.B) {
			fixture := newListingBenchmarkFixtureAcrossSpaces(b, fileCount, false, 10)
			defer fixture.cleanup()

			expectedRowCount := 0
			if distributedFileCount(fileCount, 10, 0) > 0 {
				expectedRowCount = 1
			}

			rows, err := browseFTSListingQuery(fixture.spaceCtx, fixture.spaceID, searchQuery)
			if err != nil {
				b.Fatalf("initial browse fts query failed: %v", err)
			}
			if len(rows) != expectedRowCount {
				b.Fatalf("expected %d browse fts rows, got %d", expectedRowCount, len(rows))
			}

			benchmarkBrowseFTSQuery(b, fixture, searchQuery, expectedRowCount)
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

func benchmarkBrowseFTSQuery(
	b *testing.B,
	fixture *listingBenchmarkFixture,
	searchQuery string,
	expectedRowCount int,
) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rows, err := browseFTSListingQuery(fixture.spaceCtx, fixture.spaceID, searchQuery)
		if err != nil {
			b.Fatalf("browse fts query failed: %v", err)
		}
		if len(rows) != expectedRowCount {
			b.Fatalf("expected %d browse fts rows, got %d", expectedRowCount, len(rows))
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

func benchmarkInboxListingRouter(b *testing.B, fixture *listingBenchmarkFixture, fileCount int) {
	b.Helper()
	b.ReportAllocs()

	searchQuery := ""
	if fileCount > 0 {
		searchQuery = inboxEventSearchQuery()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		listEventSize, err := runInboxListEventViaRouter(fixture, searchQuery)
		if err != nil {
			b.Fatalf("inbox list event via router failed: %v", err)
		}
		if listEventSize == 0 {
			b.Fatal("inbox list event via router returned empty body")
		}
		if listEventSize < 100 {
			b.Fatalf("inbox list event via router body too small: %d", listEventSize)
		}
	}

	b.StopTimer()
	reportEventTime(b)
}

func benchmarkBrowseListingRouter(b *testing.B, fixture *listingBenchmarkFixture, expectedRowCount int) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		listEventSize, err := runBrowseListEventViaRouter(fixture)
		if err != nil {
			b.Fatalf("browse list event via router failed: %v", err)
		}
		if listEventSize == 0 {
			b.Fatal("browse list event via router returned empty body")
		}
		if expectedRowCount > 0 && listEventSize < 100 {
			b.Fatalf("browse list event via router body too small: %d", listEventSize)
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

func runInboxListEventViaRouter(fixture *listingBenchmarkFixture, searchQuery string) (int, error) {
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
	req.Header.Set("HX-Current-URL", route.InboxRoot(fixture.tenantPublicID, fixture.spacePublicID))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: fixture.sessionValue,
	})

	rr := httptest.NewRecorder()
	fixture.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", rr.Code)
	}

	return rr.Body.Len(), nil
}

func runBrowseListEventViaRouter(fixture *listingBenchmarkFixture) (int, error) {
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
	req.Header.Set("HX-Current-URL", route.Browse(fixture.tenantPublicID, fixture.spacePublicID, fixture.rootDirPublicID))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: fixture.sessionValue,
	})

	rr := httptest.NewRecorder()
	fixture.router.ServeHTTP(rr, req)
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

func browseFTSListingQuery(
	spaceCtx *ctxx.SpaceContext,
	spaceID int64,
	searchQuery string,
) ([]*enttenant.File, error) {
	query := spaceCtx.TTx.File.Query().
		WithParent().
		WithChildren().
		Where(
			file.SpaceID(spaceID),
			file.IsInInbox(false),
		)

	if searchQuery != "" {
		query = query.Where(
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)

				qs.Where(
					sql.In(
						qs.C(file.FieldID),
						sql.Select(fileSearchTable.C(filesearch.FieldRowid)).From(fileSearchTable).
							Where(
								sql.And(
									sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), searchQuery),
									sql.LT(fileSearchTable.C(filesearch.FieldRank), 0),
								),
							).
							OrderBy(fileSearchTable.C(filesearch.FieldRank)),
					),
				)
			},
		)
	}

	return query.
		Order(file.ByIsDirectory(sql.OrderDesc()), file.ByName()).
		Limit(51).
		All(spaceCtx)
}

func newListingBenchmarkFixture(
	tb testing.TB,
	fileCount int,
	isInInbox bool,
) *listingBenchmarkFixture {
	return newListingBenchmarkFixtureAcrossSpaces(tb, fileCount, isInInbox, 1)
}

func newListingBenchmarkFixtureAcrossSpaces(
	tb testing.TB,
	fileCount int,
	isInInbox bool,
	spaceCount int,
) *listingBenchmarkFixture {
	tb.Helper()
	if spaceCount <= 0 {
		tb.Fatal("spaceCount must be greater than 0")
	}

	harness := newActionTestHarnessWithSaaS(tb, true)

	email := fmt.Sprintf("bench-listing-%d@example.com", time.Now().UnixNano())
	accountx, tenantx := signUpAccount(tb, harness, email)
	tenantDB := initTenantDB(tb, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	mainTx, tenantTx, tenantCtx := newTenantContext(tb, harness, accountx, tenantx, tenantDB)

	spaceNamePrefix := fmt.Sprintf("Listing Benchmark %d", time.Now().UnixNano())
	var targetSpace *enttenant.Space
	var targetRootDir *enttenant.File

	for spaceIndex := 0; spaceIndex < spaceCount; spaceIndex++ {
		spaceName := fmt.Sprintf("%s Space %02d", spaceNamePrefix, spaceIndex)
		createSpaceViaCmd(tb, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()

		spaceFileCount := distributedFileCount(fileCount, spaceCount, spaceIndex)

		versionSeedLimit := 0
		if !isInInbox && spaceIndex == 0 {
			versionSeedLimit = 51
			if versionSeedLimit > spaceFileCount {
				versionSeedLimit = spaceFileCount
			}
		}

		err := seedListingBenchmarkFiles(
			spaceCtx,
			rootDir.ID,
			spacex.ID,
			spaceFileCount,
			isInInbox,
			versionSeedLimit,
		)
		if err != nil {
			_ = tenantTx.Rollback()
			_ = mainTx.Rollback()
			tb.Fatalf("seed benchmark files: %v", err)
		}

		if spaceIndex == 0 {
			targetSpace = spacex
			targetRootDir = rootDir
		}
	}

	if targetSpace == nil || targetRootDir == nil {
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
		tb.Fatal("target space not initialized")
	}

	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		tb.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		tb.Fatalf("commit tenant tx: %v", err)
	}

	sessionValue := createBenchmarkSession(tb, harness, accountx.ID)

	mainReadOnlyTx, tenantReadOnlyTx, tenantReadOnlyCtx, err := newTenantContextForUpload(
		harness,
		accountx,
		tenantx,
		tenantDB,
	)
	if err != nil {
		tb.Fatalf("new read-only tenant context: %v", err)
	}

	readOnlySpace := tenantReadOnlyCtx.TTx.Space.Query().Where(space.ID(targetSpace.ID)).OnlyX(tenantReadOnlyCtx)
	readOnlySpaceCtx := ctxx.NewSpaceContext(tenantReadOnlyCtx, readOnlySpace)

	cleanup := func() {
		_ = tenantReadOnlyTx.Rollback()
		_ = mainReadOnlyTx.Rollback()
	}

	return &listingBenchmarkFixture{
		actions:         harness.actions,
		router:          harness.router,
		spaceCtx:        readOnlySpaceCtx,
		spaceID:         targetSpace.ID,
		tenantPublicID:  tenantx.PublicID.String(),
		spacePublicID:   targetSpace.PublicID.String(),
		rootDirID:       targetRootDir.ID,
		rootDirPublicID: targetRootDir.PublicID.String(),
		sessionValue:    sessionValue,
		cleanup:         cleanup,
	}
}

func createBenchmarkSession(tb testing.TB, harness *actionTestHarness, accountID int64) string {
	tb.Helper()

	sessionValue := fmt.Sprintf("bench-session-%d-%d", time.Now().UnixNano(), rand.IntN(1_000_000))
	expiresAt := time.Now().Add(14 * 24 * time.Hour)

	_, err := harness.mainDB.ReadWriteConn.Session.Create().
		SetValue(sessionValue).
		SetAccountID(accountID).
		SetIsTemporarySession(false).
		SetExpiresAt(expiresAt).
		SetDeletableAt(expiresAt).
		Save(context.Background())
	if err != nil {
		tb.Fatalf("create benchmark session: %v", err)
	}

	return sessionValue
}

func distributedFileCount(totalFileCount int, spaceCount int, spaceIndex int) int {
	if totalFileCount <= 0 {
		return 0
	}

	baseCount := totalFileCount / spaceCount
	remainder := totalFileCount % spaceCount
	if spaceIndex < remainder {
		return baseCount + 1
	}

	return baseCount
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
