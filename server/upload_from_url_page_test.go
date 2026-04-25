package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/db/entmain"
	"github.com/marcobeierer/go-core/db/entmain/account"
	"github.com/marcobeierer/go-core/db/entmain/temporaryfile"
	"github.com/marcobeierer/go-core/db/entx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
)

func TestUploadFromURLCmdCreatesTemporaryFileAndRedirects(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		email := "from-url@example.com"
		createAccount(t, harness.mainDB, email, "from-url-secret")

		accountx := harness.mainDB.ReadWriteConn.Account.Query().
			Where(account.EmailEQ(entx.NewCIText(email))).
			OnlyX(context.Background())

		harness.actions.OpenFile.UploadFromURLCmd.SetDownloadFileForTesting(
			func(_ context.Context, _ string) (string, io.ReadCloser, error) {
				return "from-url.txt", io.NopCloser(strings.NewReader("hello from url")), nil
			},
		)

		var location string
		err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
			data := url.Values{}
			data.Set("url", "https://example.com/from-url.txt")

			req := httptest.NewRequest(http.MethodPost, "/-/open-file/upload-from-url-cmd", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()
			err := harness.actions.OpenFile.UploadFromURLCmd.Handler(
				httpx2.NewResponseWriter(rr),
				httpx2.NewRequest(req),
				mainCtx,
			)
			if err != nil {
				return fmt.Errorf("upload from url command: %w", err)
			}

			if rr.Code != http.StatusFound {
				return fmt.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
			}

			location = rr.Header().Get("Location")
			if !strings.HasPrefix(location, "/open-file/select-space/") {
				return fmt.Errorf("expected redirect to select-space, got %q", location)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("upload from url command: %v", err)
		}

		uploadToken := strings.TrimPrefix(location, "/open-file/select-space/")
		if uploadToken == "" {
			t.Fatal("expected upload token in redirect")
		}

		temporaryFiles := harness.mainDB.ReadWriteConn.TemporaryFile.Query().Where(
			temporaryfile.OwnerID(accountx.ID),
			temporaryfile.UploadToken(uploadToken),
		).AllX(context.Background())
		if len(temporaryFiles) != 1 {
			t.Fatalf("expected 1 temporary file, got %d", len(temporaryFiles))
		}

		if temporaryFiles[0].Filename != "from-url.txt" {
			t.Fatalf("expected filename %q, got %q", "from-url.txt", temporaryFiles[0].Filename)
		}
	})
}

func TestUploadFromURLCmdUsesHXRedirectForHTMXRequests(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		email := "from-url-htmx@example.com"
		createAccount(t, harness.mainDB, email, "from-url-secret")

		accountx := harness.mainDB.ReadWriteConn.Account.Query().
			Where(account.EmailEQ(entx.NewCIText(email))).
			OnlyX(context.Background())

		harness.actions.OpenFile.UploadFromURLCmd.SetDownloadFileForTesting(
			func(_ context.Context, _ string) (string, io.ReadCloser, error) {
				return "from-url.txt", io.NopCloser(strings.NewReader("hello from url")), nil
			},
		)

		err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
			data := url.Values{}
			data.Set("url", "https://example.com/from-url.txt")

			req := httptest.NewRequest(http.MethodPost, "/-/open-file/upload-from-url-cmd", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rr := httptest.NewRecorder()
			err := harness.actions.OpenFile.UploadFromURLCmd.Handler(
				httpx2.NewResponseWriter(rr),
				httpx2.NewRequest(req),
				mainCtx,
			)
			if err != nil {
				return fmt.Errorf("upload from url command: %w", err)
			}

			hxRedirect := rr.Header().Get("HX-Redirect")
			if !strings.HasPrefix(hxRedirect, "/open-file/select-space/") {
				return fmt.Errorf("expected HX-Redirect to select-space, got %q", hxRedirect)
			}

			if rr.Header().Get("Location") != "" {
				return fmt.Errorf("expected no Location header for htmx request")
			}

			return nil
		})
		if err != nil {
			t.Fatalf("upload from url command: %v", err)
		}
	})
}

func TestUploadFromURLCmdRejectsMissingURL(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "from-url-missing@example.com"
	createAccount(t, harness.mainDB, email, "from-url-secret")

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	var handlerErr error
	err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
		req := httptest.NewRequest(http.MethodPost, "/-/open-file/upload-from-url-cmd", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handlerErr = harness.actions.OpenFile.UploadFromURLCmd.Handler(
			httpx2.NewResponseWriter(rr),
			httpx2.NewRequest(req),
			mainCtx,
		)
		if handlerErr == nil {
			return fmt.Errorf("expected error")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	httpErr, ok := handlerErr.(*e.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", handlerErr)
	}
	if httpErr.StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, httpErr.StatusCode())
	}
}

func TestUploadFromURLCmdAllowsLocalhostURLInDevMode(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		testServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte("hello from localhost"))
		}))
		defer testServer.Close()

		email := "from-url-localhost@example.com"
		createAccount(t, harness.mainDB, email, "from-url-secret")

		accountx := harness.mainDB.ReadWriteConn.Account.Query().
			Where(account.EmailEQ(entx.NewCIText(email))).
			OnlyX(context.Background())

		var location string
		err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
			data := url.Values{}
			data.Set("url", testServer.URL+"/private.txt")

			req := httptest.NewRequest(http.MethodPost, "/-/open-file/upload-from-url-cmd", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()
			err := harness.actions.OpenFile.UploadFromURLCmd.Handler(
				httpx2.NewResponseWriter(rr),
				httpx2.NewRequest(req),
				mainCtx,
			)
			if err != nil {
				return fmt.Errorf("upload from url command: %w", err)
			}

			if rr.Code != http.StatusFound {
				return fmt.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
			}

			location = rr.Header().Get("Location")
			if !strings.HasPrefix(location, "/open-file/select-space/") {
				return fmt.Errorf("expected redirect to select-space, got %q", location)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("upload from url command: %v", err)
		}

		uploadToken := strings.TrimPrefix(location, "/open-file/select-space/")
		if uploadToken == "" {
			t.Fatal("expected upload token in redirect")
		}

		temporaryFiles := harness.mainDB.ReadWriteConn.TemporaryFile.Query().Where(
			temporaryfile.OwnerID(accountx.ID),
			temporaryfile.UploadToken(uploadToken),
		).AllX(context.Background())
		if len(temporaryFiles) != 1 {
			t.Fatalf("expected 1 temporary file, got %d", len(temporaryFiles))
		}
	})
}
