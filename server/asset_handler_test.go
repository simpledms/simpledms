package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestAssetHandlerAnswersUnchangedAssetsWithNotModified(t *testing.T) {
	handler := NewAssetHandler(fstest.MapFS{
		"app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	})

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	etag := first.Header().Get("ETag")
	if first.Code != http.StatusOK || etag == "" {
		t.Fatalf("first request: status %d, ETag %q", first.Code, etag)
	}
	if cacheControl := first.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", cacheControl)
	}

	revalidationReq := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	revalidationReq.Header.Set("If-None-Match", etag)
	revalidation := httptest.NewRecorder()
	handler.ServeHTTP(revalidation, revalidationReq)
	if revalidation.Code != http.StatusNotModified {
		t.Fatalf("revalidation status = %d, want %d", revalidation.Code, http.StatusNotModified)
	}
}

func TestAssetHandlerChangesETagWithContent(t *testing.T) {
	etagFor := func(content string) string {
		handler := NewAssetHandler(fstest.MapFS{
			"app.css": &fstest.MapFile{Data: []byte(content)},
		})
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/app.css", nil))
		return rw.Header().Get("ETag")
	}

	if etagFor("a{}") == etagFor("b{}") {
		t.Fatal("different asset content must produce different ETags")
	}
}

func TestAssetHandlerLeavesMissingAssetsWithoutETag(t *testing.T) {
	handler := NewAssetHandler(fstest.MapFS{})

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/missing.js", nil))
	if rw.Code != http.StatusNotFound || rw.Header().Get("ETag") != "" {
		t.Fatalf("status %d, ETag %q", rw.Code, rw.Header().Get("ETag"))
	}
}
