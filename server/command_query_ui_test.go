package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestMarkAsDoneCommandEmitsInvalidationWithoutRenderingInbox(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "command-query-inbox")

	response := fixture.browserAt(
		route.Inbox(fixture.tenant.PublicID.String(), fixture.spaceID, fixture.fileID),
		h.actions.Inbox.MarkAsDoneCmd.Endpoint(),
		url.Values{"FileID": {fixture.fileID}},
	)
	if response.Code != http.StatusOK {
		t.Fatalf("mark-as-done command returned status %d", response.Code)
	}
	if response.Header().Get("HX-Trigger") != event.InboxChanged.String() {
		t.Fatalf("expected inbox invalidation event, got %q", response.Header().Get("HX-Trigger"))
	}
	if strings.Contains(response.Body.String(), "fileListRadioGroup") ||
		strings.Contains(response.Body.String(), "No files available yet.") {
		t.Fatalf("command returned replacement inbox content")
	}
}

func TestMarkAsDoneCommandIgnoresLegacyQueryDispatcherHeader(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "legacy-query-header")
	data := url.Values{"FileID": {fixture.fileID}}
	req := httptest.NewRequest(
		http.MethodPost, h.actions.Inbox.MarkAsDoneCmd.Endpoint(), strings.NewReader(data.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.Inbox(
		fixture.tenant.PublicID.String(), fixture.spaceID, fixture.fileID,
	))
	req.Header.Set("X-Query-Endpoint", h.actions.Inbox.InboxPage.Endpoint())
	req.Header.Set("X-Query-Data", "FileID="+url.QueryEscape(fixture.fileID))
	req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	h.router.ServeHTTP(response, req)
	if response.Code != http.StatusOK ||
		strings.Contains(response.Body.String(), "fileListRadioGroup") {
		t.Fatalf("legacy query header dispatched a query (status %d)", response.Code)
	}
}

func TestUpdateFileListPreferencesEmitsEventAndBrowseQueryRendersWrapper(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "browse-query-refresh")
	currentURL := route.Browse(fixture.tenant.PublicID.String(), fixture.spaceID, fixture.rootID)
	preference := fixture.browserAt(currentURL,
		h.actions.Browse.UpdateFileListPreferencesCmd.Endpoint(), url.Values{"ViewMode": {"table"}})
	if preference.Code != http.StatusOK ||
		preference.Header().Get("HX-Trigger") != event.FileListPreferencesUpdated.String() {
		t.Fatalf("preference command response: status=%d trigger=%q",
			preference.Code, preference.Header().Get("HX-Trigger"))
	}
	if strings.Contains(preference.Body.String(), `id="fileList"`) {
		t.Fatalf("preference command returned replacement list content: %s", preference.Body.String())
	}

	form := url.Values{"CurrentDirID": {fixture.rootID}, "SelectedFileID": {fixture.fileID}}
	request := httptest.NewRequest(
		http.MethodPost, h.actions.Browse.ListDirPartial.Endpoint(), strings.NewReader(form.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Current-URL", currentURL+"/file/"+fixture.fileID)
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	h.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("browse query returned status %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `id="listDirWrapper"`) {
		t.Fatalf("browse query did not render listDirWrapper root")
	}
	if !strings.Contains(response.Body.String(), fixture.fileID) {
		t.Fatalf("browse query did not preserve the selected file")
	}
}

func TestInboxQueryIgnoresLegacyUploadToken(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "query-upload-token")
	readCtx := privacy.DecisionContext(context.Background(), privacy.Allow)
	token := "staged-command-query-regression"
	expiresAt := time.Now().Add(time.Hour)
	temporary := h.mainDB.ReadWriteConn.TemporaryFile.Create().
		SetOwnerID(fixture.account.ID).
		SetFilename("staged.txt").
		SetSize(4).
		SetSizeInStorage(4).
		SetStorageType(storagetype.Unknown).
		SetStoragePath("staged-test-path").
		SetStorageFilename("staged-test-file").
		SetUploadToken(token).
		SetUploadSucceededAt(time.Now()).
		SetExpiresAt(expiresAt).
		SaveX(context.Background())
	before := fixture.db.ReadOnlyConn.File.Query().CountX(readCtx)
	form := url.Values{}
	request := httptest.NewRequest(
		http.MethodPost, h.actions.Inbox.InboxPage.Endpoint(), strings.NewReader(form.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Current-URL", route.Inbox(
		fixture.tenant.PublicID.String(), fixture.spaceID, fixture.fileID,
	)+"?upload_token="+token)
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	h.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("inbox query returned status %d", response.Code)
	}
	after := fixture.db.ReadOnlyConn.File.Query().CountX(readCtx)
	if after != before {
		t.Fatalf("Inbox query changed file rows: before=%d after=%d", before, after)
	}
	currentTemporary := h.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
		temporaryfile.ID(temporary.ID), temporaryfile.OwnerID(fixture.account.ID),
	).OnlyX(mainprivacy.DecisionContext(context.Background(), mainprivacy.Allow))
	if currentTemporary.ConvertedToStoredFileAt != nil {
		t.Fatalf("Inbox query consumed staged upload token %q", token)
	}
	get := httptest.NewRequest(http.MethodGet,
		route.InboxRoot(fixture.tenant.PublicID.String(), fixture.spaceID)+"?upload_token="+token, nil)
	get.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	h.router.RegisterPage(route.InboxRoute(false, false), h.actions.Inbox.InboxRootPage.Handler)
	getResponse := httptest.NewRecorder()
	h.router.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("Inbox GET returned status %d", getResponse.Code)
	}
	currentTemporary = h.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
		temporaryfile.ID(temporary.ID), temporaryfile.OwnerID(fixture.account.ID),
	).OnlyX(mainprivacy.DecisionContext(context.Background(), mainprivacy.Allow))
	if currentTemporary.ConvertedToStoredFileAt != nil {
		t.Fatalf("Inbox GET consumed staged upload token %q", token)
	}
}

func TestRegisteredCommandBuffersResponseUntilCommit(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "registered-buffering")
	request := httptest.NewRequest(http.MethodPost,
		h.actions.Browse.UpdateFileListPreferencesCmd.Endpoint(),
		strings.NewReader(url.Values{"ViewMode": {"table"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Current-URL", route.Browse(
		fixture.tenant.PublicID.String(), fixture.spaceID, fixture.rootID))
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	observer := &commitObservationWriter{ResponseRecorder: response, observe: func() bool {
		fresh := h.mainDB.ReadOnlyConn.Account.Query().Where(account.ID(fixture.account.ID)).OnlyX(
			privacy.DecisionContext(context.Background(), privacy.Allow),
		)
		return fresh.FileListPreferences.ViewMode == "table"
	}}
	h.router.ServeHTTP(observer, request)
	if response.Code != http.StatusOK || !observer.committedAtWrite {
		t.Fatalf("response status=%d; table preference committed at first write=%t",
			response.Code, observer.committedAtWrite)
	}
}

func TestBufferedCommandSuppressesSuccessFeedbackOnHandlerError(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "handler-error-buffer")
	cmd := h.actions.Browse.UpdateFileListPreferencesCmd
	wrapped := h.router.wrapTx(h.router.wrapCommand(func(
		rw httpx.ResponseWriter, _ *httpx.Request, ctx ctxx.Context,
	) error {
		ctx.MainCtx().MainTx.Account.UpdateOneID(fixture.account.ID).
			SetFirstName("should roll back").
			SaveX(ctx)
		rw.Header().Set("HX-Trigger", "must-not-escape")
		rw.Header().Set("HX-Location", "/must-not-escape")
		if err := h.infra.Renderer().Render(
			rw, ctx, widget.NewSnackbarf("Injected success snackbar"),
		); err != nil {
			return err
		}
		return errors.New("injected handler failure")
	}), false)
	request := httptest.NewRequest(http.MethodPost, cmd.Endpoint(), strings.NewReader(""))
	request.Header.Set("HX-Request", "true")
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	wrapped(response, request)
	assertNoSuccessResponse(t, response)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("handler error returned status %d", response.Code)
	}
	fresh := h.mainDB.ReadOnlyConn.Account.Query().Where(account.ID(fixture.account.ID)).OnlyX(
		privacy.DecisionContext(context.Background(), privacy.Allow),
	)
	if fresh.FirstName != fixture.account.FirstName {
		t.Fatalf("handler error did not roll back account mutation")
	}
}

func assertNoSuccessResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Header().Get("HX-Trigger") != "" || response.Header().Get("HX-Location") != "" ||
		strings.Contains(response.Body.String(), "Injected success snackbar") {
		t.Fatalf("released success response: headers=%v", response.Header())
	}
}
