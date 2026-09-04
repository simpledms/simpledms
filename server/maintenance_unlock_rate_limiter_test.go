package server

import (
	"fmt"
	"net/http/httptest"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMaintenanceUnlockRateLimiterWindowAndClientIsolation(t *testing.T) {
	limiter := newMaintenanceUnlockRateLimiter(nil)
	now := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	firstClient := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
	firstClient.RemoteAddr = "192.0.2.10:1234"

	for attempt := range maintenanceUnlockAttemptLimit {
		retryAfter, isAllowed := limiter.allow(firstClient, now)
		if !isAllowed {
			t.Fatalf("attempt %d: expected request to be allowed, retry after %d", attempt, retryAfter)
		}
	}
	if retryAfter, isAllowed := limiter.allow(firstClient, now); isAllowed || retryAfter != 60 {
		t.Fatalf("expected request blocked for 60 seconds, got allowed=%v retry=%d", isAllowed, retryAfter)
	}

	secondClient := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
	secondClient.RemoteAddr = "192.0.2.11:1234"
	if retryAfter, isAllowed := limiter.allow(secondClient, now); !isAllowed {
		t.Fatalf("expected separate client to be allowed, retry after %d", retryAfter)
	}

	if retryAfter, isAllowed := limiter.allow(firstClient, now.Add(30*time.Second)); isAllowed || retryAfter != 30 {
		t.Fatalf("expected request blocked for 30 seconds, got allowed=%v retry=%d", isAllowed, retryAfter)
	}
	if retryAfter, isAllowed := limiter.allow(firstClient, now.Add(time.Minute)); !isAllowed {
		t.Fatalf("expected request after window to be allowed, retry after %d", retryAfter)
	}
}

func TestMaintenanceUnlockRateLimiterUsesOnlyTrustedForwarding(t *testing.T) {
	trustedProxy := netip.MustParsePrefix("10.0.0.0/8")
	limiter := newMaintenanceUnlockRateLimiter([]netip.Prefix{trustedProxy})

	untrustedReq := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
	untrustedReq.RemoteAddr = "192.0.2.10:1234"
	untrustedReq.Header.Set("X-Forwarded-For", "198.51.100.20")
	if clientKey := limiter.clientKey(untrustedReq); clientKey != "192.0.2.10" {
		t.Fatalf("expected direct untrusted client key, got %q", clientKey)
	}

	trustedReq := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
	trustedReq.RemoteAddr = "10.0.0.10:1234"
	trustedReq.Header.Set("X-Forwarded-For", "198.51.100.20, 10.0.0.20")
	if clientKey := limiter.clientKey(trustedReq); clientKey != "198.51.100.20" {
		t.Fatalf("expected forwarded client key, got %q", clientKey)
	}

	trustedReq.Header.Set("X-Forwarded-For", "invalid")
	if clientKey := limiter.clientKey(trustedReq); clientKey != "10.0.0.10" {
		t.Fatalf("expected direct proxy key for malformed chain, got %q", clientKey)
	}
}

func TestMaintenanceUnlockRateLimiterBoundsClients(t *testing.T) {
	limiter := newMaintenanceUnlockRateLimiter(nil)
	now := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	limitedReq := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
	limitedReq.RemoteAddr = "192.0.2.10:1234"
	for range maintenanceUnlockAttemptLimit {
		if retryAfter, isAllowed := limiter.allow(limitedReq, now); !isAllowed {
			t.Fatalf("expected initial client attempt allowed, retry after %d", retryAfter)
		}
	}

	for index := range maintenanceUnlockClientLimit - 2 {
		req := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
		req.RemoteAddr = fmt.Sprintf("[2001:db8::%x]:1234", index+1)
		if retryAfter, isAllowed := limiter.allow(req, now); !isAllowed {
			t.Fatalf("client %d: expected request to be allowed, retry after %d", index, retryAfter)
		}
	}
	for index := range maintenanceUnlockAttemptLimit {
		req := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
		req.RemoteAddr = fmt.Sprintf("[2001:db9::%x]:1234", index+1)
		if retryAfter, isAllowed := limiter.allow(req, now); !isAllowed {
			t.Fatalf("overflow attempt %d: expected allowed, retry after %d", index, retryAfter)
		}
	}
	overflowReq := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
	overflowReq.RemoteAddr = "[2001:db9::100]:1234"
	if retryAfter, isAllowed := limiter.allow(overflowReq, now); isAllowed || retryAfter != 60 {
		t.Fatalf("expected overflow bucket blocked, got allowed=%v retry=%d", isAllowed, retryAfter)
	}
	if len(limiter.attempts) != maintenanceUnlockClientLimit {
		t.Fatalf("expected at most %d clients, got %d", maintenanceUnlockClientLimit, len(limiter.attempts))
	}
	if retryAfter, isAllowed := limiter.allow(limitedReq, now); isAllowed || retryAfter != 60 {
		t.Fatalf("expected saturated table to preserve active limit, got allowed=%v retry=%d", isAllowed, retryAfter)
	}
}

func TestMaintenanceUnlockRateLimiterLimitsConcurrentClientAttempts(t *testing.T) {
	limiter := newMaintenanceUnlockRateLimiter(nil)
	now := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
	start := make(chan struct{})
	var allowed atomic.Int32
	var requests sync.WaitGroup

	for range 20 {
		requests.Go(func() {
			<-start
			req := httptest.NewRequest("POST", "/-/unlock-cmd", nil)
			req.RemoteAddr = "192.0.2.10:1234"
			if _, isAllowed := limiter.allow(req, now); isAllowed {
				allowed.Add(1)
			}
		})
	}
	close(start)
	requests.Wait()

	if allowed.Load() != maintenanceUnlockAttemptLimit {
		t.Fatalf("expected %d allowed attempts, got %d", maintenanceUnlockAttemptLimit, allowed.Load())
	}
}
