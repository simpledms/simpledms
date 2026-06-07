package auth

import (
	"log"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func createAccountSession(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	accountx *entmain.Account,
	isTemporarySession bool,
	allowInsecureCookies bool,
) error {
	cookie, err := cookiex.SetSessionCookie(rw, req, isTemporarySession, allowInsecureCookies)
	if err != nil {
		log.Println(err)
		return err
	}

	deletableAt := cookiex.DeletableAt(cookie)

	ctx.VisitorCtx().MainTx.Session.
		Create().
		SetAccountID(accountx.ID).
		SetValue(cookie.Value).
		SetIsTemporarySession(isTemporarySession).
		SetDeletableAt(deletableAt).
		SetExpiresAt(cookie.Expires).
		SaveX(ctx)

	return nil
}
