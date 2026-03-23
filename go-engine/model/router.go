package model

import (
	"context"
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// Router route request đến đúng provider qua Registry
type Router struct {
	registry *providers.Registry
}

// NewRouter tạo router mới
func NewRouter(registry *providers.Registry) *Router {
	return &Router{registry: registry}
}

// Stream gửi request đến provider theo alias
// alias="" → dùng Default provider
func (r *Router) Stream(ctx context.Context, alias string, req providers.CompletionRequest, onChunk func(string)) error {
	var p providers.Provider
	var err error

	if alias != "" {
		p, err = r.registry.Route(alias)
	} else {
		p, err = r.registry.Default()
	}
	if err != nil {
		return fmt.Errorf("route provider: %w", err)
	}

	return p.StreamComplete(ctx, req, onChunk)
}

// Registry trả về underlying registry
func (r *Router) Registry() *providers.Registry {
	return r.registry
}
