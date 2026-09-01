package server

import "testing"

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
