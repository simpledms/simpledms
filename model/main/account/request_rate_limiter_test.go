package account

import (
	"testing"
	"time"
)

func TestRequestRateLimiterAllow(t *testing.T) {
	limiter := NewRequestRateLimiter()

	if !limiter.Allow("ip:198.51.100.1", time.Minute, 2) {
		t.Fatal("expected first attempt to be allowed")
	}
	if !limiter.Allow("ip:198.51.100.1", time.Minute, 2) {
		t.Fatal("expected second attempt to be allowed")
	}
	if limiter.Allow("ip:198.51.100.1", time.Minute, 2) {
		t.Fatal("expected third attempt to be rate limited")
	}
}

func TestRequestRateLimiterResetsAfterWindow(t *testing.T) {
	limiter := NewRequestRateLimiter()

	if !limiter.Allow("email:user@example.com", 20*time.Millisecond, 1) {
		t.Fatal("expected first attempt to be allowed")
	}
	if limiter.Allow("email:user@example.com", 20*time.Millisecond, 1) {
		t.Fatal("expected second attempt in window to be blocked")
	}

	time.Sleep(30 * time.Millisecond)

	if !limiter.Allow("email:user@example.com", 20*time.Millisecond, 1) {
		t.Fatal("expected attempt after window to be allowed")
	}
}
