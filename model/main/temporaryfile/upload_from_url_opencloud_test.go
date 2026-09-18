package temporaryfile

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOpenCloudURLSourceUsesConfiguredOriginAndPassword(t *testing.T) {
	const password = "Shared-secret-1!"
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		username, receivedPassword, ok := req.BasicAuth()
		if !ok || username != "public" || receivedPassword != password {
			t.Errorf("unexpected OpenCloud public-link credentials")
			http.Error(rw, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if req.URL.Path != "/remote.php/dav/public-files/random_token/report.pdf" {
			t.Errorf("unexpected request path %q", req.URL.Path)
			http.Error(rw, "invalid path", http.StatusBadRequest)
			return
		}
		if req.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Error("missing X-Requested-With header")
			http.Error(rw, "missing header", http.StatusBadRequest)
			return
		}
		rw.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		_, _ = io.WriteString(rw, "document")
	}))
	t.Cleanup(server.Close)

	service := NewUploadFromURLService(nil, true, server.URL, password)
	filename, body, err := service.downloadFile(
		context.Background(),
		server.URL+"/remote.php/dav/public-files/random_token/report.pdf",
		OpenCloudURLSource,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if filename != "report.pdf" || string(content) != "document" {
		t.Fatalf("download = %q, %q", filename, content)
	}
}

func TestOpenCloudURLSourceRejectsOtherOriginsAndRedirects(t *testing.T) {
	var redirected atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/redirected.pdf") {
			redirected.Store(true)
			_, _ = io.WriteString(rw, "secret")
			return
		}
		http.Redirect(rw, req, "/remote.php/dav/public-files/random_token/redirected.pdf", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	service := NewUploadFromURLService(nil, true, server.URL, "Shared-secret-1!")
	if _, err := service.ValidateURLForSource(
		"https://evil.example/remote.php/dav/public-files/random_token/report.pdf",
		OpenCloudURLSource,
	); err == nil {
		t.Fatal("expected another origin to be rejected")
	}
	_, body, err := service.downloadFile(
		context.Background(),
		server.URL+"/remote.php/dav/public-files/random_token/report.pdf",
		OpenCloudURLSource,
	)
	if body != nil {
		_ = body.Close()
	}
	if err == nil {
		t.Fatal("expected an OpenCloud redirect to be rejected")
	}
	if redirected.Load() {
		t.Fatal("OpenCloud redirect was followed")
	}
}
