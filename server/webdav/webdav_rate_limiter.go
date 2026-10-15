package webdav

import (
	"net"
	"strings"
	"sync"
	"time"
)

const (
	webDAVRateLimitWindow      = 5 * time.Minute
	webDAVRateLimitMaxFailures = 10
)

type webDAVRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	maxKeys  int
}

func newWebDAVRateLimiter(maxKeys int) *webDAVRateLimiter {
	return &webDAVRateLimiter{
		attempts: make(map[string][]time.Time),
		maxKeys:  maxKeys,
	}
}

// recordFailedAttempt remembers a failed authentication; blocked reports whether
// too many failures happened recently.
func (qq *webDAVRateLimiter) recordFailedAttempt(remoteAddr string, username string) {
	key := webDAVRateLimitKey(remoteAddr, username)
	if key == "" {
		return
	}

	qq.mu.Lock()
	defer qq.mu.Unlock()

	now := time.Now()
	recentFailures := qq.recentFailuresLocked(key, now)
	// further failures are not needed to keep the key blocked
	if len(recentFailures) < webDAVRateLimitMaxFailures {
		recentFailures = append(recentFailures, now)
	}
	qq.attempts[key] = recentFailures
	qq.evictLocked(now)
}

func (qq *webDAVRateLimiter) blocked(remoteAddr string, username string) bool {
	key := webDAVRateLimitKey(remoteAddr, username)
	if key == "" {
		return false
	}

	qq.mu.Lock()
	defer qq.mu.Unlock()

	recentFailures := qq.recentFailuresLocked(key, time.Now())
	if len(recentFailures) == 0 {
		delete(qq.attempts, key)
		return false
	}
	qq.attempts[key] = recentFailures
	return len(recentFailures) >= webDAVRateLimitMaxFailures
}

func (qq *webDAVRateLimiter) recentFailuresLocked(key string, now time.Time) []time.Time {
	cutoff := now.Add(-webDAVRateLimitWindow)
	history := qq.attempts[key]
	recentFailures := history[:0]
	for _, attemptedAt := range history {
		if attemptedAt.After(cutoff) {
			recentFailures = append(recentFailures, attemptedAt)
		}
	}
	return recentFailures
}

// evictLocked drops expired keys and then the keys with the oldest last failure until at
// most maxKeys remain. The linear scans are fine because the map is bounded by maxKeys
// and they only run once it is full.
func (qq *webDAVRateLimiter) evictLocked(now time.Time) {
	if qq.maxKeys <= 0 || len(qq.attempts) <= qq.maxKeys {
		return
	}

	cutoff := now.Add(-webDAVRateLimitWindow)
	for key, failures := range qq.attempts {
		if len(failures) == 0 || !failures[len(failures)-1].After(cutoff) {
			delete(qq.attempts, key)
		}
	}

	for len(qq.attempts) > qq.maxKeys {
		oldestKey := ""
		var oldestLastFailure time.Time
		for key, failures := range qq.attempts {
			lastFailure := failures[len(failures)-1]
			if oldestKey == "" || lastFailure.Before(oldestLastFailure) {
				oldestKey = key
				oldestLastFailure = lastFailure
			}
		}
		delete(qq.attempts, oldestKey)
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
