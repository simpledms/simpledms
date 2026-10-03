package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	accountmodel "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestSetInitialPasswordCmdOnlySetsMissingPassword(t *testing.T) {
	for _, tc := range []struct {
		name          string
		hasPassword   bool
		wantStatus    int
		wantPassword  string
		wantNotStored string
	}{
		{
			name:          "existing password requires change password flow",
			hasPassword:   true,
			wantStatus:    http.StatusBadRequest,
			wantPassword:  "supersecret",
			wantNotStored: "new-password-123",
		},
		{
			name:         "temporary password account sets initial password",
			hasPassword:  false,
			wantStatus:   http.StatusOK,
			wantPassword: "new-password-123",
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

			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}
			accountAfter := accountmodel.NewAccount(
				harness.mainDB.ReadWriteConn.Account.GetX(context.Background(), accountx.ID),
			)
			if !accountAfter.IsPasswordValid(nil, tc.wantPassword) {
				t.Fatalf("expected password %q to be valid", tc.wantPassword)
			}
			if tc.wantNotStored != "" && accountAfter.IsPasswordValid(nil, tc.wantNotStored) {
				t.Fatalf("password %q must not replace the existing password", tc.wantNotStored)
			}
		})
	}
}
