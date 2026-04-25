package auth

import (
	"log"

	"github.com/simpledms/simpledms/core/util/cookiex"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
)

func createAccountSession(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
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
