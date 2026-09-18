package temporaryfile

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/simpledms/simpledms/util/e"
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
	defer func() {
		if err := body.Close(); err != nil {
			t.Errorf("close download body: %v", err)
		}
	}()
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

func TestOpenCloudDownloadErrorsNeverReflectUpstreamDetails(t *testing.T) {
	const downloadPath = "/remote.php/dav/public-files/random_token/report.pdf"

	tests := []struct {
		name       string
		status     int
		want       string
		wantStatus int
	}{
		{
			name:   "password",
			status: http.StatusUnauthorized,
			want: "OpenCloud rejected the integration password. " +
				"Ask your administrator to check the integration settings.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "permission",
			status: http.StatusForbidden,
			want: "OpenCloud does not allow this file to be downloaded. " +
				"Ask the file owner or your administrator for access.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "missing",
			status: http.StatusNotFound,
			want: "The OpenCloud link has expired or is no longer available. " +
				"Start a new export from OpenCloud.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "gone",
			status: http.StatusGone,
			want: "The OpenCloud link has expired or is no longer available. " +
				"Start a new export from OpenCloud.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "throttled",
			status: http.StatusTooManyRequests,
			want: "OpenCloud is receiving too many requests. " +
				"Wait a moment and try again.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "redirect",
			status: http.StatusFound,
			want: "SimpleDMS could not download the file safely. " +
				"Ask your administrator to check the integration settings.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "transient",
			status:     http.StatusBadGateway,
			want:       "OpenCloud could not provide the file right now. Try again later.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "other",
			status: http.StatusBadRequest,
			want: "Could not download the file from OpenCloud. " +
				"Try again, or ask your administrator for help.",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reached atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.Path != downloadPath {
					t.Errorf("unexpected OpenCloud path %q", req.URL.Path)
				}
				reached.Store(true)
				rw.Header().Set("X-Request-ID", "request-secret")
				rw.Header().Set("X-Upstream-Secret", "header-secret")
				rw.WriteHeader(tt.status)
				_, _ = io.WriteString(rw, "body-secret https://cloud.example/private?password=secret")
			}))
			t.Cleanup(server.Close)

			service := NewUploadFromURLService(nil, true, server.URL, "password-secret")
			_, body, err := service.downloadFile(
				context.Background(),
				server.URL+downloadPath,
				OpenCloudURLSource,
			)
			if !reached.Load() {
				t.Fatal("OpenCloud handler was not reached")
			}
			if body != nil {
				t.Fatal("failed download returned a body")
			}
			if err == nil {
				t.Fatal("expected download failure")
			}
			var httpErr *e.HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("error type = %T, want *e.HTTPError", err)
			}
			if httpErr.StatusCode() != tt.wantStatus || httpErr.Message() != tt.want {
				t.Fatalf(
					"HTTPError = (%d, %q), want (%d, %q)",
					httpErr.StatusCode(), httpErr.Message(), tt.wantStatus, tt.want,
				)
			}
			secrets := []string{
				"body-secret",
				"header-secret",
				"request-secret",
				"password-secret",
				"cloud.example",
			}
			for _, secret := range secrets {
				if strings.Contains(httpErr.Message(), secret) {
					t.Errorf("HTTPError reflected upstream detail %q", secret)
				}
			}
		})
	}
}

func TestOpenCloudDownloadWithEmptySourceKeepsGenericError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		http.Error(rw, "body-secret", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	service := NewUploadFromURLService(nil, true, server.URL, "password-secret")
	_, body, err := service.downloadFile(context.Background(), server.URL+"/report.pdf", "")
	if body != nil {
		t.Fatal("failed download returned a body")
	}
	if err == nil || err.Error() != "could not download file from url" {
		t.Fatalf("error = %v, want the existing generic message", err)
	}
}

func TestOpenCloudDownloadRejectsUnsupportedContentDispositionFilename(t *testing.T) {
	const downloadPath = "/remote.php/dav/public-files/random_token/report.pdf"
	const wantMessage = "The OpenCloud file has an unsupported filename. " +
		"Rename the file and start a new export."
	var reached atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != downloadPath {
			t.Errorf("unexpected OpenCloud path %q", req.URL.Path)
		}
		reached.Store(true)
		rw.Header().Set("Content-Disposition", `attachment; filename="bad/name.pdf"`)
		_, _ = io.WriteString(rw, "document")
	}))
	t.Cleanup(server.Close)

	service := NewUploadFromURLService(nil, true, server.URL, "password-secret")
	_, body, err := service.downloadFile(
		context.Background(),
		server.URL+downloadPath,
		OpenCloudURLSource,
	)
	if !reached.Load() {
		t.Fatal("OpenCloud handler was not reached")
	}
	if body != nil {
		_ = body.Close()
		t.Fatal("unsupported filename returned a body")
	}
	var httpErr *e.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode() != http.StatusBadRequest ||
		httpErr.Message() != wantMessage {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenCloudRequestErrorClassifiesWrappedFailuresSafely(t *testing.T) {
	preserved := e.NewHTTPErrorf(http.StatusForbidden, "preserved message")
	tests := []struct {
		name       string
		err        error
		want       string
		wantStatus int
	}{
		{
			name: "unknown authority",
			err:  fmt.Errorf("wrapped: %w", x509.UnknownAuthorityError{}),
			want: "SimpleDMS could not establish a secure connection to OpenCloud. " +
				"Ask your administrator for help.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid certificate",
			err:  fmt.Errorf("wrapped: %w", x509.CertificateInvalidError{}),
			want: "SimpleDMS could not establish a secure connection to OpenCloud. " +
				"Ask your administrator for help.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "hostname",
			err:  fmt.Errorf("wrapped: %w", x509.HostnameError{}),
			want: "SimpleDMS could not establish a secure connection to OpenCloud. " +
				"Ask your administrator for help.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "context timeout",
			err:        fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
			want:       "OpenCloud took too long to respond. Try again.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "timeout",
			err:        fmt.Errorf("wrapped: %w", &net.DNSError{Err: "timeout-secret", IsTimeout: true}),
			want:       "OpenCloud took too long to respond. Try again.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "network",
			err:  errors.New("network-secret"),
			want: "SimpleDMS could not connect to OpenCloud. " +
				"Try again later, or ask your administrator for help.",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "preserved http error",
			err:        fmt.Errorf("wrapped: %w", preserved),
			want:       "preserved message",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := openCloudRequestError(tt.err)
			if got.StatusCode() != tt.wantStatus || got.Message() != tt.want {
				t.Fatalf(
					"HTTPError = (%d, %q), want (%d, %q)",
					got.StatusCode(), got.Message(), tt.wantStatus, tt.want,
				)
			}
			if strings.Contains(got.Message(), "secret") && tt.name != "preserved http error" {
				t.Fatalf("HTTPError reflected network detail: %q", got.Message())
			}
		})
	}
}
