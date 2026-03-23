package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// FallbackChain thực hiện fallback tuần tự qua danh sách providers
type FallbackChain struct {
	providers []providers.Provider
}

// NewFallbackChain tạo fallback chain mới
func NewFallbackChain(providerList []providers.Provider) *FallbackChain {
	return &FallbackChain{providers: providerList}
}

// StreamComplete thực hiện streaming, fallback qua chain nếu cần
func (c *FallbackChain) StreamComplete(ctx context.Context, req providers.CompletionRequest, onChunk func(string)) error {
	if len(c.providers) == 0 {
		return fmt.Errorf("fallback chain rỗng — không có provider nào")
	}
	var lastErr error
	for _, p := range c.providers {
		err := p.StreamComplete(ctx, req, onChunk)
		if err == nil {
			return nil
		}
		if IsFallbackError(err) {
			lastErr = fmt.Errorf("provider %q thất bại: %w", p.Name(), err)
			continue
		}
		return fmt.Errorf("provider %q lỗi không thể fallback: %w", p.Name(), err)
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
