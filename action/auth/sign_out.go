package auth

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SignOutData struct{}

type SignOut struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSignOut(infra *common.Infra, actions *Actions) *SignOut {
	config := actionx.NewConfig(
		actions.Route("sign-out"),
		false,
	)
	return &SignOut{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SignOut) Data() *SignOutData {
	return &SignOutData{}
}

func (qq *SignOut) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	cookie, err := req.Cookie(cookiex.SessionCookieName())
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusBadRequest, "Invalid session cookie.")
	}

	cookiex.InvalidateSessionCookie(rw, qq.infra.SystemConfig().AllowInsecureCookies())
	// cookie value can be used directly because user is authenticated if MainCtx is used
	ctx.MainCtx().MainTx.Session.Delete().Where(session.Value(cookie.Value)).ExecX(ctx)

	rw.Header().Set("HX-Redirect", "/")
	return nil
}
