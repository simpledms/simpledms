package auth

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	tenantaccessmodel "github.com/simpledms/simpledms/model/main/tenantaccess"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

// createAccountSession is shared by password, passkey, and backup code sign-in, so the
// organization check applies to every sign-in method.
func createAccountSession(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	accountx *entmain.Account,
	isTemporarySession bool,
	isSaaSModeEnabled bool,
	allowInsecureCookies bool,
) error {
	if err := ensureActiveOrganizationForSignIn(ctx, accountx, isSaaSModeEnabled); err != nil {
		return err
	}

	cookie, err := cookiex.SetSessionCookie(rw, req, isTemporarySession, allowInsecureCookies)
	if err != nil {
		log.Println(err)
		return err
	}

	deletableAt := cookiex.DeletableAt(cookie)

	_, err = ctx.VisitorCtx().MainTx.Session.
		Create().
		SetAccountID(accountx.ID).
		SetValue(cookie.Value).
		SetIsTemporarySession(isTemporarySession).
		SetDeletableAt(deletableAt).
		SetExpiresAt(cookie.Expires).
		Save(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return account.NewAccount(accountx).NilableRecordSuccessfulLogin(ctx)
}

// ensureActiveOrganizationForSignIn rejects SaaS users whose organizations are all deleted.
// Requests check this again later, but no Session must be created in the first place.
func ensureActiveOrganizationForSignIn(
	ctx ctxx.Context,
	accountx *entmain.Account,
	isSaaSModeEnabled bool,
) error {
	if !isSaaSModeEnabled || accountx.Role != mainrole.User {
		return nil
	}

	hasActiveTenantAssignment, err := tenantaccessmodel.NewTenantAccessService().HasActiveTenantAssignment(
		ctx,
		ctx.VisitorCtx().MainTx,
		accountx.ID,
	)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify organization access.")
	}
	if !hasActiveTenantAssignment {
		return e.NewHTTPErrorWithSnackbar(
			http.StatusForbidden,
			widget.NewSnackbarf("Your organization is no longer active. Please contact support."),
		)
	}

	return nil
}
