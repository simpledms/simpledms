package server

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	toxiproxy "github.com/Shopify/toxiproxy/client"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant/file"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestConcurrentUploadFileCmdWithSlowS3(t *testing.T) {
	proxy, s3Endpoint := newS3Toxiproxy(t)
	t.Setenv("SIMPLEDMS_S3_ENDPOINT", s3Endpoint)

	harness := newActionTestHarnessWithS3(t)

	_, err := proxy.AddToxic("upload-latency", "latency", "upstream", 1, toxiproxy.Attributes{
		"latency": int64(150),
		"jitter":  int64(50),
	})
	if err != nil {
		t.Fatalf("add latency toxic: %v", err)
	}

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

	const uploadCount = 4
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

			filename := fmt.Sprintf("slow-concurrent-%d.txt", i)
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

	mainTx, tenantTx, tenantCtx, err = newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
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

func TestUploadFileCmdFailsWhenS3Unavailable(t *testing.T) {
	proxy, s3Endpoint := newS3Toxiproxy(t)
	t.Setenv("SIMPLEDMS_S3_ENDPOINT", s3Endpoint)

	harness := newActionTestHarnessWithS3(t)
	if err := proxy.Disable(); err != nil {
		t.Fatalf("disable toxiproxy: %v", err)
	}
	t.Cleanup(func() {
		if err := proxy.Enable(); err != nil {
			t.Logf("enable toxiproxy: %v", err)
		}
	})

	accountx, tenantx := signUpAccount(t, harness, "uploader-fail@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, accountx, tenantx, tenantDB)
	spaceName := "Failure Uploads"
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

	req, err := newUploadRequest(parentDirID, "broken.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("new upload request: %v", err)
	}

	rr := httptest.NewRecorder()
	handlerErr := harness.actions.Browse.UploadFileCmd.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		spaceCtx,
	)
	if handlerErr == nil {
		t.Fatal("expected upload failure")
	}

	httpErr, ok := handlerErr.(*e.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", handlerErr)
	}
	if httpErr.StatusCode() != 500 {
		t.Fatalf("expected status %d, got %d", 500, httpErr.StatusCode())
	}

	_ = tenantTx.Rollback()
	_ = mainTx.Rollback()

	mainTx, tenantTx, tenantCtx, err = newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
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
	if fileCount != 0 {
		/*
			ctx := enttenantschema.WithUnfinishedUploads(spaceCtx)
			filex := spaceCtx.TTx.File.Query().Where(
				file.ParentID(rootDirID),
				file.IsDirectory(false),
			).OnlyX(ctx)
			t.Log(filex)
			versionx := filex.QueryVersions().OnlyX(ctx)
			t.Log(versionx)
		*/

		t.Fatalf("expected 0 files, got %d", fileCount)
	}
}

func TestUploadFilesCmdFailsWhenS3Unavailable(t *testing.T) {
	proxy, s3Endpoint := newS3Toxiproxy(t)
	t.Setenv("SIMPLEDMS_S3_ENDPOINT", s3Endpoint)

	harness := newActionTestHarnessWithS3(t)
	if err := proxy.Disable(); err != nil {
		t.Fatalf("disable toxiproxy: %v", err)
	}
	t.Cleanup(func() {
		if err := proxy.Enable(); err != nil {
			t.Logf("enable toxiproxy: %v", err)
		}
	})

	email := "shared-toxiproxy@example.com"
	createAccount(t, harness.mainDB, email, "shared-secret")

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	var handlerErr error
	err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
		req := newSharedUploadRequest(t, map[string]string{
			"first.txt": "hello",
		})

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
	if httpErr.StatusCode() != 500 {
		t.Fatalf("expected status %d, got %d", 500, httpErr.StatusCode())
	}

	fileCount := harness.mainDB.ReadOnlyConn.TemporaryFile.Query().Where(
		temporaryfile.OwnerID(accountx.ID),
	).CountX(context.Background())
	if fileCount != 0 {
		t.Fatalf("expected 0 temporary files, got %d", fileCount)
	}
}
