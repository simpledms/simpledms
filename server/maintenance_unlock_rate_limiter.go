package server

import (
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/simpledms/simpledms/util/httpx"
)

const (
	maintenanceUnlockAttemptLimit  = 5
	maintenanceUnlockAttemptWindow = time.Minute
	maintenanceUnlockClientLimit   = 4096
	maintenanceUnlockOverflowKey   = "overflow"
)

type maintenanceUnlockRateLimiter struct {
	mu             sync.Mutex
	attempts       map[string][]time.Time
	trustedProxies []netip.Prefix
}

func newMaintenanceUnlockRateLimiter(
	trustedProxies []netip.Prefix,
) *maintenanceUnlockRateLimiter {
	return &maintenanceUnlockRateLimiter{
		attempts:       make(map[string][]time.Time),
		trustedProxies: append([]netip.Prefix(nil), trustedProxies...),
	}
}

func (qq *maintenanceUnlockRateLimiter) allow(
	req *http.Request,
	now time.Time,
) (int, bool) {
	clientKey := qq.clientKey(req)

	qq.mu.Lock()
	defer qq.mu.Unlock()

	qq.pruneExpiredLocked(now)
	if _, exists := qq.attempts[clientKey]; !exists && len(qq.attempts) >= maintenanceUnlockClientLimit-1 {
		clientKey = maintenanceUnlockOverflowKey
	}
	clientAttempts := qq.attempts[clientKey]
	if len(clientAttempts) >= maintenanceUnlockAttemptLimit {
		retryAfter := clientAttempts[0].Add(maintenanceUnlockAttemptWindow).Sub(now)
		return retryAfterSeconds(retryAfter), false
	}
	qq.attempts[clientKey] = append(clientAttempts, now)
	return 0, true
}

func (qq *maintenanceUnlockRateLimiter) pruneExpiredLocked(now time.Time) {
	cutoff := now.Add(-maintenanceUnlockAttemptWindow)
	for clientKey, clientAttempts := range qq.attempts {
		firstActive := 0
		for firstActive < len(clientAttempts) && !clientAttempts[firstActive].After(cutoff) {
			firstActive++
		}
		if firstActive == len(clientAttempts) {
			delete(qq.attempts, clientKey)
			continue
		}

		qq.attempts[clientKey] = clientAttempts[firstActive:]
	}
}

func (qq *maintenanceUnlockRateLimiter) clientKey(req *http.Request) string {
	clientIP, ok := httpx.ClientIPThroughTrustedProxies(req, qq.trustedProxies)
	if !ok {
		return "unknown"
	}
	return clientIP.String()
}

func retryAfterSeconds(retryAfter time.Duration) int {
	seconds := int((retryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}
