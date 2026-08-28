package server

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/txx"
)

func TestBrowserUploadFinalizationRejectsRemovedTenantAssignment(t *testing.T) {
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "revoked-upload@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, accountx, tenantx, tenantDB)
	createSpaceViaCmd(t, harness.actions, tenantCtx, "Revoked Upload")
	spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Revoked Upload")).OnlyX(tenantCtx)
	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}

	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
	if err != nil {
		t.Fatalf("create upload context: %v", err)
	}
	spacex = tenantCtx.TTx.Space.GetX(tenantCtx, spacex.ID)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit upload main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit upload tenant tx: %v", err)
	}

	assignment := harness.mainDB.ReadWriteConn.TenantAccountAssignment.Query().Where(
		tenantaccountassignment.AccountID(accountx.ID),
		tenantaccountassignment.TenantID(tenantx.ID),
	).OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.TenantAccountAssignment.DeleteOneID(assignment.ID).
		ExecX(context.Background())

	finalized := false
	_, err = txx.WithFreshAuthorizedTenantWriteSpaceTx(
		spaceCtx,
		func(*ctxx.SpaceContext) (*struct{}, error) {
			finalized = true
			return &struct{}{}, nil
		},
	)
	if err == nil {
		t.Fatal("expected finalization authorization error")
	}
	var httpErr *e.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected forbidden HTTP error, got %v", err)
	}
	if finalized {
		t.Fatal("upload was finalized after tenant assignment removal")
	}
}
