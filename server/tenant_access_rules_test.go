package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/db/entmain/account"
	entmainschema "github.com/simpledms/simpledms/db/entmain/schema"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/accountutil"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestSignInCmdRejectsUserWithoutActiveTenantAssignment(t *testing.T) {
	harness := newActionTestHarness(t)

	accountx, tenantx := signUpAccount(t, harness, "inactive-tenant-owner@example.com")
	password := "supersecret"
	setAccountPasswordForTest(t, harness, accountx.ID, password)

	harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
		SetDeletedAt(time.Now()).
		SetDeletedBy(accountx.ID).
		ExecX(context.Background())

	form := url.Values{}
	form.Set("Email", accountx.Email.String())
	form.Set("Password", password)
	form.Set("TwoFactorAuthenticationCode", "")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().
		Where(session.AccountID(accountx.ID)).
		CountX(context.Background())
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions, got %d", sessionCount)
	}
}

func TestDeleteUserCmdOwningTenantDeletesAccountGlobally(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "tenant-owner-delete-user@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	memberEmail := "member-to-delete@example.com"
	createAccount(t, harness.mainDB, memberEmail, "supersecret")
	memberAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(memberEmail))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantx.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(true).
		SetIsDefault(false).
		SaveX(context.Background())

	memberTenantUser := tenantDB.ReadWriteConn.User.Create().
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetEmail(memberAccount.Email).
		SetFirstName(memberAccount.FirstName).
		SetLastName(memberAccount.LastName).
		SaveX(context.Background())

	sessionValue := createSessionForAccountForRulesTest(t, harness, ownerAccount.ID)

	form := url.Values{}
	form.Set("UserID", memberTenantUser.PublicID.String())

	req := httptest.NewRequest(http.MethodPost, "/-/manage-org-users/delete-user-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.ManageUsersOfTenant(tenantx.PublicID.String()))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: sessionValue,
		Path:  "/",
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if trigger := rr.Header().Get("HX-Trigger"); trigger != event.UserDeleted.String() {
		t.Fatalf("expected HX-Trigger %q, got %q", event.UserDeleted.String(), trigger)
	}

	memberAccountAfter := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.ID(memberAccount.ID)).
		OnlyX(entmainschema.SkipSoftDelete(context.Background()))
	if memberAccountAfter.DeletedAt.IsZero() {
		t.Fatal("expected member account to be soft deleted")
	}

	memberTenantUserAfter := tenantDB.ReadWriteConn.User.Query().
		Where(user.ID(memberTenantUser.ID)).
		OnlyX(enttenantschema.SkipSoftDelete(context.Background()))
	if memberTenantUserAfter.DeletedAt.IsZero() {
		t.Fatal("expected tenant user to be soft deleted")
	}

	assignmentCount := harness.mainDB.ReadWriteConn.TenantAccountAssignment.Query().
		Where(tenantaccountassignment.AccountID(memberAccount.ID)).
		CountX(context.Background())
	if assignmentCount != 0 {
		t.Fatalf("expected 0 tenant assignments across all tenants, got %d", assignmentCount)
	}
}

func TestDeleteUserCmdDeletedAccountCannotSignIn(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "tenant-owner-login-block@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	memberEmail := "member-login-block@example.com"
	memberPassword := "supersecret"
	createAccount(t, harness.mainDB, memberEmail, memberPassword)
	memberAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(memberEmail))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantx.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(true).
		SetIsDefault(false).
		SaveX(context.Background())

	memberTenantUser := tenantDB.ReadWriteConn.User.Create().
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetEmail(memberAccount.Email).
		SetFirstName(memberAccount.FirstName).
		SetLastName(memberAccount.LastName).
		SaveX(context.Background())

	ownerSessionValue := createSessionForAccountForRulesTest(t, harness, ownerAccount.ID)

	deleteForm := url.Values{}
	deleteForm.Set("UserID", memberTenantUser.PublicID.String())

	deleteReq := httptest.NewRequest(http.MethodPost, "/-/manage-org-users/delete-user-cmd", strings.NewReader(deleteForm.Encode()))
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteReq.Header.Set("HX-Request", "true")
	deleteReq.Header.Set("HX-Current-URL", route.ManageUsersOfTenant(tenantx.PublicID.String()))
	deleteReq.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: ownerSessionValue,
		Path:  "/",
	})

	deleteRR := httptest.NewRecorder()
	harness.router.ServeHTTP(deleteRR, deleteReq)

	if deleteRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, deleteRR.Code)
	}

	signInForm := url.Values{}
	signInForm.Set("Email", memberEmail)
	signInForm.Set("Password", memberPassword)
	signInForm.Set("TwoFactorAuthenticationCode", "")

	signInReq := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(signInForm.Encode()))
	signInReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signInReq.Header.Set("HX-Request", "true")

	signInRR := httptest.NewRecorder()
	harness.router.ServeHTTP(signInRR, signInReq)

	if signInRR.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, signInRR.Code)
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().
		Where(session.AccountID(memberAccount.ID)).
		CountX(context.Background())
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions, got %d", sessionCount)
	}
}

func TestDeleteUserCmdRemovesMembershipWhenUserBelongsToMultipleTenants(t *testing.T) {
	harness := newActionTestHarness(t)

	_, owningTenant := signUpAccount(t, harness, "tenant-owner-owning-org@example.com")
	nonOwningTenantOwnerAccount, nonOwningTenant := signUpAccount(t, harness, "tenant-owner-non-owning-org@example.com")
	nonOwningTenantDB := initTenantDB(t, harness, nonOwningTenant)

	memberEmail := "member-multi-org@example.com"
	createAccount(t, harness.mainDB, memberEmail, "supersecret")
	memberAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(memberEmail))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(owningTenant.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(true).
		SetIsDefault(false).
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(nonOwningTenant.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(false).
		SetIsDefault(false).
		SaveX(context.Background())

	memberTenantUser := nonOwningTenantDB.ReadWriteConn.User.Create().
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetEmail(memberAccount.Email).
		SetFirstName(memberAccount.FirstName).
		SetLastName(memberAccount.LastName).
		SaveX(context.Background())

	sessionValue := createSessionForAccountForRulesTest(t, harness, nonOwningTenantOwnerAccount.ID)

	form := url.Values{}
	form.Set("UserID", memberTenantUser.PublicID.String())

	req := httptest.NewRequest(http.MethodPost, "/-/manage-org-users/delete-user-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.ManageUsersOfTenant(nonOwningTenant.PublicID.String()))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: sessionValue,
		Path:  "/",
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	memberAccountAfter := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.ID(memberAccount.ID)).
		OnlyX(context.Background())
	if !memberAccountAfter.DeletedAt.IsZero() {
		t.Fatal("expected member account not to be deleted")
	}

	assignmentCount := harness.mainDB.ReadWriteConn.TenantAccountAssignment.Query().
		Where(tenantaccountassignment.AccountID(memberAccount.ID)).
		CountX(context.Background())
	if assignmentCount != 1 {
		t.Fatalf("expected 1 tenant assignment, got %d", assignmentCount)
	}

	remainingAssignmentCount := harness.mainDB.ReadWriteConn.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(owningTenant.ID),
			tenantaccountassignment.AccountID(memberAccount.ID),
		).
		CountX(context.Background())
	if remainingAssignmentCount != 1 {
		t.Fatalf("expected remaining assignment in owning tenant, got %d", remainingAssignmentCount)
	}

	nonOwningAssignmentCount := harness.mainDB.ReadWriteConn.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(nonOwningTenant.ID),
			tenantaccountassignment.AccountID(memberAccount.ID),
		).
		CountX(context.Background())
	if nonOwningAssignmentCount != 0 {
		t.Fatalf("expected removed assignment in non-owning tenant, got %d", nonOwningAssignmentCount)
	}
}

func TestDeleteUserCmdBlocksTenantOwnerDeletingSelf(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, tenantx := signUpAccount(t, harness, "tenant-owner-self-delete@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	ownerTenantUser := tenantDB.ReadWriteConn.User.Query().
		Where(user.AccountID(ownerAccount.ID)).
		OnlyX(context.Background())

	sessionValue := createSessionForAccountForRulesTest(t, harness, ownerAccount.ID)

	form := url.Values{}
	form.Set("UserID", ownerTenantUser.PublicID.String())

	req := httptest.NewRequest(http.MethodPost, "/-/manage-org-users/delete-user-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.ManageUsersOfTenant(tenantx.PublicID.String()))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: sessionValue,
		Path:  "/",
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rr.Code)
	}

	ownerAccountAfter := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.ID(ownerAccount.ID)).
		OnlyX(context.Background())
	if !ownerAccountAfter.DeletedAt.IsZero() {
		t.Fatal("expected owner account not to be deleted")
	}
}

func TestEditAccountCmdAllowsOwningTenantAdmin(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerAccount, owningTenant := signUpAccount(t, harness, "tenant-owner-edit-account@example.com")

	memberEmail := "member-editable@example.com"
	createAccount(t, harness.mainDB, memberEmail, "supersecret")
	memberAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(memberEmail))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(owningTenant.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(true).
		SetIsDefault(false).
		SaveX(context.Background())

	sessionValue := createSessionForAccountForRulesTest(t, harness, ownerAccount.ID)

	form := url.Values{}
	form.Set("AccountID", memberAccount.PublicID.String())
	form.Set("Language", language.English.String())
	form.Set("SubscribeToNewsletter", "false")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/edit-account-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: sessionValue,
		Path:  "/",
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	memberAccountAfter := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.ID(memberAccount.ID)).
		OnlyX(context.Background())
	if memberAccountAfter.Language != language.English {
		t.Fatalf("expected language %q, got %q", language.English, memberAccountAfter.Language)
	}
}

func TestEditAccountCmdAllowsSelfEditForNonAdminMember(t *testing.T) {
	harness := newActionTestHarness(t)

	_, owningTenant := signUpAccount(t, harness, "tenant-owner-self-edit-setup@example.com")

	memberEmail := "member-self-edit@example.com"
	createAccount(t, harness.mainDB, memberEmail, "supersecret")
	memberAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(memberEmail))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(owningTenant.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(true).
		SetIsDefault(false).
		SaveX(context.Background())

	sessionValue := createSessionForAccountForRulesTest(t, harness, memberAccount.ID)

	form := url.Values{}
	form.Set("AccountID", memberAccount.PublicID.String())
	form.Set("Language", language.English.String())
	form.Set("SubscribeToNewsletter", "false")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/edit-account-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: sessionValue,
		Path:  "/",
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	memberAccountAfter := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.ID(memberAccount.ID)).
		OnlyX(context.Background())
	if memberAccountAfter.Language != language.English {
		t.Fatalf("expected language %q, got %q", language.English, memberAccountAfter.Language)
	}
}

func TestEditAccountCmdBlocksNonOwningTenantAdmin(t *testing.T) {
	harness := newActionTestHarness(t)

	_, owningTenant := signUpAccount(t, harness, "tenant-owner-owning-edit@example.com")
	nonOwningTenantOwner, nonOwningTenant := signUpAccount(t, harness, "tenant-owner-non-owning-edit@example.com")

	memberEmail := "member-non-owning-edit@example.com"
	createAccount(t, harness.mainDB, memberEmail, "supersecret")
	memberAccount := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(memberEmail))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(owningTenant.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(true).
		SetIsDefault(false).
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(nonOwningTenant.ID).
		SetAccountID(memberAccount.ID).
		SetRole(tenantrole.User).
		SetIsOwningTenant(false).
		SetIsDefault(false).
		SaveX(context.Background())

	sessionValue := createSessionForAccountForRulesTest(t, harness, nonOwningTenantOwner.ID)

	form := url.Values{}
	form.Set("AccountID", memberAccount.PublicID.String())
	form.Set("Language", language.English.String())
	form.Set("SubscribeToNewsletter", "false")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/edit-account-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: sessionValue,
		Path:  "/",
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	memberAccountAfter := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.ID(memberAccount.ID)).
		OnlyX(context.Background())
	if memberAccountAfter.Language == language.English {
		t.Fatalf("expected language not to be updated to %q", language.English)
	}
}

func createSessionForAccountForRulesTest(
	t testing.TB,
	harness *actionTestHarness,
	accountID int64,
) string {
	t.Helper()

	sessionValue := fmt.Sprintf("rules-session-%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(14 * 24 * time.Hour)

	_, err := harness.mainDB.ReadWriteConn.Session.Create().
		SetValue(sessionValue).
		SetAccountID(accountID).
		SetIsTemporarySession(false).
		SetExpiresAt(expiresAt).
		SetDeletableAt(expiresAt).
		Save(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	return sessionValue
}

func setAccountPasswordForTest(t testing.TB, harness *actionTestHarness, accountID int64, password string) {
	t.Helper()

	salt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate password salt")
	}

	harness.mainDB.ReadWriteConn.Account.UpdateOneID(accountID).
		SetPasswordSalt(salt).
		SetPasswordHash(accountutil.PasswordHash(password, salt)).
		SetTemporaryPasswordSalt("").
		SetTemporaryPasswordHash("").
		SetTemporaryPasswordExpiresAt(time.Time{}).
		ExecX(context.Background())
}
