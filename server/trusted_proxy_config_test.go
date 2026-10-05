package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/util/httpx"
)

func TestValidateTrustedProxyPublicOrigin(t *testing.T) {
	for _, tc := range []struct {
		hasTrustedProxies bool
		publicOrigin      string
		wantError         bool
	}{
		{hasTrustedProxies: false},
		{hasTrustedProxies: true, publicOrigin: "http://example.com", wantError: true},
		{hasTrustedProxies: true, publicOrigin: "https://example.com"},
	} {
		err := validateTrustedProxyPublicOrigin(tc.hasTrustedProxies, tc.publicOrigin)
		if (err != nil) != tc.wantError {
			t.Fatalf("trusted=%v origin=%q: got %v", tc.hasTrustedProxies, tc.publicOrigin, err)
		}
	}
}

func TestRouterTrustsForwardedHTTPSOnlyFromConfiguredProxy(t *testing.T) {
	router := NewRouter(
		nil,
		nil,
		nil,
		false,
		"",
		nil,
		[]netip.Prefix{netip.MustParsePrefix("192.0.2.10/32")},
	)
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Scheme != "https" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.WriteHeader(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name       string
		remoteAddr string
		proto      string
		wantStatus int
	}{
		{
			name:       "trusted HTTPS",
			remoteAddr: "192.0.2.10:1234",
			proto:      "https",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "untrusted HTTPS",
			remoteAddr: "198.51.100.10:1234",
			proto:      "https",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "trusted HTTP",
			remoteAddr: "192.0.2.10:1234",
			proto:      "http",
			wantStatus: http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("X-Forwarded-Proto", tc.proto)
			rw := httptest.NewRecorder()

			router.ServeHTTP(rw, req)

			if rw.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rw.Code)
			}
		})
	}
}

func TestRouterResolvesClientIPOnlyThroughConfiguredProxies(t *testing.T) {
	router := NewRouter(
		nil,
		nil,
		nil,
		false,
		"",
		nil,
		[]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
	)
	var gotClientIP string
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		gotClientIP = httpx.NewRequest(req).ClientIP()
		rw.WriteHeader(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name         string
		remoteAddr   string
		forwardedFor string
		wantClientIP string
	}{
		{
			name:         "direct client ignores forwarded header",
			remoteAddr:   "198.51.100.10:1234",
			forwardedFor: "203.0.113.5",
			wantClientIP: "198.51.100.10",
		},
		{
			name:         "trusted proxy uses rightmost untrusted address",
			remoteAddr:   "192.0.2.10:1234",
			forwardedFor: "203.0.113.99, 203.0.113.5, 192.0.2.11",
			wantClientIP: "203.0.113.5",
		},
		{
			name:         "trusted proxy without forwarded header",
			remoteAddr:   "192.0.2.10:1234",
			wantClientIP: "192.0.2.10",
		},
		{
			name:         "trusted proxy with malformed forwarded header",
			remoteAddr:   "192.0.2.10:1234",
			forwardedFor: "not-an-ip",
			wantClientIP: "192.0.2.10",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.forwardedFor)
			}
			router.ServeHTTP(httptest.NewRecorder(), req)

			if gotClientIP != tc.wantClientIP {
				t.Fatalf("client IP = %q, want %q", gotClientIP, tc.wantClientIP)
			}
		})
	}
}

// Behind a reverse proxy, every request arrives from the proxy address. Rate limits must
// use the forwarded client address, otherwise one client could lock out all sign-ins.
func TestAuthRateLimitsArePerClientBehindTrustedProxy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		newRequest func(harness *actionTestHarness, attempt int) *http.Request
	}{
		{
			name: "password sign-in",
			newRequest: func(harness *actionTestHarness, attempt int) *http.Request {
				form := url.Values{}
				// Distinct emails keep the per-email limit out of this test.
				form.Set("Email", fmt.Sprintf("unknown-%d@example.com", attempt))
				form.Set("Password", "wrong-password")
				req := httptest.NewRequest(
					http.MethodPost,
					harness.actions.Auth.SignInCmd.Endpoint(),
					strings.NewReader(form.Encode()),
				)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
		},
		{
			name: "passkey sign-in",
			newRequest: func(harness *actionTestHarness, _ int) *http.Request {
				req := httptest.NewRequest(
					http.MethodPost,
					"http://localhost"+harness.actions.Auth.PasskeySignInBeginCmd.Endpoint(),
					nil,
				)
				req.Host = "localhost"
				return req
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "http://localhost")
			t.Setenv("SIMPLEDMS_WEBAUTHN_RP_ID", "localhost")
			harness := newActionTestHarness(t)
			router := NewRouter(
				harness.mainDB,
				harness.tenantDBs,
				harness.infra,
				true,
				harness.metaPath,
				harness.i18n,
				[]netip.Prefix{netip.MustParsePrefix("192.0.2.10/32")},
			)
			router.RegisterActions(harness.actions)

			send := func(clientIP string, attempt int) int {
				req := tc.newRequest(harness, attempt)
				req.RemoteAddr = "192.0.2.10:1234"
				req.Header.Set("X-Forwarded-For", clientIP)
				req.Header.Set("HX-Request", "true")
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)
				return rr.Code
			}

			limited := false
			for attempt := 0; attempt < 100 && !limited; attempt++ {
				limited = send("203.0.113.1", attempt) == http.StatusTooManyRequests
			}
			if !limited {
				t.Fatal("expected the attacking client to be rate limited")
			}
			if send("203.0.113.2", 0) == http.StatusTooManyRequests {
				t.Fatal("another client behind the same proxy must not share the rate limit")
			}
		})
	}
}
