package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsTWARequest(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if isTWARequest(nil) {
			t.Fatal("expected nil request to not be treated as TWA")
		}
	})

	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{
			name: "x-is-twa true",
			headers: map[string]string{
				"X-Is-TWA": "true",
			},
			want: true,
		},
		{
			name: "x-is-twa numeric true",
			headers: map[string]string{
				"X-Is-TWA": "1",
			},
			want: true,
		},
		{
			name: "android app referer",
			headers: map[string]string{
				"Referer": "android-app://com.example.twa",
			},
			want: true,
		},
		{
			name: "non twa request",
			headers: map[string]string{
				"Referer": "https://app.simpledms.ch/",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://app.simpledms.ch/", nil)
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}

			got := isTWARequest(req)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
