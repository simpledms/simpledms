package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/e"
)

func TestSpaceFileRepositoryFactory_RequiresSpaceContext(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "space-repo-factory@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		repos, err := harness.infra.SpaceFileRepoFactory().ForSpace(tenantCtx)
		if err == nil {
			t.Fatalf("expected error for tenant context")
		}
		if repos != nil {
			t.Fatalf("expected nil repos for tenant context")
		}

		var httpErr *e.HTTPError
		if !errors.As(err, &httpErr) {
			t.Fatalf("expected HTTPError, got %T", err)
		}
		if httpErr.StatusCode() != 400 {
			t.Fatalf("expected status 400, got %d", httpErr.StatusCode())
		}

		return nil
	})
	if err != nil {
		t.Fatalf("factory guard test failed: %v", err)
	}
}

func TestEntSpaceFileReadRepository_PolicyRejectsMismatchedSpaceContext(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "space-repo-policy@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Repository Policy Space A")
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Repository Policy Space B")

		spaceA := tenantCtx.TTx.Space.Query().Where(space.Name("Repository Policy Space A")).OnlyX(tenantCtx)
		spaceB := tenantCtx.TTx.Space.Query().Where(space.Name("Repository Policy Space B")).OnlyX(tenantCtx)

		spaceCtxA := ctxx.NewSpaceContext(tenantCtx, spaceA)
		filex := createRegularFileForTest(spaceCtxA, spaceCtxA.SpaceRootDir().ID, "policy-target.txt")

		spaceCtxB := ctxx.NewSpaceContext(tenantCtx, spaceB)

		readRepo := filemodel.NewEntSpaceFileReadRepository(spaceA.ID)
		_, readErr := readRepo.FileByPublicID(spaceCtxB, filex.Data.PublicID.String())
		if readErr == nil {
			t.Fatalf("expected mismatched space context query to fail")
		}
		if !enttenant.IsNotFound(readErr) {
			t.Fatalf("expected not found from policy + scope constraints, got: %v", readErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("policy test failed: %v", err)
	}
}

func TestSpaceFileReadRepository_IsScopedBySpace(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "space-repo-scope@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Repository Scope A")
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Repository Scope B")

		spaceA := tenantCtx.TTx.Space.Query().Where(space.Name("Repository Scope A")).OnlyX(tenantCtx)
		spaceB := tenantCtx.TTx.Space.Query().Where(space.Name("Repository Scope B")).OnlyX(tenantCtx)

		spaceCtxA := ctxx.NewSpaceContext(tenantCtx, spaceA)
		fileInA := createRegularFileForTest(spaceCtxA, spaceCtxA.SpaceRootDir().ID, "in-a.txt")

		spaceCtxB := ctxx.NewSpaceContext(tenantCtx, spaceB)
		fileInB := createRegularFileForTest(spaceCtxB, spaceCtxB.SpaceRootDir().ID, "in-b.txt")

		spaceCtxA = ctxx.NewSpaceContext(tenantCtx, spaceA)

		reposA := harness.infra.SpaceFileRepoFactory().ForSpaceX(spaceCtxA)

		gotA, readErr := reposA.Read.FileByPublicID(spaceCtxA, fileInA.Data.PublicID.String())
		if readErr != nil {
			t.Fatalf("read file from own space: %v", readErr)
		}
		if gotA.PublicID != fileInA.Data.PublicID.String() {
			t.Fatalf("unexpected file from own space: %s", gotA.PublicID)
		}

		_, crossErr := reposA.Read.FileByPublicID(spaceCtxA, fileInB.Data.PublicID.String())
		if crossErr == nil {
			t.Fatalf("expected cross-space lookup to fail")
		}
		if !enttenant.IsNotFound(crossErr) {
			t.Fatalf("expected not found for cross-space lookup, got %v", crossErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("space-scope test failed: %v", err)
	}
}

func TestSpaceFileWriteRepository_RestoreDeletedFileIsScopedAndHandlesMissingParent(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "space-repo-restore@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Repository Restore A")
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Repository Restore B")

		spaceA := tenantCtx.TTx.Space.Query().Where(space.Name("Repository Restore A")).OnlyX(tenantCtx)
		spaceB := tenantCtx.TTx.Space.Query().Where(space.Name("Repository Restore B")).OnlyX(tenantCtx)

		spaceCtxA := ctxx.NewSpaceContext(tenantCtx, spaceA)
		rootA := spaceCtxA.SpaceRootDir()
		parentDirA := createDirectoryForTest(spaceCtxA, rootA.ID, "to-delete-parent")
		fileInA := createRegularFileForTest(spaceCtxA, parentDirA.ID, "to-restore.txt")

		spaceCtxB := ctxx.NewSpaceContext(tenantCtx, spaceB)
		createDirectoryForTest(spaceCtxB, spaceCtxB.SpaceRootDir().ID, "other-parent")
		fileInB := createRegularFileForTest(spaceCtxB, spaceCtxB.SpaceRootDir().ID, "other-space.txt")

		spaceCtxA = ctxx.NewSpaceContext(tenantCtx, spaceA)

		now := time.Now()
		parentDirA.Update().SetDeletedAt(now).SaveX(spaceCtxA)
		fileInA.Data.Update().SetDeletedAt(now).SaveX(spaceCtxA)
		spaceCtxB = ctxx.NewSpaceContext(tenantCtx, spaceB)
		fileInB.Data.Update().SetDeletedAt(now).SaveX(spaceCtxB)
		spaceCtxA = ctxx.NewSpaceContext(tenantCtx, spaceA)

		reposA := harness.infra.SpaceFileRepoFactory().ForSpaceX(spaceCtxA)

		_, crossRestoreErr := reposA.Write.RestoreDeletedFile(spaceCtxA, fileInB.Data.PublicID.String())
		if crossRestoreErr == nil {
			t.Fatalf("expected cross-space restore to fail")
		}
		if !enttenant.IsNotFound(crossRestoreErr) {
			t.Fatalf("expected not found for cross-space restore, got %v", crossRestoreErr)
		}

		result, restoreErr := reposA.Write.RestoreDeletedFile(spaceCtxA, fileInA.Data.PublicID.String())
		if restoreErr != nil {
			t.Fatalf("restore deleted file: %v", restoreErr)
		}
		if result.ParentExists {
			t.Fatalf("expected missing parent to be detected")
		}
		if !result.File.IsInInbox {
			t.Fatalf("expected restored file to be moved to inbox")
		}
		if result.File.ParentID != rootA.ID {
			t.Fatalf("expected restored parent %d, got %d", rootA.ID, result.File.ParentID)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("restore test failed: %v", err)
	}
}
