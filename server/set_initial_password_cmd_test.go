package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/db/entmain/session"
	accountmodel "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/util/cookiex"
)

// The temporary password may have been intercepted, so setting the initial password also
// signs out all other sessions.
func TestSetInitialPasswordCmdOnlySetsMissingPassword(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		hasPassword           bool
		wantStatus            int
		wantPassword          string
		wantNotStored         string
		wantOtherSessionAlive bool
	}{
		{
			name:                  "existing password requires change password flow",
			hasPassword:           true,
			wantStatus:            http.StatusBadRequest,
			wantPassword:          "supersecret",
			wantNotStored:         "new-password-123",
			wantOtherSessionAlive: true,
		},
		{
			name:                  "temporary password account sets initial password",
			hasPassword:           false,
			wantStatus:            http.StatusOK,
			wantPassword:          "new-password-123",
			wantOtherSessionAlive: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			harness := newActionTestHarness(t)
			accountx, _ := signUpAccount(t, harness, "initial-password@example.com")
			setAccountPasswordForTest(t, harness, accountx.ID, "supersecret")
			if !tc.hasPassword {
				harness.mainDB.ReadWriteConn.Account.UpdateOneID(accountx.ID).
					SetPasswordSalt("").
					SetPasswordHash("").
					SetTemporaryPasswordSalt("temporary-salt").
					SetTemporaryPasswordHash("temporary-hash").
					SetTemporaryPasswordExpiresAt(time.Now().Add(time.Hour)).
					ExecX(context.Background())
			}
			sessionValue := createSessionForAccountForRulesTest(t, harness, accountx.ID)
			otherSessionValue := createSessionForAccountForRulesTest(t, harness, accountx.ID)

			form := url.Values{}
			form.Set("NewPassword", "new-password-123")
			form.Set("ConfirmPassword", "new-password-123")
			req := httptest.NewRequest(
				http.MethodPost,
				harness.actions.Auth.SetInitialPasswordCmd.Endpoint(),
				strings.NewReader(form.Encode()),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: sessionValue})
			rr := httptest.NewRecorder()
			harness.router.ServeHTTP(rr, req)

			assertInitialPasswordCommandOutcome(
				t,
				harness,
				accountx.ID,
				rr,
				tc.wantStatus,
				tc.wantPassword,
				tc.wantNotStored,
				sessionValue,
				otherSessionValue,
				tc.wantOtherSessionAlive,
			)
		})
	}
}

func assertInitialPasswordCommandOutcome(
	t *testing.T,
	harness *actionTestHarness,
	accountID int64,
	rr *httptest.ResponseRecorder,
	wantStatus int,
	wantPassword string,
	wantNotStored string,
	sessionValue string,
	otherSessionValue string,
	wantOtherSessionAlive bool,
) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d: %s", rr.Code, wantStatus, rr.Body.String())
	}
	accountAfter := accountmodel.NewAccount(
		harness.mainDB.ReadWriteConn.Account.GetX(context.Background(), accountID),
	)
	if !accountAfter.IsPasswordValid(nil, wantPassword) {
		t.Fatalf("expected password %q to be valid", wantPassword)
	}
	if wantNotStored != "" && accountAfter.IsPasswordValid(nil, wantNotStored) {
		t.Fatalf("password %q must not replace the existing password", wantNotStored)
	}
	if !initialPasswordSessionExists(harness, sessionValue) {
		t.Fatal("expected current session to stay signed in")
	}
	if initialPasswordSessionExists(harness, otherSessionValue) != wantOtherSessionAlive {
		t.Fatalf("other session alive = %v, want %v", !wantOtherSessionAlive, wantOtherSessionAlive)
	}
}

func initialPasswordSessionExists(harness *actionTestHarness, value string) bool {
	return harness.mainDB.ReadWriteConn.Session.Query().
		Where(session.Value(value)).ExistX(context.Background())
}
