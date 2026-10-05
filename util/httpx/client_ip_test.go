package httpx

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIPThroughTrustedProxiesIgnoresClientControlledEntries(t *testing.T) {
	trustedProxies := []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}

	for _, tc := range []struct {
		name         string
		remoteAddr   string
		forwardedFor []string
		want         string
	}{
		{
			name:         "untrusted peer ignores forwarded header",
			remoteAddr:   "198.51.100.10:1234",
			forwardedFor: []string{"203.0.113.5"},
			want:         "198.51.100.10",
		},
		{
			name:         "rightmost untrusted entry wins",
			remoteAddr:   "192.0.2.10:1234",
			forwardedFor: []string{"203.0.113.99, 203.0.113.5, 192.0.2.11"},
			want:         "203.0.113.5",
		},
		{
			name:         "proxy adds its own header line after the spoofed one",
			remoteAddr:   "192.0.2.10:1234",
			forwardedFor: []string{"203.0.113.99", "203.0.113.5"},
			want:         "203.0.113.5",
		},
		{
			name:         "entries with ports",
			remoteAddr:   "192.0.2.10:1234",
			forwardedFor: []string{"203.0.113.5:5555, [2001:db8::1]:443"},
			want:         "2001:db8::1",
		},
		{
			name:         "IPv4-mapped trusted proxy entry",
			remoteAddr:   "[::ffff:192.0.2.10]:1234",
			forwardedFor: []string{"203.0.113.5, ::ffff:192.0.2.11"},
			want:         "203.0.113.5",
		},
		{
			name:         "malformed chain falls back to proxy",
			remoteAddr:   "192.0.2.10:1234",
			forwardedFor: []string{"not-an-ip"},
			want:         "192.0.2.10",
		},
		{
			name:       "trusted peer without forwarded header",
			remoteAddr: "192.0.2.10:1234",
			want:       "192.0.2.10",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			for _, value := range tc.forwardedFor {
				req.Header.Add("X-Forwarded-For", value)
			}

			got, ok := ClientIPThroughTrustedProxies(req, trustedProxies)
			if !ok || got.String() != tc.want {
				t.Fatalf("client IP = %q (%v), want %q", got, ok, tc.want)
			}
		})
	}
}
