package server

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUnzipArchiveCmdExtractsFilesAndDeletesArchive(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		spaceName := "Archive Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()

		zipData := createZipArchive(t)
		prepared, zipFile, err := harness.infra.FileSystem().PrepareFileUpload(
			spaceCtx,
			"archive.zip",
			rootDir.ID,
			false,
		)
		if err != nil {
			return fmt.Errorf("prepare zip file: %w", err)
		}

		fileInfo, fileSize, err := harness.infra.FileSystem().UploadPreparedFile(
			spaceCtx,
			bytes.NewReader(zipData),
			prepared,
		)
		if err != nil {
			return fmt.Errorf("upload zip file: %w", err)
		}

		err = harness.infra.FileSystem().FinalizePreparedUpload(spaceCtx, prepared, fileInfo, fileSize)
		if err != nil {
			return fmt.Errorf("finalize zip file: %w", err)
		}

		form := url.Values{}
		form.Set("FileID", zipFile.PublicID.String())
		form.Set("DeleteOnSuccess", "true")

		req := httptest.NewRequest(http.MethodPost, "/-/browse/unzip-archive-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		err = harness.actions.Browse.UnzipArchiveCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
		)
		if err != nil {
			return fmt.Errorf("unzip archive command: %w", err)
		}

		if header := rr.Header().Get("HX-Trigger"); header != event.ZIPArchiveUnzipped.String() {
			return fmt.Errorf("expected HX-Trigger %q, got %q", event.ZIPArchiveUnzipped.String(), header)
		}

		docsDir := spaceCtx.TTx.File.Query().Where(
			file.Name("docs"),
			file.ParentID(rootDir.ID),
			file.IsDirectory(true),
		).OnlyX(spaceCtx)

		_ = spaceCtx.TTx.File.Query().Where(
			file.Name("readme.txt"),
			file.ParentID(docsDir.ID),
			file.IsDirectory(false),
		).OnlyX(spaceCtx)

		_ = spaceCtx.TTx.File.Query().Where(
			file.Name("notes.txt"),
			file.ParentID(rootDir.ID),
			file.IsDirectory(false),
		).OnlyX(spaceCtx)

		zipRecord := spaceCtx.TTx.File.Query().Where(
			file.PublicIDEQ(entx.NewCIText(zipFile.PublicID.String())),
		).OnlyX(schema.SkipSoftDelete(spaceCtx))
		if zipRecord.DeletedAt.IsZero() {
			return fmt.Errorf("expected zip archive to be marked deleted")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("unzip archive command: %v", err)
	}
}

func TestUnzipArchiveCmdRejectsNonZipFile(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var handlerErr error
	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		spaceName := "Non Zip Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()

		prepared, filex, err := harness.infra.FileSystem().PrepareFileUpload(
			spaceCtx,
			"notes.txt",
			rootDir.ID,
			false,
		)
		if err != nil {
			return fmt.Errorf("prepare file: %w", err)
		}

		fileInfo, fileSize, err := harness.infra.FileSystem().UploadPreparedFile(
			spaceCtx,
			bytes.NewReader([]byte("not a zip")),
			prepared,
		)
		if err != nil {
			return fmt.Errorf("upload file: %w", err)
		}

		err = harness.infra.FileSystem().FinalizePreparedUpload(spaceCtx, prepared, fileInfo, fileSize)
		if err != nil {
			return fmt.Errorf("finalize file: %w", err)
		}

		form := url.Values{}
		form.Set("FileID", filex.PublicID.String())

		req := httptest.NewRequest(http.MethodPost, "/-/browse/unzip-archive-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()
		handlerErr = harness.actions.Browse.UnzipArchiveCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
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

func createZipArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	readme, err := zipWriter.Create("docs/readme.txt")
	if err != nil {
		t.Fatalf("create readme entry: %v", err)
	}
	if _, err := readme.Write([]byte("hello")); err != nil {
		t.Fatalf("write readme entry: %v", err)
	}

	notes, err := zipWriter.Create("notes.txt")
	if err != nil {
		t.Fatalf("create notes entry: %v", err)
	}
	if _, err := notes.Write([]byte("notes")); err != nil {
		t.Fatalf("write notes entry: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buf.Bytes()
}
