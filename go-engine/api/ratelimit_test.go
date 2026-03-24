package api

import (
	"testing"
	"time"
)

func TestRateLimiter_StopTerminatesGoroutine(t *testing.T) {
	rl := NewRateLimiter()
	time.Sleep(10 * time.Millisecond)
	rl.Stop()
	// Calling Stop again should not panic (idempotent)
	rl.Stop()
}

func TestRateLimiter_AllowAfterStop(t *testing.T) {
	rl := NewRateLimiter()
	rl.Stop()
	allowed := rl.Allow("test.method", "caller1")
	if !allowed {
		t.Fatal("expected Allow to return true for first call")
	}
}
