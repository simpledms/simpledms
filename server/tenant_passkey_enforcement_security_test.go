package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
)

func TestToggleTenantPasskeyEnforcementRequiresUnexpiredTargetOwnership(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expiresIn  time.Duration
		wantStatus int
	}{
		{
			name:       "expired owner with active membership elsewhere",
			expiresIn:  -time.Hour,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "owner whose assignment expires in the future",
			expiresIn:  time.Hour,
			wantStatus: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			harness := newActionTestHarness(t)
			const email = "passkey-policy-owner@example.com"
			const password = "supersecret"
			createAccount(t, harness.mainDB, email, password)
			cookie := signInAndGetSessionCookie(t, harness, email, password)
			ctx := context.Background()
			actor := harness.mainDB.ReadWriteConn.Account.Query().
				Where(account.Email(entx.NewCIText(email))).OnlyX(ctx)
			target := createTenantWithPasskeyPolicy(t, harness.mainDB, "Target organization", true, true)
			harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
				SetTenantID(target.ID).
				SetAccountID(actor.ID).
				SetRole(tenantrole.Owner).
				SetExpiresAt(time.Now().Add(tc.expiresIn)).
				SaveX(ctx)

			form := url.Values{
				"TenantID":        {target.PublicID.String()},
				"EnforcePasskeys": {"false"},
			}
			req := httptest.NewRequest(
				http.MethodPost,
				harness.actions.Dashboard.ToggleTenantPasskeyEnforcementCmd.Endpoint(),
				strings.NewReader(form.Encode()),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			req.Header.Set("HX-Current-URL", "https://example.com/dashboard/")
			req.AddCookie(cookie)
			rr := httptest.NewRecorder()
			harness.router.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}

			updated := harness.mainDB.ReadWriteConn.Tenant.GetX(ctx, target.ID)
			wantEnforced := tc.wantStatus == http.StatusForbidden
			if updated.PasskeyAuthEnforced != wantEnforced {
				t.Fatalf("passkey enforcement = %t, want %t", updated.PasskeyAuthEnforced, wantEnforced)
			}
			if wantEnforced && rr.Header().Get("HX-Trigger") != "" {
				t.Fatal("rejected policy change emitted a success event")
			}
		})
	}
}
