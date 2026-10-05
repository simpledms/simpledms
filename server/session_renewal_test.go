package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

// The main DB has a single read-write connection and a write request's transaction holds it,
// so renewing the session must not wait for another read-write connection.
func TestWriteCommandRenewsDueSessionWithinItsTransaction(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "session-renewal-write")
	decisionCtx := mainprivacy.DecisionContext(context.Background(), mainprivacy.Allow)
	dueExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	h.mainDB.ReadWriteConn.Session.Update().
		Where(session.Value(fixture.session)).
		SetExpiresAt(dueExpiresAt).
		SetDeletableAt(dueExpiresAt).
		ExecX(decisionCtx)

	requestCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request := httptest.NewRequest(
		http.MethodPost,
		h.actions.Browse.UpdateFileListPreferencesCmd.Endpoint(),
		strings.NewReader(url.Values{"ViewMode": {"table"}}.Encode()),
	).WithContext(requestCtx)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Current-URL", route.Browse(
		fixture.tenant.PublicID.String(), fixture.spaceID, fixture.rootID))
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	h.router.ServeHTTP(response, request)

	if requestCtx.Err() != nil {
		t.Fatalf("write command waited for the read-write connection: %v", requestCtx.Err())
	}
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	renewed := h.mainDB.ReadOnlyConn.Session.Query().
		Where(session.Value(fixture.session)).
		OnlyX(decisionCtx)
	if !renewed.ExpiresAt.After(time.Now().Add(13 * 24 * time.Hour)) {
		t.Fatalf("expected renewed session expiry, got %s", renewed.ExpiresAt)
	}
}
