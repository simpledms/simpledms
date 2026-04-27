package auth

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/simpledms/simpledms/util/httpx"
)

const (
	signInRateLimitWindow       = time.Minute
	signInRateLimitPerIP        = 20
	signInRateLimitPerEmail     = 8
	resetRateLimitWindow        = 10 * time.Minute
	resetRateLimitPerIP         = 10
	resetRateLimitPerEmail      = 3
	passkeyBeginRateLimitWindow = time.Minute
	passkeyBeginRateLimitPerIP  = 40
	passkeyBeginRateLimitGlobal = 120
)

func clientIPFromRequest(req *httpx.Request) string {
	remoteAddr := strings.TrimSpace(req.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = host
	}

	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	return strings.ToLower(remoteAddr)
}

func normalizeRateLimitedEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func rateLimitKey(scope, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	return fmt.Sprintf("%s:%s", strings.TrimSpace(scope), value)
}
