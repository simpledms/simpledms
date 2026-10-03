package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	credentialmodel "github.com/simpledms/simpledms/model/main/webdavcredential"
)

func TestWebDAVCredentialStopsWorkingAfterUserIsRemovedFromSpace(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)
	ownerAccount, tenantx := signUpAccount(t, harness, "webdav-space-owner@example.com")
	memberAccount := createTenantUser(
		t, harness, tenantx, "webdav-space-member@example.com", tenantrole.User,
	)
	tenantDB := initTenantDB(t, harness, tenantx)

	ownerMainTx, ownerTenantTx, ownerCtx := newTenantContext(t, harness, ownerAccount, tenantx, tenantDB)
	createSpaceViaCmd(t, harness.actions, ownerCtx, "Member Space")
	spacex := ownerTenantTx.Space.Query().Where(space.Name("Member Space")).OnlyX(ownerCtx)
	ownerSpaceCtx := ctxx.NewSpaceContext(ownerCtx, spacex)
	memberUser := ownerTenantTx.User.Query().Where(user.AccountID(memberAccount.ID)).OnlyX(ownerCtx)
	ownerTenantTx.SpaceUserAssignment.Create().
		SetSpaceID(spacex.ID).
		SetUserID(memberUser.ID).
		SetRole(spacerole.User).
		SaveX(ownerSpaceCtx)
	commitTestTxs(t, ownerMainTx, ownerTenantTx)

	memberMainTx, memberTenantTx, memberCtx := newTenantContext(t, harness, memberAccount, tenantx, tenantDB)
	memberSpace := memberTenantTx.Space.Query().Where(space.ID(spacex.ID)).OnlyX(memberCtx)
	result, err := credentialmodel.NewCredentialService().CreateOwnerCredential(
		ctxx.NewSpaceContext(memberCtx, memberSpace),
		"Scanner",
		webDAVTestURL(tenantx, spacex, "/"),
		0,
		false,
	)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	commitTestTxs(t, memberMainTx, memberTenantTx)

	propfind := func() int {
		return webDAVRequest(
			t, harness, result.Username, result.Secret, "PROPFIND",
			webDAVTestURL(tenantx, spacex, "/"), strings.NewReader(webDAVPropfindBody),
			func(req *http.Request) { req.Header.Set("Depth", "0") },
		).Code
	}
	if got := propfind(); got != http.StatusMultiStatus {
		t.Fatalf("expected assigned member to use WebDAV, got %d", got)
	}

	ownerMainTx, ownerTenantTx, ownerCtx = newTenantContext(t, harness, ownerAccount, tenantx, tenantDB)
	ownerTenantTx.SpaceUserAssignment.Delete().
		Where(spaceuserassignment.UserID(memberUser.ID)).
		ExecX(ctxx.NewSpaceContext(ownerCtx, spacex))
	commitTestTxs(t, ownerMainTx, ownerTenantTx)

	if got := propfind(); got != http.StatusForbidden {
		t.Fatalf("expected removed member to be forbidden, got %d", got)
	}
}

func commitTestTxs(t *testing.T, mainTx *entmain.Tx, tenantTx *enttenant.Tx) {
	t.Helper()
	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}
}
