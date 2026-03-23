package model

import (
	"context"
	"fmt"
	"strings"
)

// StreamRequest là request chung cho tất cả providers trong fallback chain
type StreamRequest struct {
	Model       string
	Prompt      string
	System      string
	Temperature float64
	MaxTokens   int
}

// StreamProvider là interface chung cho tất cả AI providers
type StreamProvider interface {
	StreamComplete(ctx context.Context, req StreamRequest, onChunk func(string)) error
}

// NamedProvider kết hợp tên và provider
type NamedProvider struct {
	Name     string
	Provider StreamProvider
}

// FallbackChain thực hiện fallback tuần tự qua danh sách providers
type FallbackChain struct {
	providers []NamedProvider
}

// NewFallbackChain tạo fallback chain mới
func NewFallbackChain(providers []NamedProvider) *FallbackChain {
	return &FallbackChain{providers: providers}
}

// StreamComplete thực hiện streaming, fallback qua chain nếu cần
func (c *FallbackChain) StreamComplete(ctx context.Context, req StreamRequest, onChunk func(string)) error {
	if len(c.providers) == 0 {
		return fmt.Errorf("fallback chain rỗng — không có provider nào")
	}
	var lastErr error
	for _, np := range c.providers {
		err := np.Provider.StreamComplete(ctx, req, onChunk)
		if err == nil {
			return nil
		}
		if IsFallbackError(err) {
			lastErr = fmt.Errorf("provider %q thất bại: %w", np.Name, err)
			continue
		}
		return fmt.Errorf("provider %q lỗi không thể fallback: %w", np.Name, err)
	}
	return fmt.Errorf("tất cả providers đều thất bại, lỗi cuối: %w", lastErr)
}

// IsFallbackError kiểm tra xem lỗi có nên trigger fallback không
func IsFallbackError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "503") || strings.Contains(msg, "502") ||
		strings.Contains(msg, "504") || strings.Contains(msg, "500") ||
		strings.Contains(msg, "service unavailable") || strings.Contains(msg, "bad gateway") {
		return true
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context canceled") {
		return true
	}
	return false
}
