package server

import (
	"net/http"
	"time"
)

const readHeaderTimeout = 20 * time.Second

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
