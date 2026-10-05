package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/common/execution"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUploadFromURLStagedFilePersistsAfterRequestCancellation(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		testCanceledURLUploadPersists(t, disableEncryption)
	})
}

func testCanceledURLUploadPersists(t *testing.T, disableEncryption bool) {
	t.Helper()
	harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
	accountx, tenantx := signUpAccount(t, harness, "from-url-canceled@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadOnlyConn.Tenant.GetX(context.Background(), tenantx.ID)
	uploadToken := stageURLUploadForTest(t, harness, accountx, "from-url.txt")
	spaceID, spacePublicID := createSpaceForURLUploadTest(
		t, harness, accountx, tenantx, tenantDB, "Canceled Upload Space",
	)
	ctx := canceledRequestScopeForURLUpload(
		t, harness, accountx, tenantx, tenantDB, spacePublicID,
	)
	form := url.Values{"UploadToken": {uploadToken}}
	req := httptest.NewRequest(
		http.MethodPost,
		consumeUploadsEndpointForTest(tenantx, spacePublicID),
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	err := harness.actions.Inbox.ConsumeUploadsCmd.Handler(
		httpx.NewResponseWriter(httptest.NewRecorder()), httpx.NewRequest(req), ctx,
	)
	if err != nil {
		t.Fatalf("consume uploads after request cancellation: %v", err)
	}
	assertCanceledURLUploadPersisted(t, harness, tenantDB, accountx.ID, uploadToken, spaceID)
}

func canceledRequestScopeForURLUpload(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	tenantDB *sqlx.TenantDB,
	spacePublicID string,
) ctxx.Context {
	t.Helper()
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	mainTx, err := harness.mainDB.ReadOnlyConn.Tx(requestCtx)
	if err != nil {
		t.Fatalf("start main read transaction: %v", err)
	}
	visitorCtx := ctxx.NewVisitorContext(
		requestCtx, mainTx, harness.i18n, "", "", false, false,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(
		visitorCtx, accountx, harness.i18n, harness.mainDB, harness.tenantDBs, true,
	)
	tenantTx, err := tenantDB.ReadOnlyConn.Tx(requestCtx)
	if err != nil {
		t.Fatalf("start tenant read transaction: %v", err)
	}
	ctx, err := execution.NewScopeResolver().Resolve(mainCtx, tenantTx, tenantx, spacePublicID, true)
	if err != nil {
		t.Fatalf("resolve space scope: %v", err)
	}
	// Like Router.wrapManualTx: authorization transactions close before the command runs.
	if err := mainTx.Commit(); err != nil {
		t.Fatalf("commit main read transaction: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant read transaction: %v", err)
	}
	cancelRequest()
	return ctx
}

func assertCanceledURLUploadPersisted(
	t *testing.T,
	harness *actionTestHarness,
	tenantDB *sqlx.TenantDB,
	accountID int64,
	uploadToken string,
	spaceID int64,
) {
	t.Helper()
	temporaryFiles := harness.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
		temporaryfile.OwnerID(accountID), temporaryfile.UploadToken(uploadToken),
		temporaryfile.ConvertedToStoredFileAtNotNil(),
	).AllX(context.Background())
	if len(temporaryFiles) != 1 {
		t.Fatalf("expected staged temporary file to be converted, got %d", len(temporaryFiles))
	}
	inboxFiles := tenantDB.ReadOnlyConn.File.Query().Where(
		file.SpaceID(spaceID), file.IsInInbox(true),
	).AllX(privacy.DecisionContext(context.Background(), privacy.Allow))
	if len(inboxFiles) != 1 {
		t.Fatalf("expected exactly one inbox file, got %d", len(inboxFiles))
	}
}

func TestUploadFromURLFirstInboxResponseContainsImportedFile(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		for _, htmx := range []bool{false, true} {
			name := "normal"
			if htmx {
				name = "htmx"
			}
			t.Run(name, func(t *testing.T) {
				t.Setenv("SIMPLEDMS_DB_READ_ONLY_MAX_OPEN_CONNS", "1")
				harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
				// Registered like in Server: the Inbox pages are read-only queries.
				harness.router.RegisterPage(
					route.InboxRoute(false, false),
					harness.actions.Inbox.InboxRootPage.Handler,
				)
				harness.router.RegisterPage(
					route.InboxRoute(true, false),
					harness.actions.Inbox.InboxWithSelectionPage.Handler,
				)
				accountx, tenantx := signUpAccount(t, harness, "first-inbox-response@example.com")
				tenantDB := initTenantDB(t, harness, tenantx)
				tenantx = harness.mainDB.ReadOnlyConn.Tenant.GetX(context.Background(), tenantx.ID)
				filename := "imported-first-response.txt"

				uploadToken := stageURLUploadForTest(t, harness, accountx, filename)
				spaceID, spacePublicID := createSpaceForURLUploadTest(
					t,
					harness,
					accountx,
					tenantx,
					tenantDB,
					"First Response Space",
				)

				session := createSessionForAccountForRulesTest(t, harness, accountx.ID)
				requestCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				serve := func(method, target string, form url.Values) *httptest.ResponseRecorder {
					req := httptest.NewRequest(method, target, strings.NewReader(form.Encode())).
						WithContext(requestCtx)
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session, Path: "/"})
					if htmx {
						req.Header.Set("HX-Request", "true")
						req.Header.Set("HX-Current-URL", "/open-file/select-space/"+uploadToken)
					}
					rr := httptest.NewRecorder()
					harness.router.ServeHTTP(rr, req)
					return rr
				}
				consumeUploads := func() *httptest.ResponseRecorder {
					return serve(
						http.MethodPost,
						consumeUploadsEndpointForTest(tenantx, spacePublicID),
						url.Values{"UploadToken": {uploadToken}},
					)
				}

				inboxURL := route.InboxRoot(tenantx.PublicID.String(), spacePublicID)
				consumeRR := consumeUploads()
				assertConsumeUploadsNavigatesToInbox(t, consumeRR, htmx, inboxURL)

				rr := serve(http.MethodGet, inboxURL, nil)
				if rr.Code != http.StatusOK {
					t.Fatalf("expected first inbox response status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
				}
				if !strings.Contains(rr.Body.String(), filename) {
					t.Fatalf("expected first inbox response to contain %q, got %s", filename, rr.Body.String())
				}

				converted := harness.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
					temporaryfile.OwnerID(accountx.ID),
					temporaryfile.UploadToken(uploadToken),
					temporaryfile.ConvertedToStoredFileAtNotNil(),
				).AllX(context.Background())
				if len(converted) != 1 {
					t.Fatalf("expected exactly one converted temporary file, got %d", len(converted))
				}
				inboxFiles := tenantDB.ReadOnlyConn.File.Query().Where(
					file.SpaceID(spaceID),
					file.IsInInbox(true),
				).AllX(privacy.DecisionContext(context.Background(), privacy.Allow))
				if len(inboxFiles) != 1 {
					t.Fatalf("expected exactly one inbox file, got %d", len(inboxFiles))
				}
				if inboxFiles[0].Name != filename {
					t.Fatalf("expected inbox file %q, got %q", filename, inboxFiles[0].Name)
				}
				if inboxFiles[0].Source != filesource.URLImport {
					t.Fatalf("expected URL import source, got %q", inboxFiles[0].Source)
				}

				selectedURL := route.Inbox(
					tenantx.PublicID.String(),
					spacePublicID,
					inboxFiles[0].PublicID.String(),
				)
				selectedRR := serve(http.MethodGet, selectedURL, nil)
				if selectedRR.Code != http.StatusOK {
					t.Fatalf(
						"expected selected inbox response status %d, got %d: %s",
						http.StatusOK,
						selectedRR.Code,
						selectedRR.Body.String(),
					)
				}
				if !strings.Contains(selectedRR.Body.String(), filename) {
					t.Fatalf("expected selected inbox response to contain %q, got %s", filename, selectedRR.Body.String())
				}

				// Replaying the token must not create another file.
				replayRR := consumeUploads()
				assertConsumeUploadsNavigatesToInbox(t, replayRR, htmx, inboxURL)
				inboxFiles = tenantDB.ReadOnlyConn.File.Query().Where(
					file.SpaceID(spaceID),
					file.IsInInbox(true),
				).AllX(privacy.DecisionContext(context.Background(), privacy.Allow))
				if len(inboxFiles) != 1 {
					t.Fatalf("expected replay to keep exactly one inbox file, got %d", len(inboxFiles))
				}
			})
		}
	})
}

func stageURLUploadForTest(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	filename string,
) string {
	t.Helper()
	harness.actions.OpenFile.UploadFromURLCmd.SetDownloadFileForTesting(
		func(_ context.Context, _ string) (string, io.ReadCloser, error) {
			return filename, io.NopCloser(strings.NewReader("hello from url")), nil
		},
	)

	var uploadToken string
	err := withMainContext(t, harness, accountx, func(
		_ *entmain.Tx,
		mainCtx *ctxx.MainContext,
	) error {
		data := url.Values{"url": {"https://example.com/" + filename}}
		req := httptest.NewRequest(
			http.MethodPost,
			"/-/open-file/upload-from-url-cmd",
			strings.NewReader(data.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		if err := harness.actions.OpenFile.UploadFromURLCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			mainCtx,
		); err != nil {
			return fmt.Errorf("stage URL file: %w", err)
		}
		uploadToken = strings.TrimPrefix(rr.Header().Get("Location"), "/open-file/select-space/")
		if uploadToken == "" {
			return errors.New("expected upload token in redirect")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return uploadToken
}

func createSpaceForURLUploadTest(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	tenantDB *sqlx.TenantDB,
	name string,
) (int64, string) {
	t.Helper()
	var spaceID int64
	var spacePublicID string
	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, name)
		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(name)).OnlyX(tenantCtx)
		spaceID = spacex.ID
		spacePublicID = spacex.PublicID.String()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return spaceID, spacePublicID
}

func consumeUploadsEndpointForTest(tenantx *entmain.Tenant, spacePublicID string) string {
	return "/-/org/" + tenantx.PublicID.String() + "/space/" + spacePublicID +
		"/inbox/consume-uploads"
}

func assertConsumeUploadsNavigatesToInbox(
	t *testing.T,
	rr *httptest.ResponseRecorder,
	htmx bool,
	inboxURL string,
) {
	t.Helper()
	if htmx {
		if rr.Code != http.StatusOK || rr.Header().Get("HX-Location") != inboxURL {
			t.Fatalf("consume uploads: %d %v %s", rr.Code, rr.Header(), rr.Body.String())
		}
		return
	}
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != inboxURL {
		t.Fatalf("consume uploads: %d %v %s", rr.Code, rr.Header(), rr.Body.String())
	}
}
