package server

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUploadFilesCmdCreatesTemporaryFilesAndRedirects(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	email := "shared@example.com"
	createAccount(t, harness.mainDB, email, "shared-secret")

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	var location string
	err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
		req := newSharedUploadRequest(t, map[string]string{
			"first.txt":  "hello",
			"second.txt": "world",
		})

		rr := httptest.NewRecorder()
		err := harness.actions.OpenFile.UploadFilesCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			mainCtx,
		)
		if err != nil {
			return fmt.Errorf("upload files command: %w", err)
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
		t.Fatalf("upload files command: %v", err)
	}

	uploadToken := strings.TrimPrefix(location, "/open-file/select-space/")
	if uploadToken == "" {
		t.Fatal("expected upload token in redirect")
	}

	temporaryFiles := harness.mainDB.ReadWriteConn.TemporaryFile.Query().Where(
		temporaryfile.OwnerID(accountx.ID),
		temporaryfile.UploadToken(uploadToken),
	).AllX(context.Background())
	if len(temporaryFiles) != 2 {
		t.Fatalf("expected 2 temporary files, got %d", len(temporaryFiles))
	}
}

func TestUploadFilesCmdRejectsNonMultipartRequests(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	email := "shared-invalid@example.com"
	createAccount(t, harness.mainDB, email, "shared-secret")

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	var handlerErr error
	err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
		req := httptest.NewRequest(http.MethodPost, "/-/open-file/upload-files-cmd", strings.NewReader("plain"))
		req.Header.Set("Content-Type", "text/plain")

		rr := httptest.NewRecorder()
		handlerErr = harness.actions.OpenFile.UploadFilesCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
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
	if httpErr.StatusCode() != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, httpErr.StatusCode())
	}
}

func newSharedUploadRequest(t *testing.T, files map[string]string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for filename, content := range files {
		fileWriter, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := fileWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/-/open-file/upload-files-cmd", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
