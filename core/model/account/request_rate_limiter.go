package account

import (
	"strings"
	"sync"
	"time"
)

type RequestRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func NewRequestRateLimiter() *RequestRateLimiter {
	return &RequestRateLimiter{
		attempts: make(map[string][]time.Time),
	}
}

func (qq *RequestRateLimiter) Allow(key string, window time.Duration, limit int) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return true
	}
	if limit <= 0 {
		return true
	}

	qq.mu.Lock()
	defer qq.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	history := qq.attempts[key]
	filteredHistory := make([]time.Time, 0, len(history)+1)
	for _, attemptedAt := range history {
		if attemptedAt.After(cutoff) {
			filteredHistory = append(filteredHistory, attemptedAt)
		}
	}

	if len(filteredHistory) >= limit {
		qq.attempts[key] = filteredHistory
		return false
	}

	filteredHistory = append(filteredHistory, now)
	qq.attempts[key] = filteredHistory

	return true
}
