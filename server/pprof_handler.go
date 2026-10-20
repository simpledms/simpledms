package server

import (
	"errors"
	"net/http"
	"net/http/pprof"
	"regexp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

var pprofPathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// PprofHandler serves runtime diagnostics using the application's authorization flow.
type PprofHandler struct {
	router  *Router
	handler http.Handler
}

// NewPprofHandler creates the profiling subtree for the application router.
func NewPprofHandler(router *Router, path string) (*PprofHandler, error) {
	if !pprofPathPattern.MatchString(path) {
		return nil, errors.New("SIMPLEDMS_PPROF_PATH must contain only letters, digits, '-' or '_'; profiling disabled")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	return &PprofHandler{
		router:  router,
		handler: http.StripPrefix("/"+path, mux),
	}, nil
}

// ServeHTTP authorizes the request before dispatching it to a standard pprof handler.
func (qq *PprofHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("Referrer-Policy", "no-referrer")
	authorized := false
	authorization := newTransactionResponse()
	qq.router.wrapTx(func(_ httpx.ResponseWriter, _ *httpx.Request, ctx ctxx.Context) error {
		if ctx.VisitorCtx().IsTemporarySession || ctx.MainCtx().Account.Role != mainrole.Admin {
			return e.NewHTTPErrorf(http.StatusForbidden, "System administrator sign-in is required.")
		}
		authorized = true
		return nil
	}, true)(authorization, req)
	if !authorized || authorization.status != 0 {
		authorization.flush(rw)
		return
	}
	for key, values := range authorization.Header() {
		rw.Header()[key] = values
	}
	// Complete authorization (including its commit) before profiling, so a long capture
	// neither holds a database transaction open nor buffers the profiling response.
	qq.handler.ServeHTTP(rw, req)
}
