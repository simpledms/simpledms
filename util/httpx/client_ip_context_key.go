package httpx

import (
	"context"
	"net/http"
	"net/netip"
)

type clientIPContextKey struct{}

// WithClientIP stores the client address resolved by the router. Only the router knows
// the trusted proxy configuration needed to interpret X-Forwarded-For safely.
func WithClientIP(req *http.Request, clientIP netip.Addr) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), clientIPContextKey{}, clientIP))
}
