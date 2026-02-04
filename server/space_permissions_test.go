package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestSpaceCreatePermissions(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "owner@example.com")
	userAccount := createTenantUser(t, harness, tenantx, "user@example.com", tenantrole.User)
	tenantDB := initTenantDB(t, harness, tenantx)

	testCases := []struct {
		name      string
		accountx  *entmain.Account
		expectErr bool
	}{
		{
			name:      "owner",
			accountx:  ownerAccount,
			expectErr: false,
		},
		{
			name:      "user",
			accountx:  userAccount,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			spaceName := fmt.Sprintf("Create Space %s", tc.name)

			err := withTenantContext(t, harness, tc.accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
				form := url.Values{}
				form.Set("Name", spaceName)
				form.Set("Description", "Test description")
				form.Set("AddMeAsSpaceOwner", "true")

				req := httptest.NewRequest(http.MethodPost, "/-/spaces/create-space-cmd", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				rr := httptest.NewRecorder()
				err := harness.actions.Spaces.CreateSpaceCmd.Handler(
					httpx.NewResponseWriter(rr),
					httpx.NewRequest(req),
					tenantCtx,
				)
				if err != nil {
					return err
				}
				return nil
			})
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
			} else if err != nil {
				t.Fatalf("create space: %v", err)
			}

			verifyMainTx, verifyTenantTx, verifyCtx := newTenantContext(t, harness, ownerAccount, tenantx, tenantDB)
			defer func() {
				_ = verifyTenantTx.Rollback()
				_ = verifyMainTx.Rollback()
			}()
			exists := verifyCtx.TTx.Space.Query().Where(space.Name(spaceName)).ExistX(verifyCtx)
			if tc.expectErr && exists {
				t.Fatalf("expected space to be blocked")
			}
			if !tc.expectErr && !exists {
				t.Fatalf("expected space to be created")
			}
		})
	}
}

func TestSpaceDeletePermissions(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "owner@example.com")
	userAccount := createTenantUser(t, harness, tenantx, "user@example.com", tenantrole.User)
	tenantDB := initTenantDB(t, harness, tenantx)

	testCases := []struct {
		name      string
		accountx  *entmain.Account
		expectErr bool
	}{
		{
			name:      "owner",
			accountx:  ownerAccount,
			expectErr: false,
		},
		{
			name:      "user",
			accountx:  userAccount,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			spaceID, spaceEntityID := createSpaceForDelete(t, harness, tenantx, tenantDB, ownerAccount, tc.name)

			err := withTenantContext(t, harness, tc.accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
				form := url.Values{}
				form.Set("SpaceID", spaceID)

				req := httptest.NewRequest(http.MethodPost, "/-/spaces/delete-space-cmd", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				rr := httptest.NewRecorder()
				err := harness.actions.Spaces.DeleteSpaceCmd.Handler(
					httpx.NewResponseWriter(rr),
					httpx.NewRequest(req),
					tenantCtx,
				)
				if err != nil {
					return err
				}
				return nil
			})
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
			} else if err != nil {
				t.Fatalf("delete space: %v", err)
			}

			verifyMainTx, verifyTenantTx, verifyCtx := newTenantContext(t, harness, ownerAccount, tenantx, tenantDB)
			defer func() {
				_ = verifyTenantTx.Rollback()
				_ = verifyMainTx.Rollback()
			}()
			ctxWithDeleted := schema.SkipSoftDelete(verifyCtx)
			spacex := verifyCtx.TTx.Space.Query().Where(space.ID(spaceEntityID)).OnlyX(ctxWithDeleted)
			isDeleted := !spacex.DeletedAt.IsZero()
			if tc.expectErr && isDeleted {
				t.Fatalf("expected space deletion to be blocked")
			}
			if !tc.expectErr && !isDeleted {
				t.Fatalf("expected space to be deleted")
			}
		})
	}
}

func createTenantUser(
	t *testing.T,
	harness *actionTestHarness,
	tenantx *entmain.Tenant,
	email string,
	role tenantrole.TenantRole,
) *entmain.Account {
	t.Helper()

	createAccount(t, harness.mainDB, email, "supersecret")

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantx.ID).
		SetAccountID(accountx.ID).
		SetRole(role).
		SetIsDefault(false).
		SaveX(context.Background())

	return accountx
}

func createSpaceForDelete(
	t *testing.T,
	harness *actionTestHarness,
	tenantx *entmain.Tenant,
	tenantDB *sqlx.TenantDB,
	ownerAccount *entmain.Account,
	suffix string,
) (string, int64) {
	t.Helper()

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, ownerAccount, tenantx, tenantDB)
	spaceName := fmt.Sprintf("Delete Space %s", suffix)
	createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

	spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
	spaceID := spacex.PublicID.String()
	spaceEntityID := spacex.ID

	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}

	return spaceID, spaceEntityID
}
