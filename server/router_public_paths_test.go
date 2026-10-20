package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestRouterPublicAndVisitorOnlyPagesAreDeclaredAtRegistration(t *testing.T) {
	harness := newActionTestHarness(t)
	email := "public-paths@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)
	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	noContent := func(rw httpx.ResponseWriter, _ *httpx.Request, _ ctxx.Context) error {
		rw.WriteHeader(http.StatusNoContent)
		return nil
	}
	harness.router.RegisterPublicPage("GET /test-public/", noContent)
	harness.router.RegisterVisitorOnlyPage("GET /test-visitor-only/", noContent)
	harness.router.RegisterPage("GET /test-private/", noContent)

	for _, tc := range []struct {
		name         string
		path         string
		isSignedIn   bool
		wantStatus   int
		wantLocation string
	}{
		{"public page for visitor", "/test-public/", false, http.StatusNoContent, ""},
		{"public page for user", "/test-public/", true, http.StatusNoContent, ""},
		{"visitor-only page for visitor", "/test-visitor-only/", false, http.StatusNoContent, ""},
		{
			"visitor-only page for user",
			"/test-visitor-only/",
			true,
			http.StatusSeeOther,
			route.Dashboard(),
		},
		{"private page for visitor", "/test-private/", false, http.StatusSeeOther, "/"},
		{"private page for user", "/test-private/", true, http.StatusNoContent, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.isSignedIn {
				req.AddCookie(sessionCookie)
			}
			rr := httptest.NewRecorder()
			harness.router.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if location := rr.Header().Get("Location"); location != tc.wantLocation {
				t.Fatalf("location = %q, want %q", location, tc.wantLocation)
			}
		})
	}

	// The empty sign-in form fails, but a visitor must reach the public command.
	req := httptest.NewRequest(http.MethodPost, harness.actions.Auth.SignInCmd.Endpoint(), nil)
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)
	if rr.Code == http.StatusSeeOther {
		t.Fatalf("public action redirected visitor to %q", rr.Header().Get("Location"))
	}
}
