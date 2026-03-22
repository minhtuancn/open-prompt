package model

import (
	"context"
	"fmt"

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
func (r *Router) Stream(ctx context.Context, req providers.CompletionRequest, onChunk func(string)) error {
	p, ok := r.providers["anthropic"]
	if !ok {
		return fmt.Errorf("no provider configured")
	}
	return p.StreamComplete(ctx, req, onChunk)
}
