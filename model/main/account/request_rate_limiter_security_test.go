package account

import (
	"fmt"
	"testing"
	"time"
)

func TestRequestRateLimiterBoundsKeysWithoutEvictingBlockedEntries(t *testing.T) {
	limiter := NewRequestRateLimiter()
	window := time.Hour
	if !limiter.Allow("blocked", window, 1) || limiter.Allow("blocked", window, 1) {
		t.Fatal("failed to establish a live blocked key")
	}
	for i := 0; i < maxRequestRateLimitKeys-1; i++ {
		if !limiter.Allow(fmt.Sprintf("key-%d", i), window, 1) {
			t.Fatalf("key %d was unexpectedly rejected", i)
		}
	}
	if limiter.Allow("attacker-new-key", window, 1) {
		t.Fatal("accepted a new key beyond the key cap")
	}
	if limiter.Allow("blocked", window, 1) {
		t.Fatal("flooding evicted a still-blocked key")
	}
	if got := len(limiter.attempts); got != maxRequestRateLimitKeys {
		t.Fatalf("tracked %d keys, want cap %d", got, maxRequestRateLimitKeys)
	}
}

func TestRequestRateLimiterReclaimsExpiredKeys(t *testing.T) {
	limiter := NewRequestRateLimiter()
	if !limiter.Allow("expired", time.Nanosecond, 1) {
		t.Fatal("initial request was rejected")
	}
	for i := 0; i < maxRequestRateLimitKeys-2; i++ {
		if !limiter.Allow(fmt.Sprintf("live-%d", i), time.Hour, 1) {
			t.Fatalf("live key %d was rejected", i)
		}
	}
	if !limiter.Allow("blocked-live", time.Hour, 1) || limiter.Allow("blocked-live", time.Hour, 1) {
		t.Fatal("failed to establish blocked live key")
	}
	time.Sleep(time.Millisecond)
	if !limiter.Allow("replacement", time.Hour, 1) {
		t.Fatal("expired entry prevented a new key")
	}
	if _, exists := limiter.attempts["expired"]; exists {
		t.Fatal("expired key was not reclaimed")
	}
	if limiter.Allow("blocked-live", time.Hour, 1) {
		t.Fatal("reclaiming an expired key evicted a live blocked key")
	}
}
