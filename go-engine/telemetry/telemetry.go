package telemetry

import (
	"log"
	"sync"
)

// Client quản lý telemetry opt-in
// Chỉ track events cơ bản (app_start, query_count, provider_used)
// KHÔNG gửi nội dung query hay response
type Client struct {
	mu      sync.RWMutex
	enabled bool
	events  []Event
}

// Event là một telemetry event
type Event struct {
	Name  string            `json:"name"`
	Props map[string]string `json:"props,omitempty"`
}

// New tạo client mới
func New(enabled bool) *Client {
	return &Client{enabled: enabled}
}

// SetEnabled bật/tắt telemetry
func (c *Client) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// IsEnabled kiểm tra telemetry có bật không
func (c *Client) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

const maxEvents = 1000

// Track ghi nhận event (chỉ khi enabled, tối đa 1000 events trong buffer)
func (c *Client) Track(name string, props map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.enabled {
		return
	}
	if len(c.events) >= maxEvents {
		// Xoá 25% events cũ nhất khi đầy
		c.events = c.events[maxEvents/4:]
	}
	c.events = append(c.events, Event{Name: name, Props: props})
	log.Printf("[telemetry] %s %v", name, props)
}

// Flush lấy tất cả events và xoá buffer
// Dùng khi cần gửi đi (phase sau)
func (c *Client) Flush() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	events := c.events
	c.events = nil
	return events
}
