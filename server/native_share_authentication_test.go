package server

import (
	"context"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestNativeShareUploadReachesInboxAfterSpaceSelection(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
		harness.router.RegisterPage(
			route.SelectSpaceRoute(false), harness.actions.OpenFile.SelectSpacePage.Handler,
		)
		harness.router.RegisterPage(
			route.SelectSpaceRoute(true), harness.actions.OpenFile.SelectSpacePage.Handler,
		)
		harness.router.RegisterPage(
			route.InboxRoute(false, false), harness.actions.Inbox.InboxRootPage.Handler,
		)
		harness.router.RegisterPage(
			route.InboxRoute(true, false), harness.actions.Inbox.InboxWithSelectionPage.Handler,
		)

		accountx, tenantx := signUpAccount(t, harness, "native-share@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadOnlyConn.Tenant.GetX(context.Background(), tenantx.ID)
		spaceID, spacePublicID := createSpaceForURLUploadTest(
			t, harness, accountx, tenantx, tenantDB, "Native Share Space",
		)
		token := ""
		session := createSessionForAccountForRulesTest(t, harness, accountx.ID)
		cookie := &http.Cookie{Name: cookiex.SessionCookieName(), Value: session, Path: "/"}
		handler := http.NewCrossOriginProtection().Handler(harness.router)
		serve := func(req *http.Request) *httptest.ResponseRecorder {
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			return rr
		}

		uploadReq := newSharedUploadRequest(t, map[string]string{
			"shared-one.txt": "first shared content",
			"shared-two.txt": "second shared content",
		})
		uploadReq.Header.Set("Origin", "null")
		uploadReq.Header.Set("Sec-Fetch-Site", "none")
		uploadRR := serve(uploadReq)
		if uploadRR.Code != http.StatusFound ||
			!strings.HasPrefix(uploadRR.Header().Get("Location"), "/open-file/select-space/") {
			t.Fatalf("expected redirect to select-space, got status %d and location %q",
				uploadRR.Code, uploadRR.Header().Get("Location"))
		}
		selectURL := uploadRR.Header().Get("Location")
		token = strings.TrimPrefix(selectURL, "/open-file/select-space/")
		if token == "" {
			t.Fatal("expected upload token in select-space redirect")
		}
		staged := harness.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
			temporaryfile.OwnerID(accountx.ID), temporaryfile.UploadToken(token),
		).AllX(context.Background())
		if len(staged) != 2 {
			t.Fatalf("expected two staged account-owned files, got %d", len(staged))
		}
		for _, stagedFile := range staged {
			if stagedFile.UploadSucceededAt == nil || stagedFile.UploadSucceededAt.IsZero() {
				t.Errorf("expected %q to be finalized", stagedFile.Filename)
			}
		}

		inboxURL := route.InboxRoot(tenantx.PublicID.String(), spacePublicID)
		selectRR := serve(httptest.NewRequest(http.MethodGet, selectURL, nil))
		if selectRR.Code != http.StatusOK {
			t.Fatalf("expected select-space page status %d, got %d: %s",
				http.StatusOK, selectRR.Code, selectRR.Body.String())
		}
		selectionHTML := html.UnescapeString(selectRR.Body.String())
		consumeURL := consumeUploadsEndpointForTest(tenantx, spacePublicID)
		if !strings.Contains(selectionHTML, "Native Share Space") ||
			!strings.Contains(selectionHTML, `hx-post="`+consumeURL+`"`) ||
			!strings.Contains(selectionHTML, `"UploadToken":"`+token+`"`) {
			t.Fatalf("expected selection page to offer the destination space and upload token, got %s",
				selectRR.Body.String())
		}
		filesBeforeConsume := tenantDB.ReadOnlyConn.File.Query().Where(
			file.SpaceID(spaceID), file.IsInInbox(true),
		).AllX(privacy.DecisionContext(context.Background(), privacy.Allow))
		if len(filesBeforeConsume) != 0 {
			t.Fatalf("expected selection query not to consume uploads, got %d files",
				len(filesBeforeConsume))
		}

		consume := func() *httptest.ResponseRecorder {
			form := url.Values{"UploadToken": {token}}
			req := httptest.NewRequest(
				http.MethodPost,
				consumeUploadsEndpointForTest(tenantx, spacePublicID),
				strings.NewReader(form.Encode()),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			req.Header.Set("HX-Current-URL", selectURL)
			return serve(req)
		}
		consumeRR := consume()
		assertConsumeUploadsNavigatesToInbox(t, consumeRR, true, inboxURL)

		inboxRR := serve(httptest.NewRequest(http.MethodGet, inboxURL, nil))
		if inboxRR.Code != http.StatusOK {
			t.Fatalf("expected first inbox response status %d, got %d: %s",
				http.StatusOK, inboxRR.Code, inboxRR.Body.String())
		}
		for _, filename := range []string{"shared-one.txt", "shared-two.txt"} {
			if !strings.Contains(inboxRR.Body.String(), filename) {
				t.Errorf("expected first inbox response to contain %q, got %s", filename, inboxRR.Body.String())
			}
		}
		converted := harness.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
			temporaryfile.OwnerID(accountx.ID),
			temporaryfile.UploadToken(token),
			temporaryfile.ConvertedToStoredFileAtNotNil(),
		).AllX(context.Background())
		if len(converted) != 2 {
			t.Fatalf("expected both staged files to be converted, got %d", len(converted))
		}
		inboxFiles := tenantDB.ReadOnlyConn.File.Query().Where(
			file.SpaceID(spaceID), file.IsInInbox(true),
		).AllX(privacy.DecisionContext(context.Background(), privacy.Allow))
		if len(inboxFiles) != 2 {
			t.Fatalf("expected exactly two inbox files, got %d", len(inboxFiles))
		}
		for _, inboxFile := range inboxFiles {
			if inboxFile.Source != filesource.PWAOSOpen {
				t.Errorf("expected native share source %q, got %q", filesource.PWAOSOpen, inboxFile.Source)
			}
		}

		replayRR := consume()
		assertConsumeUploadsNavigatesToInbox(t, replayRR, true, inboxURL)
		inboxFiles = tenantDB.ReadOnlyConn.File.Query().Where(
			file.SpaceID(spaceID), file.IsInInbox(true),
		).AllX(privacy.DecisionContext(context.Background(), privacy.Allow))
		if len(inboxFiles) != 2 {
			t.Fatalf("expected token replay to keep exactly two inbox files, got %d", len(inboxFiles))
		}
	})
}

func TestNativeShareAuthenticationAndCrossOriginProtection(t *testing.T) {
	harness := newActionTestHarness(t)
	email := "native-share-security@example.com"
	createAccount(t, harness.mainDB, email, "native-share-secret")
	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())
	sessionCookie := signInAndGetSessionCookie(t, harness, email, "native-share-secret")
	handler := http.NewCrossOriginProtection().Handler(harness.router)

	for _, tc := range []struct {
		name   string
		cookie bool
		origin string
		fetch  string
		host   string
	}{
		{name: "unauthenticated"},
		{
			name:   "cross-site despite session",
			cookie: true,
			origin: "https://attacker.example",
			fetch:  "cross-site",
		},
		{name: "cross-origin without fetch metadata", cookie: true, origin: "https://attacker.example"},
		{
			name:   "same-site cross-origin",
			cookie: true,
			origin: "http://sub.example.test",
			fetch:  "same-site",
			host:   "app.example.test",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := newSharedUploadRequest(t, map[string]string{"blocked.txt": "must not stage"})
			if tc.host != "" {
				req.Host = tc.host
			}
			if tc.cookie {
				req.AddCookie(sessionCookie)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
				if tc.fetch != "" {
					req.Header.Set("Sec-Fetch-Site", tc.fetch)
				}
			} else {
				req.Header.Set("Origin", "null")
				req.Header.Set("Sec-Fetch-Site", "none")
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if tc.cookie {
				if rr.Code != http.StatusForbidden {
					t.Fatalf("expected cross-origin protection status %d, got %d",
						http.StatusForbidden, rr.Code)
				}
			} else if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/" {
				t.Fatalf("expected unauthenticated redirect to /, got status %d and location %q",
					rr.Code, rr.Header().Get("Location"))
			} else {
				getRequest := httptest.NewRequest(http.MethodGet, "/", nil)
				getRequest.AddCookie(sessionCookie)
				getResponse := httptest.NewRecorder()
				handler.ServeHTTP(getResponse, getRequest)
				if getResponse.Code != http.StatusSeeOther ||
					getResponse.Header().Get("Location") != "/dashboard/" {
					t.Fatalf("expected GET / to redirect to dashboard, got status %d and location %q",
						getResponse.Code, getResponse.Header().Get("Location"))
				}
			}
			count := harness.mainDB.ReadWriteConn.TemporaryFile.Query().Where(
				temporaryfile.OwnerID(accountx.ID),
			).CountX(context.Background())
			if count != 0 {
				t.Fatalf("expected no staged files, got %d", count)
			}
		})
	}
}
