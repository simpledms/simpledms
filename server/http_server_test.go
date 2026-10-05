package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPublicHandlerDeniesCrossOriginFramingAndSniffing(t *testing.T) {
	handler := newPublicHandler(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rr.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
}

// Without header and idle timeouts, slow or idle clients keep connections open
// indefinitely (Slowloris). Body timeouts are intentionally unset for large transfers.
func TestNewHTTPServerBoundsHeaderAndIdleTimeButNotBodies(t *testing.T) {
	server := newHTTPServer(":0", http.NotFoundHandler())

	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("expected a read header timeout")
	}
	if server.IdleTimeout <= 0 {
		t.Fatal("expected an idle timeout; zero falls back to the unset ReadTimeout")
	}
	if server.ReadTimeout != 0 || server.WriteTimeout != 0 {
		t.Fatal("body timeouts would abort large uploads and downloads")
	}
}
