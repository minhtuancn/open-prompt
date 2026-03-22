package model

import (
	"context"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// Router route request đến đúng provider
type Router struct {
	providers map[string]*providers.AnthropicProvider
}

// NewRouter tạo router mới
func NewRouter() *Router {
	return &Router{
		providers: make(map[string]*providers.AnthropicProvider),
	}
}

// RegisterAnthropic đăng ký Anthropic provider
func (r *Router) RegisterAnthropic(apiKey string) {
	r.providers["anthropic"] = providers.NewAnthropicProvider(apiKey)
}

// Stream gửi request và stream response qua callback
func (r *Router) Stream(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	p, ok := r.providers["anthropic"]
	if !ok {
		return fmt.Errorf("no provider configured")
	}

	start := time.Now()
	err := p.StreamComplete(ctx, providers.CompletionRequest{
		Model:       req.Model,
		Prompt:      req.Prompt,
		System:      req.System,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}, onChunk)

	_ = time.Since(start) // latency logging sẽ thêm ở Phase 2
	return err
}
