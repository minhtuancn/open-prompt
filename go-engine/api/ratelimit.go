package api

import (
	"sync"
	"time"
)

// RateLimiter giới hạn số lượng requests theo method
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
	count    int
	resetAt  time.Time
}

// NewRateLimiter tạo rate limiter với per-method limits
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		limits: map[string]limitConfig{
			"auth.login":          {maxRequests: 5, window: time.Minute},       // 5 login/phút
			"auth.register":       {maxRequests: 3, window: time.Minute},       // 3 register/phút
			"marketplace.publish": {maxRequests: 5, window: time.Minute},       // 5 publish/phút
			"marketplace.search":  {maxRequests: 30, window: time.Minute},      // 30 search/phút
		},
	}
}

// Allow kiểm tra xem request có được phép không
// key = method (vd: "auth.login"), caller = identifier (vd: remote addr)
func (rl *RateLimiter) Allow(method, caller string) bool {
	limit, exists := rl.limits[method]
	if !exists {
		return true // Không có limit cho method này
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
