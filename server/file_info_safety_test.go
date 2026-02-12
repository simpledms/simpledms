package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	browseaction "github.com/simpledms/simpledms/action/browse"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestFileSystemMakeDirAllIfNotExists_IdempotentAndFileCollision(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "fileinfo-mkdir@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "FileInfo Safety Mkdir")

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("FileInfo Safety Mkdir")).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()

		createdDir, err := harness.infra.FileSystem().MakeDirAllIfNotExists(spaceCtx, rootDir, "clients/acme")
		if err != nil {
			return fmt.Errorf("create nested dirs: %w", err)
		}

		createdDirAgain, err := harness.infra.FileSystem().MakeDirAllIfNotExists(spaceCtx, rootDir, "clients/acme")
		if err != nil {
			return fmt.Errorf("create nested dirs second time: %w", err)
		}
		if createdDir.ID != createdDirAgain.ID {
			return fmt.Errorf("expected idempotent result, got %d and %d", createdDir.ID, createdDirAgain.ID)
		}

		clientDirs := spaceCtx.TTx.File.Query().Where(
			file.SpaceID(spacex.ID),
			file.ParentID(rootDir.ID),
			file.Name("clients"),
			file.IsDirectory(true),
		).AllX(spaceCtx)
		if len(clientDirs) != 1 {
			return fmt.Errorf("expected exactly one clients directory, got %d", len(clientDirs))
		}

		acmeDirs := spaceCtx.TTx.File.Query().Where(
			file.SpaceID(spacex.ID),
			file.ParentID(clientDirs[0].ID),
			file.Name("acme"),
			file.IsDirectory(true),
		).AllX(spaceCtx)
		if len(acmeDirs) != 1 {
			return fmt.Errorf("expected exactly one acme directory, got %d", len(acmeDirs))
		}

		_ = createRegularFileForTest(spaceCtx, rootDir.ID, "report.txt")

		_, err = harness.infra.FileSystem().MakeDirAllIfNotExists(spaceCtx, rootDir, "report.txt/nested")
		if err == nil {
			return fmt.Errorf("expected collision error when path segment is a file")
		}

		var httpErr *e.HTTPError
		if !errors.As(err, &httpErr) {
			return fmt.Errorf("expected http error, got %T", err)
		}
		if httpErr.StatusCode() != http.StatusBadRequest {
			return fmt.Errorf("expected status %d, got %d", http.StatusBadRequest, httpErr.StatusCode())
		}

		return nil
	})
	if err != nil {
		t.Fatalf("mkdir all if not exists behavior: %v", err)
	}
}

func TestFileSystemMoveRejectsMovingDirectoryIntoDescendant(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "fileinfo-move@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "FileInfo Safety Move")

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("FileInfo Safety Move")).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()

		dirA, err := harness.infra.FileSystem().MakeDir(spaceCtx, rootDir.PublicID.String(), "alpha")
		if err != nil {
			return fmt.Errorf("create alpha dir: %w", err)
		}
		dirB, err := harness.infra.FileSystem().MakeDir(spaceCtx, dirA.Data.PublicID.String(), "beta")
		if err != nil {
			return fmt.Errorf("create beta dir: %w", err)
		}

		_, err = harness.infra.FileSystem().Move(spaceCtx, dirB, dirA, "", "")
		if err == nil {
			return fmt.Errorf("expected move rejection for descendant destination")
		}

		var httpErr *e.HTTPError
		if !errors.As(err, &httpErr) {
			return fmt.Errorf("expected http error, got %T", err)
		}
		if httpErr.StatusCode() != http.StatusBadRequest {
			return fmt.Errorf("expected status %d, got %d", http.StatusBadRequest, httpErr.StatusCode())
		}

		alphaAfter := spaceCtx.TTx.File.GetX(spaceCtx, dirA.Data.ID)
		if alphaAfter.ParentID != rootDir.ID {
			return fmt.Errorf("expected alpha parent id %d, got %d", rootDir.ID, alphaAfter.ParentID)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("move safety behavior: %v", err)
	}
}

func TestBrowseListDirPartialRecursiveSearchIsScopedToCurrentDirectory(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "fileinfo-browse-recursive@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "FileInfo Safety Browse Recursive")

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("FileInfo Safety Browse Recursive")).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()

		dirA := createDirectoryForTest(spaceCtx, rootDir.ID, "a")
		dirANested := createDirectoryForTest(spaceCtx, dirA.ID, "nested")
		dirB := createDirectoryForTest(spaceCtx, rootDir.ID, "b")

		_ = createDirectoryForTest(spaceCtx, dirANested.ID, "invoice-alpha")
		_ = createDirectoryForTest(spaceCtx, dirB.ID, "invoice-beta")

		form := url.Values{}
		form.Set("CurrentDirID", dirA.PublicID.String())
		form.Set("SelectedFileID", "")

		req := httptest.NewRequest(
			http.MethodPost,
			"/-/browse/list-dir-partial?hx-target=%23fileList",
			strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.Header.Set(
			"HX-Current-URL",
			route.Browse(tenantx.PublicID.String(), spacex.PublicID.String(), dirA.PublicID.String())+"?q=invoice",
		)

		rr := httptest.NewRecorder()
		err := harness.actions.Browse.ListDirPartial.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
		)
		if err != nil {
			return fmt.Errorf("render recursive browse list: %w", err)
		}
		if rr.Code != http.StatusOK {
			return fmt.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		body := rr.Body.String()
		if !strings.Contains(body, "invoice-alpha") {
			return fmt.Errorf("expected scoped recursive result to include invoice-alpha")
		}
		if strings.Contains(body, "invoice-beta") {
			return fmt.Errorf("expected scoped recursive result to exclude invoice-beta")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("recursive browse scope behavior: %v", err)
	}
}

func TestBrowseListDirPartialWidgetBuildsFolderBreadcrumbs(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "fileinfo-breadcrumbs@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var spaceID int64
	var dirBetaPublicID string

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "FileInfo Safety Breadcrumbs")

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("FileInfo Safety Breadcrumbs")).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()
		spaceID = spacex.ID

		dirAlpha := createDirectoryForTest(spaceCtx, rootDir.ID, "alpha")
		dirBeta := createDirectoryForTest(spaceCtx, dirAlpha.ID, "beta")
		dirBetaPublicID = dirBeta.PublicID.String()

		return nil
	})
	if err != nil {
		t.Fatalf("prepare breadcrumb data: %v", err)
	}

	mainTx, tenantTx, tenantReadOnlyCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
	if err != nil {
		t.Fatalf("new read-only tenant context: %v", err)
	}
	defer func() {
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
	}()

	readOnlySpace := tenantReadOnlyCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantReadOnlyCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantReadOnlyCtx, readOnlySpace)

	layout := harness.actions.Browse.ListDirPartial.Widget(
		spaceCtx,
		&browseaction.ListDirPartialState{},
		dirBetaPublicID,
		"",
	)

	listColumn, ok := layout.List.(*wx.Column)
	if !ok {
		t.Fatalf("expected list to be *wx.Column, got %T", layout.List)
	}

	listChildren, ok := listColumn.Children.([]wx.IWidget)
	if !ok {
		t.Fatalf("expected list children to be []wx.IWidget, got %T", listColumn.Children)
	}

	var statusBar *wx.StatusBar
	for _, child := range listChildren {
		statusBarCandidate, isStatusBar := child.(*wx.StatusBar)
		if isStatusBar {
			statusBar = statusBarCandidate
			break
		}
	}
	if statusBar == nil {
		t.Fatal("expected breadcrumbs status bar")
	}

	breadcrumbWidgets, ok := statusBar.Child.([]wx.IWidget)
	if !ok {
		t.Fatalf("expected breadcrumbs to be []wx.IWidget, got %T", statusBar.Child)
	}
	if len(breadcrumbWidgets) < 5 {
		t.Fatalf("expected at least 5 breadcrumb widgets, got %d", len(breadcrumbWidgets))
	}

	foundAlphaLink := false
	for _, breadcrumbWidget := range breadcrumbWidgets {
		breadcrumbLink, isLink := breadcrumbWidget.(*wx.Link)
		if !isLink {
			continue
		}
		breadcrumbText, isText := breadcrumbLink.Child.(*wx.Text)
		if isText && breadcrumbText.String(spaceCtx) == "alpha" {
			foundAlphaLink = true
			break
		}
	}
	if !foundAlphaLink {
		t.Fatal("expected breadcrumb link for alpha directory")
	}

	lastBreadcrumbText, ok := breadcrumbWidgets[len(breadcrumbWidgets)-1].(*wx.Text)
	if !ok {
		t.Fatalf("expected last breadcrumb element to be text, got %T", breadcrumbWidgets[len(breadcrumbWidgets)-1])
	}
	if got := lastBreadcrumbText.String(spaceCtx); got != "beta" {
		t.Fatalf("expected last breadcrumb text beta, got %q", got)
	}
}

func createDirectoryForTest(spaceCtx *ctxx.SpaceContext, parentID int64, name string) *enttenant.File {
	now := time.Now()
	return spaceCtx.TTx.File.Create().
		SetName(name).
		SetIsDirectory(true).
		SetIndexedAt(now).
		SetModifiedAt(now).
		SetParentID(parentID).
		SetSpaceID(spaceCtx.Space.ID).
		SaveX(spaceCtx)
}

func createRegularFileForTest(spaceCtx *ctxx.SpaceContext, parentID int64, name string) *model.File {
	now := time.Now()
	filex := spaceCtx.TTx.File.Create().
		SetName(name).
		SetIsDirectory(false).
		SetIndexedAt(now).
		SetModifiedAt(now).
		SetParentID(parentID).
		SetSpaceID(spaceCtx.Space.ID).
		SetIsInInbox(false).
		SaveX(spaceCtx)

	return model.NewFile(filex)
}
