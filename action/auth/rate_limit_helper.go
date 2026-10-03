package auth

import (
	"fmt"
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
	return strings.ToLower(req.ClientIP())
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
