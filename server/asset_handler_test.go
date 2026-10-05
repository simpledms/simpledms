package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/simpledms/simpledms/ui"
	"github.com/simpledms/simpledms/ui/uix"
)

const immutableCacheControl = "public, max-age=31536000, immutable"

func newAssetHandlerForTest(assetsFS fs.FS) *AssetHandler {
	return NewAssetHandler(assetsFS, ui.NewAssetVersions(assetsFS))
}

func newEmbeddedAssetVersionsForTest(t testing.TB) *ui.AssetVersions {
	t.Helper()
	assetsFS, err := uix.NewAssetsFS()
	if err != nil {
		t.Fatal(err)
	}
	return ui.NewAssetVersions(assetsFS)
}

func TestAssetHandlerAnswersUnchangedAssetsWithNotModified(t *testing.T) {
	handler := newAssetHandlerForTest(fstest.MapFS{
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
		handler := newAssetHandlerForTest(fstest.MapFS{
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
	handler := newAssetHandlerForTest(fstest.MapFS{})

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/missing.js", nil))
	if rw.Code != http.StatusNotFound || rw.Header().Get("ETag") != "" {
		t.Fatalf("status %d, ETag %q", rw.Code, rw.Header().Get("ETag"))
	}
}

func TestAssetHandlerCachesContentVersionedURLsImmutably(t *testing.T) {
	assetsFS := fstest.MapFS{
		"app.css":                  &fstest.MapFile{Data: []byte("a{}")},
		"vendor/chunk-7MEJP3IB.js": &fstest.MapFile{Data: []byte("export{}")},
	}
	assetVersions := ui.NewAssetVersions(assetsFS)
	handler := NewAssetHandler(assetsFS, assetVersions)
	cacheControlFor := func(target string) string {
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, target, nil))
		if rw.Code != http.StatusOK {
			t.Fatalf("%s: status %d", target, rw.Code)
		}
		return rw.Header().Get("Cache-Control")
	}

	versionedURL := strings.TrimPrefix(assetVersions.URL("/assets/app.css"), "/assets")
	if !strings.Contains(versionedURL, "?v=") {
		t.Fatalf("expected a content version in %q", versionedURL)
	}
	if got := cacheControlFor(versionedURL); got != immutableCacheControl {
		t.Fatalf("versioned asset: Cache-Control = %q", got)
	}
	// HTML of a previous deployment may still request an outdated version.
	if got := cacheControlFor("/app.css?v=outdated"); got != "no-cache" {
		t.Fatalf("outdated version: Cache-Control = %q", got)
	}
	if got := cacheControlFor("/vendor/chunk-7MEJP3IB.js"); got != immutableCacheControl {
		t.Fatalf("content-hashed chunk: Cache-Control = %q", got)
	}
}
