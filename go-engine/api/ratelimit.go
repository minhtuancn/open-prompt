package api

import (
	"sync"
	"time"
)

// RateLimiter giới hạn số lượng requests theo method.
// Tự động cleanup expired entries mỗi 5 phút.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limits  map[string]limitConfig
}

type limitConfig struct {
	maxRequests int
	window      time.Duration
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		limits: map[string]limitConfig{
			"auth.login":          {maxRequests: 5, window: time.Minute},
			"auth.register":       {maxRequests: 3, window: time.Minute},
			"marketplace.publish": {maxRequests: 5, window: time.Minute},
			"marketplace.search":  {maxRequests: 30, window: time.Minute},
		},
	}
	// Cleanup expired buckets mỗi 5 phút để tránh memory leak
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

// Allow kiểm tra xem request có được phép không
func (rl *RateLimiter) Allow(method, caller string) bool {
	limit, exists := rl.limits[method]
	if !exists {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := method + ":" + caller
	b, exists := rl.buckets[key]
	now := time.Now()

	if !exists || now.After(b.resetAt) {
		rl.buckets[key] = &bucket{count: 1, resetAt: now.Add(limit.window)}
		return true
	}

	if b.count >= limit.maxRequests {
		return false
	}
	b.count++
	return true
}

// cleanup xoá expired entries khỏi buckets map
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for key, b := range rl.buckets {
		if now.After(b.resetAt) {
			delete(rl.buckets, key)
		}
	}
}
