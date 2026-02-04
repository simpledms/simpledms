package server

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUploadFilesCmdCreatesTemporaryFilesAndRedirects(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	email := "shared@example.com"
	password := "shared-secret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	mainTx, err := harness.mainDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = mainTx.Rollback()
	}()

	visitorCtx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtx := ctxx.NewMainContext(visitorCtx, accountx, harness.i18n, harness.tenantDBs)

	req := newSharedUploadRequest(t, map[string]string{
		"first.txt":  "hello",
		"second.txt": "world",
	})

	rr := httptest.NewRecorder()
	err = harness.actions.OpenFile.UploadFilesCmd.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		mainCtx,
	)
	if err != nil {
		t.Fatalf("upload files command: %v", err)
	}

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if !strings.HasPrefix(location, "/open-file/select-space/") {
		t.Fatalf("expected redirect to select-space, got %q", location)
	}
	uploadToken := strings.TrimPrefix(location, "/open-file/select-space/")
	if uploadToken == "" {
		t.Fatal("expected upload token in redirect")
	}

	if err := mainTx.Commit(); err != nil {
		t.Fatalf("commit main tx: %v", err)
	}
	committed = true

	temporaryFiles := harness.mainDB.ReadWriteConn.TemporaryFile.Query().Where(
		temporaryfile.OwnerID(accountx.ID),
		temporaryfile.UploadToken(uploadToken),
	).AllX(context.Background())
	if len(temporaryFiles) != 2 {
		t.Fatalf("expected 2 temporary files, got %d", len(temporaryFiles))
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
