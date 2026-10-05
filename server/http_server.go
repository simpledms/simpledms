package server

import (
	"net/http"
	"time"

	"github.com/gorilla/handlers"
)

const (
	readHeaderTimeout = 20 * time.Second
	idleTimeout       = 2 * time.Minute
)

// newHTTPServer is used for every listener, including maintenance mode and autocert.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
		// Body timeouts would abort large uploads and downloads. Bound header reads and
		// idle keep-alive connections so that slow or idle clients cannot hold
		// connections open indefinitely; without IdleTimeout, idle connections never expire.
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// newPublicHandler wraps the app and maintenance handlers with compression, panic
// recovery, and security headers.
func newPublicHandler(next http.Handler) http.Handler {
	return handlers.CompressHandler(
		handlers.RecoveryHandler(
			handlers.PrintRecoveryStack(true),
		)(
			securityHeadersHandler(next),
			// handlers.LoggingHandler(),
		),
	)
}

// securityHeadersHandler sets defaults for every response. Handlers serving uploaded
// files additionally sandbox active content.
func securityHeadersHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// The app embeds its own previews only; deny framing by other origins (clickjacking).
		rw.Header().Set("X-Frame-Options", "SAMEORIGIN")
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(rw, req)
	})
}
