package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

// Organization user management and settings are owner-only, not just hidden in navigation;
// the user list exposes the names and email addresses of all members.
func TestTenantManagementQueriesRequireTenantOwner(t *testing.T) {
	harness := newActionTestHarness(t)
	ownerAccount, tenantx := signUpAccount(t, harness, "tenant-query-owner@example.com")
	memberAccount := createTenantUser(
		t, harness, tenantx, "tenant-query-member@example.com", tenantrole.User,
	)
	initTenantDB(t, harness, tenantx)
	harness.router.RegisterPage(
		route.ManageUsersOfTenantRoute(),
		harness.actions.ManageTenantUsers.ManageUsersOfTenantPage.Handler,
	)
	harness.router.RegisterPage(
		route.OrganizationSettingsRoute(),
		harness.actions.Dashboard.OrganizationSettingsPage.Handler,
	)

	tenantID := tenantx.PublicID.String()
	usersURL := route.ManageUsersOfTenant(tenantID)
	requests := []struct {
		name   string
		method string
		url    string
	}{
		{name: "users page", method: http.MethodGet, url: usersURL},
		{
			name:   "user list partial",
			method: http.MethodPost,
			url:    harness.actions.ManageTenantUsers.UserListPartial.Endpoint(),
		},
		{name: "settings page", method: http.MethodGet, url: route.OrganizationSettings(tenantID)},
	}

	for _, tc := range []struct {
		name       string
		accountID  int64
		wantStatus int
	}{
		{name: "owner", accountID: ownerAccount.ID, wantStatus: http.StatusOK},
		{name: "member", accountID: memberAccount.ID, wantStatus: http.StatusForbidden},
	} {
		session := createSessionForAccountForRulesTest(t, harness, tc.accountID)
		for _, request := range requests {
			t.Run(tc.name+" "+request.name, func(t *testing.T) {
				req := httptest.NewRequest(request.method, request.url, nil)
				req.Header.Set("HX-Request", "true")
				req.Header.Set("HX-Current-URL", usersURL)
				req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
				rr := httptest.NewRecorder()
				harness.router.ServeHTTP(rr, req)

				if rr.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
				}
				if tc.wantStatus == http.StatusForbidden &&
					strings.Contains(rr.Body.String(), "tenant-query-owner@example.com") {
					t.Fatal("member response exposed the user list")
				}
			})
		}
	}
}
