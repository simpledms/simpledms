package server

import "testing"

func TestShouldUseAutocert(t *testing.T) {
	testCases := []struct {
		name           string
		enableAutocert bool
		devMode        bool
		expected       bool
	}{
		{name: "enabled in prod", enableAutocert: true, devMode: false, expected: true},
		{name: "enabled in dev", enableAutocert: true, devMode: true, expected: false},
		{name: "disabled in prod", enableAutocert: false, devMode: false, expected: false},
		{name: "disabled in dev", enableAutocert: false, devMode: true, expected: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual := shouldUseAutocert(tc.enableAutocert, tc.devMode)
			if actual != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestResolveListenMode(t *testing.T) {
	testCases := []struct {
		name        string
		useAutocert bool
		cert        string
		key         string
		expected    listenMode
	}{
		{name: "autocert has priority", useAutocert: true, cert: "cert.pem", key: "key.pem", expected: listenModeTLSAutocert},
		{name: "http without cert and key", useAutocert: false, cert: "", key: "", expected: listenModeHTTP},
		{name: "http with cert only", useAutocert: false, cert: "cert.pem", key: "", expected: listenModeHTTP},
		{name: "http with key only", useAutocert: false, cert: "", key: "key.pem", expected: listenModeHTTP},
		{name: "tls with cert and key", useAutocert: false, cert: "cert.pem", key: "key.pem", expected: listenModeTLSFiles},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual := resolveListenMode(tc.useAutocert, tc.cert, tc.key)
			if actual != tc.expected {
				t.Fatalf("expected mode %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestListenFlowMatrix(t *testing.T) {
	testCases := []struct {
		name       string
		isLocked   bool
		enableAuto bool
		devMode    bool
		cert       string
		key        string
		expectAuto bool
		expectMode listenMode
	}{
		{
			name:       "unlocked autocert",
			isLocked:   false,
			enableAuto: true,
			devMode:    false,
			expectAuto: true,
			expectMode: listenModeTLSAutocert,
		},
		{
			name:       "unlocked plain http",
			isLocked:   false,
			enableAuto: false,
			devMode:    false,
			expectAuto: false,
			expectMode: listenModeHTTP,
		},
		{
			name:       "unlocked tls files",
			isLocked:   false,
			enableAuto: false,
			devMode:    false,
			cert:       "cert.pem",
			key:        "key.pem",
			expectAuto: false,
			expectMode: listenModeTLSFiles,
		},
		{
			name:       "locked autocert",
			isLocked:   true,
			enableAuto: true,
			devMode:    false,
			expectAuto: true,
			expectMode: listenModeTLSAutocert,
		},
		{
			name:       "locked plain http",
			isLocked:   true,
			enableAuto: false,
			devMode:    false,
			expectAuto: false,
			expectMode: listenModeHTTP,
		},
		{
			name:       "locked tls files",
			isLocked:   true,
			enableAuto: false,
			devMode:    false,
			cert:       "cert.pem",
			key:        "key.pem",
			expectAuto: false,
			expectMode: listenModeTLSFiles,
		},
		{
			name:       "autocert disabled in dev",
			isLocked:   true,
			enableAuto: true,
			devMode:    true,
			expectAuto: false,
			expectMode: listenModeHTTP,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			useAutocert := shouldUseAutocert(tc.enableAuto, tc.devMode)
			if useAutocert != tc.expectAuto {
				t.Fatalf("expected useAutocert=%v, got %v", tc.expectAuto, useAutocert)
			}

			mode := resolveListenMode(useAutocert, tc.cert, tc.key)
			if mode != tc.expectMode {
				t.Fatalf("expected mode %v, got %v", tc.expectMode, mode)
			}

			if tc.isLocked {
				if mode != tc.expectMode {
					t.Fatalf("maintenance mode should match main mode, got %v", mode)
				}
			}
		})
	}
}

func TestServerPortSelection(t *testing.T) {
	testCases := []struct {
		name              string
		unsafePort        int
		useAutocert       bool
		tlsCertFilepath   string
		tlsPrivateKeypath string
		expected          int
	}{
		{name: "unsafe port overrides http", unsafePort: 18080, expected: 18080},
		{name: "unsafe port overrides tls", unsafePort: 18443, useAutocert: true, expected: 18443},
		{name: "default http port", unsafePort: 0, useAutocert: false, expected: 80},
		{name: "default tls port from autocert", unsafePort: 0, useAutocert: true, expected: 443},
		{name: "default tls port from cert files", unsafePort: 0, tlsCertFilepath: "cert.pem", tlsPrivateKeypath: "key.pem", expected: 443},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{unsafePort: tc.unsafePort}
			actual := server.port(tc.useAutocert, tc.tlsCertFilepath, tc.tlsPrivateKeypath)
			if actual != tc.expected {
				t.Fatalf("expected port %d, got %d", tc.expected, actual)
			}
		})
	}
}
