package server

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestConcurrentUploadFileCmd(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	accountx, tenantx := signUpAccount(t, harness, "uploader@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, accountx, tenantx, tenantDB)
	spaceName := "Concurrent Uploads"
	createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

	spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	rootDir := spaceCtx.SpaceRootDir()
	parentDirID := rootDir.PublicID.String()
	rootDirID := rootDir.ID
	spaceID := spacex.ID

	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}

	const uploadCount = 6
	var wg sync.WaitGroup
	errCh := make(chan error, uploadCount)

	for i := 0; i < uploadCount; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
			if err != nil {
				errCh <- err
				return
			}

			committed := false
			defer func() {
				if committed {
					return
				}
				_ = tenantTx.Rollback()
				_ = mainTx.Rollback()
			}()

			spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

			filename := fmt.Sprintf("concurrent-%d.txt", i)
			req, err := newUploadRequest(parentDirID, filename, []byte("hello"))
			if err != nil {
				errCh <- err
				return
			}

			rr := httptest.NewRecorder()
			err = harness.actions.Browse.UploadFileCmd.Handler(
				httpx.NewResponseWriter(rr),
				httpx.NewRequest(req),
				spaceCtx,
			)
			if err != nil {
				errCh <- err
				return
			}

			if err := mainTx.Commit(); err != nil {
				errCh <- err
				return
			}
			if err := tenantTx.Commit(); err != nil {
				errCh <- err
				return
			}
			committed = true
			errCh <- nil
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("upload failed: %v", err)
		}
	}

	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
	if err != nil {
		t.Fatalf("new tenant context: %v", err)
	}
	defer func() {
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
	}()

	spacex = tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
	spaceCtx = ctxx.NewSpaceContext(tenantCtx, spacex)
	fileCount := spaceCtx.TTx.File.Query().Where(
		file.ParentID(rootDirID),
		file.IsDirectory(false),
	).CountX(spaceCtx)
	if fileCount != uploadCount {
		t.Fatalf("expected %d files, got %d", uploadCount, fileCount)
	}
}

func newTenantContextForUpload(
	harness *actionTestHarness,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	tenantDB *sqlx.TenantDB,
) (*entmain.Tx, *enttenant.Tx, *ctxx.TenantContext, error) {
	mainTx, err := harness.mainDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		return nil, nil, nil, err
	}

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

	tenantTx, err := tenantDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		_ = mainTx.Rollback()
		return nil, nil, nil, err
	}

	tenantCtx := ctxx.NewTenantContext(mainCtx, tenantTx, tenantx)
	return mainTx, tenantTx, tenantCtx, nil
}

func newUploadRequest(parentDirID, filename string, content []byte) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("ParentDirID", parentDirID); err != nil {
		return nil, err
	}
	if err := writer.WriteField("AddToInbox", "false"); err != nil {
		return nil, err
	}

	fileWriter, err := writer.CreateFormFile("File", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fileWriter.Write(content); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/-/browse/upload-file-cmd", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}
