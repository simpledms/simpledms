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

type SignOutCmdData struct{}

type SignOutCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSignOutCmd(infra *common.Infra, actions *Actions) *SignOutCmd {
	config := actionx.NewConfig(
		actions.Route("sign-out-cmd"),
		false,
	)
	return &SignOutCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SignOutCmd) Data() *SignOutCmdData {
	return &SignOutCmdData{}
}

func (qq *SignOutCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
