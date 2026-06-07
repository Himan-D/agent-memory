package main

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(3, time.Second)

	if allowed, _, _ := rl.allow("client1"); !allowed {
		t.Error("expected first request to be allowed")
	}
	if allowed, _, _ := rl.allow("client1"); !allowed {
		t.Error("expected second request to be allowed")
	}
	if allowed, _, _ := rl.allow("client1"); !allowed {
		t.Error("expected third request to be allowed")
	}

	if allowed, _, _ := rl.allow("client1"); allowed {
		t.Error("expected fourth request to be blocked")
	}

	if allowed, _, _ := rl.allow("client2"); !allowed {
		t.Error("expected different client to be allowed")
	}

	time.Sleep(time.Second + time.Millisecond)

	if allowed, _, _ := rl.allow("client1"); !allowed {
		t.Error("expected request to be allowed after window")
	}
}

func TestRateLimiterMultipleClients(t *testing.T) {
	rl := newRateLimiter(2, time.Second)

	rl.allow("client1")
	rl.allow("client1")
	if allowed, _, _ := rl.allow("client1"); allowed {
		t.Error("client1 should be rate limited")
	}

	if allowed, _, _ := rl.allow("client2"); !allowed {
		t.Error("client2 should be allowed")
	}
	if allowed, _, _ := rl.allow("client2"); !allowed {
		t.Error("client2 should be allowed")
	}
	if allowed, _, _ := rl.allow("client2"); allowed {
		t.Error("client2 should be rate limited")
	}
}

func TestRateLimiterHeaders(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)

	allowed, limit, remaining := rl.allow("client1")
	if !allowed {
		t.Error("first request should be allowed")
	}
	if limit != 3 {
		t.Errorf("expected limit 3, got %d", limit)
	}
	if remaining != 2 {
		t.Errorf("expected remaining 2, got %d", remaining)
	}

	allowed, limit, remaining = rl.allow("client1")
	if remaining != 1 {
		t.Errorf("expected remaining 1, got %d", remaining)
	}

	allowed, limit, remaining = rl.allow("client1")
	if remaining != 0 {
		t.Errorf("expected remaining 0, got %d", remaining)
	}

	allowed, _, remaining = rl.allow("client1")
	if allowed {
		t.Error("should be blocked at limit")
	}
	if remaining != 0 {
		t.Errorf("expected remaining 0 when blocked, got %d", remaining)
	}
}
