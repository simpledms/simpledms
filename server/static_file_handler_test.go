package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func serveStaticFileForTest(target string) *httptest.ResponseRecorder {
	handler := http.StripPrefix("/static/", NewStaticFileHandler(fstest.MapFS{
		"app.js":             &fstest.MapFile{Data: []byte("console.log(1)")},
		"vendor/chunk.js":    &fstest.MapFile{Data: []byte("export{}")},
		"vendor/deep/lib.js": &fstest.MapFile{Data: []byte("export{}")},
	}))
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, target, nil))
	return rw
}

func TestStaticFileHandlerAnswersDirectoriesLikeMissingFiles(t *testing.T) {
	missing := serveStaticFileForTest("/static/missing.js")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing file: status %d", missing.Code)
	}

	targets := []string{
		"/static/",
		"/static/vendor",
		"/static/vendor/",
		"/static/vendor/deep/",
		"/static/vendor/deep/..",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			rw := serveStaticFileForTest(target)
			if rw.Code != http.StatusNotFound || rw.Body.String() != missing.Body.String() {
				t.Fatalf("status %d, body %q", rw.Code, rw.Body.String())
			}
			if robotsTag := rw.Header().Get("X-Robots-Tag"); robotsTag != "noindex" {
				t.Fatalf("X-Robots-Tag = %q, want noindex", robotsTag)
			}
		})
	}
}

func TestStaticFileHandlerServesFilesMarkedNoIndex(t *testing.T) {
	for _, target := range []string{"/static/app.js", "/static/vendor/chunk.js"} {
		t.Run(target, func(t *testing.T) {
			rw := serveStaticFileForTest(target)
			if rw.Code != http.StatusOK || rw.Body.Len() == 0 {
				t.Fatalf("status %d, body %q", rw.Code, rw.Body.String())
			}
			if robotsTag := rw.Header().Get("X-Robots-Tag"); robotsTag != "noindex" {
				t.Fatalf("X-Robots-Tag = %q, want noindex", robotsTag)
			}
		})
	}
}
