package server

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUnzipArchiveCmdExtractsFilesAndDeletesArchive(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)

	accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, accountx, tenantx, tenantDB)
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
	}()

	spaceName := "Archive Space"
	createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

	spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	rootDir := spaceCtx.SpaceRootDir()

	zipData := createZipArchive(t)
	zipFile, err := harness.infra.FileSystem().AddFile(
		spaceCtx,
		bytes.NewReader(zipData),
		"archive.zip",
		false,
		rootDir.ID,
	)
	if err != nil {
		t.Fatalf("add zip file: %v", err)
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
		t.Fatalf("unzip archive command: %v", err)
	}

	if header := rr.Header().Get("HX-Trigger"); header != event.ZIPArchiveUnzipped.String() {
		t.Fatalf("expected HX-Trigger %q, got %q", event.ZIPArchiveUnzipped.String(), header)
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
		t.Fatal("expected zip archive to be marked deleted")
	}

	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}
	committed = true
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
