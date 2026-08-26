package webdav

import (
	"net"
	"strings"
	"sync"
	"time"
)

type webDAVRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	order    []string
	maxKeys  int
}

func newWebDAVRateLimiter(maxKeys int) *webDAVRateLimiter {
	return &webDAVRateLimiter{
		attempts: make(map[string][]time.Time),
		maxKeys:  maxKeys,
	}
}

func (qq *webDAVRateLimiter) allow(remoteAddr string, username string) bool {
	key := webDAVRateLimitKey(remoteAddr, username)
	if key == "" {
		return true
	}

	qq.mu.Lock()
	defer qq.mu.Unlock()

	if _, ok := qq.attempts[key]; !ok {
		qq.order = append(qq.order, key)
	}
	qq.trimLocked()

	now := time.Now()
	cutoff := now.Add(-5 * time.Minute)
	history := qq.attempts[key]
	filtered := make([]time.Time, 0, len(history)+1)
	for _, attemptedAt := range history {
		if attemptedAt.After(cutoff) {
			filtered = append(filtered, attemptedAt)
		}
	}
	if len(filtered) >= 10 {
		qq.attempts[key] = filtered
		return false
	}

	qq.attempts[key] = append(filtered, now)
	return true
}

func (qq *webDAVRateLimiter) blocked(remoteAddr string, username string) bool {
	key := webDAVRateLimitKey(remoteAddr, username)
	if key == "" {
		return false
	}

	qq.mu.Lock()
	defer qq.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-5 * time.Minute)
	history := qq.attempts[key]
	filtered := history[:0]
	for _, attemptedAt := range history {
		if attemptedAt.After(cutoff) {
			filtered = append(filtered, attemptedAt)
		}
	}
	qq.attempts[key] = filtered
	return len(filtered) >= 10
}

func (qq *webDAVRateLimiter) trimLocked() {
	if qq.maxKeys <= 0 {
		return
	}
	for len(qq.order) > qq.maxKeys {
		delete(qq.attempts, qq.order[0])
		qq.order = qq.order[1:]
	}
}

func webDAVRateLimitKey(remoteAddr string, username string) string {
	host := strings.TrimSpace(remoteAddr)
	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if len(username) > 128 {
		username = username[:128]
	}
	if host == "" && username == "" {
		return ""
	}
	return host + "|" + username
}
