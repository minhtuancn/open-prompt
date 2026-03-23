package provider

import (
	"context"
	"sync"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// HealthStatus chứa trạng thái sức khỏe của provider
type HealthStatus struct {
	Name      string `json:"name"`
	Healthy   bool   `json:"healthy"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
	CheckedAt string `json:"checked_at"`
}

// HealthChecker kiểm tra sức khỏe providers định kỳ
type HealthChecker struct {
	registry *providers.Registry
	mu       sync.RWMutex
	statuses map[string]HealthStatus
	interval time.Duration
	stopCh   chan struct{}
}

// NewHealthChecker tạo health checker mới
func NewHealthChecker(registry *providers.Registry, interval time.Duration) *HealthChecker {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &HealthChecker{
		registry: registry,
		statuses: make(map[string]HealthStatus),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start bắt đầu kiểm tra định kỳ
func (h *HealthChecker) Start() {
	// Chạy lần đầu ngay
	h.CheckAll()
	go func() {
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				h.CheckAll()
			case <-h.stopCh:
				return
			}
		}
	}()
}

// Stop dừng health checker
func (h *HealthChecker) Stop() {
	close(h.stopCh)
}

// CheckAll kiểm tra tất cả providers
func (h *HealthChecker) CheckAll() {
	allProviders := h.registry.All()
	var wg sync.WaitGroup
	results := make([]HealthStatus, len(allProviders))

	for i, p := range allProviders {
		wg.Add(1)
		go func(idx int, prov providers.Provider) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			start := time.Now()
			err := prov.Validate(ctx)
			latency := time.Since(start).Milliseconds()

			status := HealthStatus{
				Name:      prov.Name(),
				Healthy:   err == nil,
				LatencyMs: latency,
				CheckedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
			}
			if err != nil {
				status.Error = err.Error()
			}
			results[idx] = status
		}(i, p)
	}
	wg.Wait()

	h.mu.Lock()
	for _, s := range results {
		h.statuses[s.Name] = s
	}
	h.mu.Unlock()
}

// GetAll trả về trạng thái tất cả providers
func (h *HealthChecker) GetAll() []HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]HealthStatus, 0, len(h.statuses))
	for _, s := range h.statuses {
		result = append(result, s)
	}
	return result
}

// Get trả về trạng thái của 1 provider
func (h *HealthChecker) Get(name string) (HealthStatus, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.statuses[name]
	return s, ok
}
