package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestChangePasswordCmdSignsOutOtherSessions(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, _ := signUpAccount(t, harness, "change-password-sessions@example.com")
	setAccountPasswordForTest(t, harness, accountx.ID, "supersecret")
	currentSession := createSessionForAccountForRulesTest(t, harness, accountx.ID)
	otherSession := createSessionForAccountForRulesTest(t, harness, accountx.ID)

	form := url.Values{}
	form.Set("CurrentOrTemporaryPassword", "supersecret")
	form.Set("NewPassword", "new-password-123")
	form.Set("ConfirmPassword", "new-password-123")
	req := httptest.NewRequest(
		http.MethodPost,
		harness.actions.Auth.ChangePasswordCmd.Endpoint(),
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: currentSession})
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rr.Code, rr.Body.String())
	}
	sessionExists := func(value string) bool {
		return harness.mainDB.ReadWriteConn.Session.Query().
			Where(session.Value(value)).
			ExistX(context.Background())
	}
	if !sessionExists(currentSession) {
		t.Fatal("expected current session to stay signed in")
	}
	if sessionExists(otherSession) {
		t.Fatal("expected other session to be signed out")
	}
}
