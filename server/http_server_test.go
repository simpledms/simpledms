package server

import (
	"io"
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

func TestNewPublicHandlerAbortsStartedResponseOnErrAbortHandler(t *testing.T) {
	server := httptest.NewServer(newPublicHandler(http.HandlerFunc(
		func(rw http.ResponseWriter, _ *http.Request) {
			rw.Header().Set("Content-Type", "application/zip")
			rw.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(rw, "partial archive")
			if err := http.NewResponseController(rw).Flush(); err != nil {
				t.Errorf("flush: %v", err)
			}
			panic(http.ErrAbortHandler)
		},
	)))
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected started response with status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("expected aborted response body, got a complete response")
	}
}

func TestNewPublicHandlerRecoversOtherPanicsWithInternalServerError(t *testing.T) {
	handler := newPublicHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("unexpected")
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
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
