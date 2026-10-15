package webdav

import (
	"fmt"
	"testing"
	"time"
)

func TestWebDAVRateLimiterBlocksAfterTenRecentFailures(t *testing.T) {
	limiter := newWebDAVRateLimiter(10)

	for i := 0; i < 9; i++ {
		limiter.recordFailedAttempt("192.0.2.1:1234", "alice")
	}
	if limiter.blocked("192.0.2.1:5678", "Alice") {
		t.Fatal("blocked after 9 failures")
	}

	limiter.recordFailedAttempt("192.0.2.1:1234", "alice")
	if !limiter.blocked("192.0.2.1:5678", "Alice") {
		t.Fatal("not blocked after 10 failures")
	}
	if limiter.blocked("192.0.2.2:1234", "alice") {
		t.Fatal("failures of another address blocked alice")
	}
}

func TestWebDAVRateLimiterCheckingUnknownCredentialsKeepsNoState(t *testing.T) {
	limiter := newWebDAVRateLimiter(10)

	if limiter.blocked("192.0.2.1:1234", "alice") {
		t.Fatal("unknown credentials are blocked")
	}
	if len(limiter.attempts) != 0 {
		t.Fatalf("expected no tracked keys, got %d", len(limiter.attempts))
	}
}

func TestWebDAVRateLimiterForgetsExpiredFailures(t *testing.T) {
	limiter := newWebDAVRateLimiter(10)
	key := webDAVRateLimitKey("192.0.2.1:1234", "alice")
	expired := time.Now().Add(-webDAVRateLimitWindow - time.Second)
	for i := 0; i < webDAVRateLimitMaxFailures; i++ {
		limiter.attempts[key] = append(limiter.attempts[key], expired)
	}

	if limiter.blocked("192.0.2.1:1234", "alice") {
		t.Fatal("blocked by expired failures")
	}
	if _, ok := limiter.attempts[key]; ok {
		t.Fatal("expired key is still tracked")
	}
}

func TestWebDAVRateLimiterTracksAtMostMaxKeys(t *testing.T) {
	limiter := newWebDAVRateLimiter(3)

	for i := 0; i < 10; i++ {
		username := fmt.Sprintf("user%d", i)
		limiter.blocked("192.0.2.1:1234", username)
		limiter.recordFailedAttempt("192.0.2.1:1234", username)
	}

	if len(limiter.attempts) > 3 {
		t.Fatalf("expected at most 3 tracked keys, got %d", len(limiter.attempts))
	}
}

func TestWebDAVRateLimiterEvictsExpiredKeysBeforeActiveOnes(t *testing.T) {
	limiter := newWebDAVRateLimiter(2)
	for i := 0; i < webDAVRateLimitMaxFailures; i++ {
		limiter.recordFailedAttempt("192.0.2.1:1234", "active")
	}
	expiredKey := webDAVRateLimitKey("192.0.2.1:1234", "expired")
	limiter.attempts[expiredKey] = []time.Time{
		time.Now().Add(-webDAVRateLimitWindow - time.Second),
	}

	limiter.recordFailedAttempt("192.0.2.1:1234", "new")

	if !limiter.blocked("192.0.2.1:1234", "active") {
		t.Fatal("active key was evicted while an expired key was available")
	}
	if _, ok := limiter.attempts[expiredKey]; ok {
		t.Fatal("expired key was kept")
	}
}

func TestWebDAVRateLimiterEvictsKeyWithOldestLastFailureWhenFull(t *testing.T) {
	limiter := newWebDAVRateLimiter(2)
	now := time.Now()
	olderKey := webDAVRateLimitKey("192.0.2.1:1234", "older")
	newerKey := webDAVRateLimitKey("192.0.2.1:1234", "newer")
	// older failed first but most recently; newer was first seen later but failed earlier
	limiter.attempts[olderKey] = []time.Time{now.Add(-3 * time.Minute), now.Add(-time.Minute)}
	limiter.attempts[newerKey] = []time.Time{now.Add(-2 * time.Minute)}

	limiter.recordFailedAttempt("192.0.2.1:1234", "new")

	if _, ok := limiter.attempts[newerKey]; ok {
		t.Fatal("key with the oldest last failure was kept")
	}
	if _, ok := limiter.attempts[olderKey]; !ok {
		t.Fatal("key with a more recent failure was evicted")
	}
}
