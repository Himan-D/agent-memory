package main

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(3, time.Second)

	// Should allow first 3 requests
	if !rl.allow("client1") {
		t.Error("expected first request to be allowed")
	}
	if !rl.allow("client1") {
		t.Error("expected second request to be allowed")
	}
	if !rl.allow("client1") {
		t.Error("expected third request to be allowed")
	}

	// 4th request should be blocked
	if rl.allow("client1") {
		t.Error("expected fourth request to be blocked")
	}

	// Different client should be allowed
	if !rl.allow("client2") {
		t.Error("expected different client to be allowed")
	}

	// Wait for window to pass
	time.Sleep(time.Second + time.Millisecond)

	// Should be allowed again after window
	if !rl.allow("client1") {
		t.Error("expected request to be allowed after window")
	}
}

func TestRateLimiterMultipleClients(t *testing.T) {
	rl := newRateLimiter(2, time.Second)

	// Client 1 gets limited
	rl.allow("client1")
	rl.allow("client1")
	if rl.allow("client1") {
		t.Error("client1 should be rate limited")
	}

	// Client 2 should be independent
	if !rl.allow("client2") {
		t.Error("client2 should be allowed")
	}
	if !rl.allow("client2") {
		t.Error("client2 should be allowed")
	}
	if rl.allow("client2") {
		t.Error("client2 should be rate limited")
	}
}
