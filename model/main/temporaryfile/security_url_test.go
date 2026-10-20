package temporaryfile

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/e"
)

func TestURLImportIgnoresProxyEnvironmentAndValidatesOpenCloudPaths(t *testing.T) {
	var direct, proxy bool
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		direct = true
		_, _ = io.WriteString(w, "ok")
	}))
	defer origin.Close()
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy = true
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxyServer.Close()
	t.Setenv("HTTP_PROXY", proxyServer.URL)
	t.Setenv("http_proxy", proxyServer.URL)
	t.Setenv("HTTPS_PROXY", proxyServer.URL)
	t.Setenv("https_proxy", proxyServer.URL)
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	service := NewUploadFromURLService(nil, true, origin.URL, "secret")
	if transport := service.newHTTPTransport(&net.Dialer{}, false); transport.Proxy != nil {
		t.Fatal("URL import transport delegates target resolution to an environment proxy")
	}
	filename, body, err := service.downloadFile(context.Background(), origin.URL+"/file.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = body.Close()
	if filename != "file.txt" || !direct || proxy {
		t.Fatalf("filename=%q direct=%t proxy=%t", filename, direct, proxy)
	}

	cloud := NewUploadFromURLService(nil, true, origin.URL, "secret")
	valid := origin.URL + "/remote.php/dav/public-files/token/folder/report.pdf"
	if _, err := cloud.ValidateURLForSource(valid, OpenCloudURLSource); err != nil {
		t.Fatalf("valid nested path rejected: %v", err)
	}
	for _, raw := range []string{
		origin.URL + "/remote.php/dav/public-files/token/../secret.pdf",
		origin.URL + "/remote.php/dav/public-files/token/%2e%2e/secret.pdf",
		origin.URL + "/remote.php/dav/public-files/token/folder%2F..%2Fsecret.pdf",
		origin.URL + "/remote.php/dav/public-files/token/folder%5Csecret.pdf",
	} {
		if _, err := cloud.ValidateURLForSource(raw, OpenCloudURLSource); err == nil {
			t.Errorf("accepted unsafe path %q", raw)
		}
	}
}

func TestGenericURLRedirectFailureRedactsURLFromErrorsAndLogs(t *testing.T) {
	const token = "generic-download-token-secret"
	const target = "http://10.0.0.1/file.pdf?token=" + token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}))
	defer server.Close()
	service := NewUploadFromURLService(nil, true, "", "")
	rawURL := server.URL + "/file.pdf?token=" + token
	var logs strings.Builder
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	_, body, err := service.downloadFile(context.Background(), rawURL, "")
	if body != nil {
		_ = body.Close()
	}
	var httpErr *e.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("download error = %T %v, want sanitized HTTPError", err, err)
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), server.URL) || strings.Contains(err.Error(), "10.0.0.1") {
		t.Fatalf("download error reflected sensitive URL: %v", err)
	}

	service.SetDownloadFileForTesting(func(_ context.Context, rawURL string) (string, io.ReadCloser, error) {
		return service.downloadFile(context.Background(), rawURL, "")
	})
	_, err = service.UploadFromURL(ctxx.Context(nil), rawURL, "")
	if !errors.As(err, &httpErr) {
		t.Fatalf("upload flow error = %T %v, want sanitized HTTPError", err, err)
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), server.URL) || strings.Contains(err.Error(), "10.0.0.1") {
		t.Fatalf("upload flow error reflected sensitive URL: %v", err)
	}
	if strings.Contains(logs.String(), token) || strings.Contains(logs.String(), server.URL) || strings.Contains(logs.String(), "10.0.0.1") {
		t.Fatalf("generic URL failure logs exposed URL secret: %s", logs.String())
	}
}

func TestOpenCloudDownloadFailureLogDoesNotContainShareToken(t *testing.T) {
	const token = "share-token-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "failure")
	}))
	defer server.Close()
	service := NewUploadFromURLService(nil, true, server.URL, "password")
	var logs strings.Builder
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	_, body, err := service.downloadFile(context.Background(), server.URL+"/remote.php/dav/public-files/"+token+"/x.pdf", OpenCloudURLSource)
	if body != nil {
		_ = body.Close()
	}
	if err == nil {
		t.Fatal("expected failed request")
	}
	if strings.Contains(logs.String(), token) {
		t.Fatalf("log exposed token: %s", logs.String())
	}
}
