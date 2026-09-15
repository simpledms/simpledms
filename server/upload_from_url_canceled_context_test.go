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

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUploadFromURLStagedFilePersistsAfterRequestCancellation(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
		accountx, tenantx := signUpAccount(t, harness, "from-url-canceled@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadOnlyConn.Tenant.GetX(
			context.Background(),
			tenantx.ID,
		)

		harness.actions.OpenFile.UploadFromURLCmd.SetDownloadFileForTesting(
			func(_ context.Context, _ string) (string, io.ReadCloser, error) {
				return "from-url.txt", io.NopCloser(strings.NewReader("hello from url")), nil
			},
		)

		var uploadToken string
		err := withMainContext(t, harness, accountx, func(
			_ *entmain.Tx,
			mainCtx *ctxx.MainContext,
		) error {
			data := url.Values{"url": {"https://example.com/from-url.txt"}}
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
			uploadToken = strings.TrimPrefix(
				rr.Header().Get("Location"),
				"/open-file/select-space/",
			)
			if uploadToken == "" {
				return fmt.Errorf("expected upload token in redirect")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		var spaceID int64
		err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
			_ *entmain.Tx,
			_ *enttenant.Tx,
			tenantCtx *ctxx.TenantContext,
		) error {
			createSpaceViaCmd(t, harness.actions, tenantCtx, "Canceled Upload Space")
			spacex := tenantCtx.TTx.Space.Query().
				Where(space.Name("Canceled Upload Space")).
				OnlyX(tenantCtx)
			spaceID = spacex.ID
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		requestCtx, cancelRequest := context.WithCancel(context.Background())
		mainTx, err := harness.mainDB.ReadOnlyConn.Tx(requestCtx)
		if err != nil {
			t.Fatalf("start main read transaction: %v", err)
		}
		defer func() { _ = mainTx.Rollback() }()
		visitorCtx := ctxx.NewVisitorContext(
			requestCtx,
			mainTx,
			harness.i18n,
			"",
			"",
			true,
			false,
			harness.infra.SystemConfig().CommercialLicenseEnabled(),
		)
		mainCtx := ctxx.NewMainContext(
			visitorCtx,
			accountx,
			harness.i18n,
			harness.mainDB,
			harness.tenantDBs,
			true,
		)
		tenantTx, err := tenantDB.ReadOnlyConn.Tx(requestCtx)
		if err != nil {
			t.Fatalf("start tenant read transaction: %v", err)
		}
		defer func() { _ = tenantTx.Rollback() }()
		tenantCtx := ctxx.NewTenantContext(mainCtx, tenantTx, tenantx, true)
		spacex := tenantTx.Space.Query().
			Where(space.ID(spaceID)).
			OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		cancelRequest()

		req := httptest.NewRequest(
			http.MethodGet,
			"/inbox?upload_token="+url.QueryEscape(uploadToken),
			nil,
		)
		var handlerErr error
		var panicValue any
		func() {
			defer func() {
				panicValue = recover()
			}()
			_, handlerErr = harness.actions.Inbox.InboxPage.WidgetHandler(
				httpx.NewResponseWriter(httptest.NewRecorder()),
				httpx.NewRequest(req),
				spaceCtx,
				"",
			)
		}()
		if panicValue != nil {
			t.Fatalf("inbox page panicked: %v", panicValue)
		}
		if !errors.Is(handlerErr, context.Canceled) {
			t.Fatalf("expected canceled page request after persistence, got %v", handlerErr)
		}

		temporaryFiles := harness.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
			temporaryfile.OwnerID(accountx.ID),
			temporaryfile.UploadToken(uploadToken),
			temporaryfile.ConvertedToStoredFileAtNotNil(),
		).AllX(context.Background())
		if len(temporaryFiles) != 1 {
			t.Fatalf("expected staged temporary file to be converted, got %d", len(temporaryFiles))
		}

		inboxFiles := tenantDB.ReadOnlyConn.File.Query().Where(
			file.SpaceID(spaceID),
			file.IsInInbox(true),
		).AllX(privacy.DecisionContext(
			context.Background(),
			privacy.Allow,
		))
		if len(inboxFiles) != 1 {
			t.Fatalf("expected exactly one inbox file, got %d", len(inboxFiles))
		}
	})
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
				harness.router.RegisterManualTxPage(
					route.InboxRoute(false, false),
					harness.actions.Inbox.InboxRootPage.Handler,
				)
				harness.router.RegisterManualTxPage(
					route.InboxRoute(true, false),
					harness.actions.Inbox.InboxWithSelectionPage.Handler,
				)
				accountx, tenantx := signUpAccount(t, harness, "first-inbox-response@example.com")
				tenantDB := initTenantDB(t, harness, tenantx)
				tenantx = harness.mainDB.ReadOnlyConn.Tenant.GetX(context.Background(), tenantx.ID)
				filename := "imported-first-response.txt"

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

				var spaceID int64
				var spacePublicID string
				err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
					_ *entmain.Tx,
					_ *enttenant.Tx,
					tenantCtx *ctxx.TenantContext,
				) error {
					createSpaceViaCmd(t, harness.actions, tenantCtx, "First Response Space")
					spacex := tenantCtx.TTx.Space.Query().
						Where(space.Name("First Response Space")).OnlyX(tenantCtx)
					spaceID = spacex.ID
					spacePublicID = spacex.PublicID.String()
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}

				session := createSessionForAccountForRulesTest(t, harness, accountx.ID)
				inboxURL := route.InboxRoot(tenantx.PublicID.String(), spacePublicID) +
					"?upload_token=" + url.QueryEscape(uploadToken)
				requestCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				req := httptest.NewRequest(http.MethodGet, inboxURL, nil).WithContext(requestCtx)
				req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session, Path: "/"})
				if htmx {
					req.Header.Set("HX-Request", "true")
				}

				rr := httptest.NewRecorder()
				harness.router.ServeHTTP(rr, req)
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
				selectedReq := httptest.NewRequest(http.MethodGet, selectedURL, nil).WithContext(requestCtx)
				selectedReq.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session, Path: "/"})
				selectedRR := httptest.NewRecorder()
				harness.router.ServeHTTP(selectedRR, selectedReq)
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
				replayReq := httptest.NewRequest(http.MethodGet, inboxURL, nil).WithContext(requestCtx)
				replayReq.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session, Path: "/"})
				replayRR := httptest.NewRecorder()
				harness.router.ServeHTTP(replayRR, replayReq)
				if replayRR.Code != http.StatusOK {
					t.Fatalf("expected replay status %d, got %d", http.StatusOK, replayRR.Code)
				}
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
