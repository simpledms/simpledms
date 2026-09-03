package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
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
