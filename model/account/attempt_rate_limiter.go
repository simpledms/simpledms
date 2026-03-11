package account

import (
	"sync"
	"time"
)

type attemptRateLimiter struct {
	mu          sync.Mutex
	attemptedAt map[string]time.Time
}

func newAttemptRateLimiter() *attemptRateLimiter {
	return &attemptRateLimiter{
		attemptedAt: make(map[string]time.Time),
	}
}

func (qq *attemptRateLimiter) Allow(key string, window time.Duration) bool {
	qq.mu.Lock()
	defer qq.mu.Unlock()

	lastAttemptAt, ok := qq.attemptedAt[key]
	if !ok {
		return true
	}

	if lastAttemptAt.After(time.Now().Add(-window)) {
		return false
	}

	delete(qq.attemptedAt, key)
	return true
}

func (qq *attemptRateLimiter) Record(key string) {
	qq.mu.Lock()
	defer qq.mu.Unlock()

	qq.attemptedAt[key] = time.Now()
}
