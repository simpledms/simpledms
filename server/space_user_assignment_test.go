package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/marcobeierer/go-core/db/entmain"
	"github.com/marcobeierer/go-core/db/entx"

	"github.com/marcobeierer/go-core/ctxx"
	ctxx2 "github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model/tenant/common/spacerole"
)

func TestSpaceUser_AssignUserToSpaceCmd_RejectsDuplicateAssignment(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "owner@example.com")
	memberAccount := createTenantUser(t, harness, tenantx, "member@example.com", tenantrole.User)
	tenantDB := initTenantDB(t, harness, tenantx)

	var handlerErr error
	err := withTenantContext(t, harness, ownerAccount, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
		memberUser := ensureTenantUserForAccount(t, tenantCtx, memberAccount, tenantrole.User)

		spaceName := "Manage Users Duplicate Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		if _, err := runAssignUserToSpaceCmd(harness, spaceCtx, memberUser.PublicID.String(), spacerole.User); err != nil {
			return fmt.Errorf("initial assign user to space command: %w", err)
		}

		_, handlerErr = runAssignUserToSpaceCmd(harness, spaceCtx, memberUser.PublicID.String(), spacerole.User)
		if handlerErr == nil {
			return fmt.Errorf("expected duplicate assignment error")
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
	if !strings.Contains(httpErr.Error(), "user is already assigned to this space") {
		t.Fatalf("expected duplicate assignment error, got %q", httpErr.Error())
	}
}

func TestSpaceUser_UnassignUserFromSpaceCmd_RejectsSelfUnassignment(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	var handlerErr error
	err := withTenantContext(t, harness, ownerAccount, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
		spaceName := "Manage Users Self Unassign Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		ownerAssignment := spaceCtx.TTx.SpaceUserAssignment.Query().
			Where(spaceuserassignment.UserID(spaceCtx.User.ID)).
			OnlyX(spaceCtx)

		_, handlerErr = runUnassignUserFromSpaceCmd(harness, spaceCtx, ownerAssignment.ID)
		if handlerErr == nil {
			return fmt.Errorf("expected self-unassign error")
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
	if httpErr.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, httpErr.StatusCode())
	}
	if !strings.Contains(httpErr.Error(), "you cannot unassign yourself from a space") {
		t.Fatalf("expected self-unassign error, got %q", httpErr.Error())
	}
}

func TestSpaceUser_AssignUserToSpaceCmd_RequiresOwnerRole(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "owner@example.com")
	memberAccount := createTenantUser(t, harness, tenantx, "member@example.com", tenantrole.User)
	candidateAccount := createTenantUser(t, harness, tenantx, "candidate@example.com", tenantrole.User)
	tenantDB := initTenantDB(t, harness, tenantx)

	var spacePublicID string
	var candidatePublicID string

	err := withTenantContext(t, harness, ownerAccount, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
		memberUser := ensureTenantUserForAccount(t, tenantCtx, memberAccount, tenantrole.User)
		candidateUser := ensureTenantUserForAccount(t, tenantCtx, candidateAccount, tenantrole.User)

		spaceName := "Manage Users Owner Required Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		err := spaceCtx.TTx.SpaceUserAssignment.Create().
			SetSpaceID(spacex.ID).
			SetUserID(memberUser.ID).
			SetRole(spacerole.User).
			Exec(spaceCtx)
		if err != nil {
			return fmt.Errorf("assign member as non-owner: %w", err)
		}

		spacePublicID = spacex.PublicID.String()
		candidatePublicID = candidateUser.PublicID.String()
		return nil
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	var handlerErr error
	err = withTenantContext(t, harness, memberAccount, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx2.TenantContext) error {
		spacex := tenantCtx.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(spacePublicID))).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		_, handlerErr = runAssignUserToSpaceCmd(harness, spaceCtx, candidatePublicID, spacerole.User)
		if handlerErr == nil {
			return fmt.Errorf("expected owner-only error")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("member request setup failed: %v", err)
	}

	httpErr, ok := handlerErr.(*e.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", handlerErr)
	}
	if httpErr.StatusCode() != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, httpErr.StatusCode())
	}
	if !strings.Contains(httpErr.Error(), "you are not allowed to assign users to spaces because you aren't the owner") {
		t.Fatalf("expected owner-only error, got %q", httpErr.Error())
	}
}

func runAssignUserToSpaceCmd(
	harness *actionTestHarness,
	spaceCtx *ctxx.SpaceContext,
	userPublicID string,
	role spacerole.SpaceRole,
) (*httptest.ResponseRecorder, error) {
	form := url.Values{}
	form.Set("UserID", userPublicID)
	form.Set("Role", role.String())

	req := httptest.NewRequest(http.MethodPost, "/-/manage-users-of-space/assign-user-to-space-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	err := harness.actions.ManageSpaceUsers.AssignUserToSpaceCmd.Handler(
		httpx2.NewResponseWriter(rr),
		httpx2.NewRequest(req),
		spaceCtx,
	)

	return rr, err
}

func runUnassignUserFromSpaceCmd(
	harness *actionTestHarness,
	spaceCtx *ctxx.SpaceContext,
	userAssignmentID int64,
) (*httptest.ResponseRecorder, error) {
	form := url.Values{}
	form.Set("UserAssignmentID", fmt.Sprintf("%d", userAssignmentID))

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/manage-users-of-space/unassign-user-from-space-cmd",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	err := harness.actions.ManageSpaceUsers.UnassignUserFromSpaceCmd.Handler(
		httpx2.NewResponseWriter(rr),
		httpx2.NewRequest(req),
		spaceCtx,
	)

	return rr, err
}

func ensureTenantUserForAccount(
	t testing.TB,
	tenantCtx *ctxx2.TenantContext,
	accountx *entmain.Account,
	role tenantrole.TenantRole,
) *enttenant.User {
	t.Helper()

	exists := tenantCtx.TTx.User.Query().Where(user.AccountID(accountx.ID)).ExistX(tenantCtx)
	if exists {
		return tenantCtx.TTx.User.Query().Where(user.AccountID(accountx.ID)).OnlyX(tenantCtx)
	}

	return tenantCtx.TTx.User.Create().
		SetAccountID(accountx.ID).
		SetRole(role).
		SetEmail(accountx.Email).
		SetFirstName(accountx.FirstName).
		SetLastName(accountx.LastName).
		SaveX(tenantCtx)
}
