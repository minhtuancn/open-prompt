package providers

import (
	"fmt"
	"strings"
	"sync"
)

// Registry quản lý tất cả providers đã đăng ký
type Registry struct {
	mu        sync.RWMutex
	providers []Provider          // giữ thứ tự đăng ký
	byName    map[string]Provider // name → provider
	aliases   map[string]string   // alias → name
}

// NewRegistry tạo registry rỗng
func NewRegistry() *Registry {
	return &Registry{
		byName:  make(map[string]Provider),
		aliases: make(map[string]string),
	}
}

// Register thêm provider và map tất cả alias
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(p.Name())
	r.providers = append(r.providers, p)
	r.byName[name] = p
	r.aliases[name] = name
	for _, alias := range p.Aliases() {
		r.aliases[strings.ToLower(alias)] = name
	}
}

// Route tìm provider theo alias (case-insensitive, strip "@")
func (r *Registry) Route(alias string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alias = strings.ToLower(strings.TrimPrefix(alias, "@"))
	name, ok := r.aliases[alias]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", alias)
	}
	return r.byName[name], nil
}

// Default trả về provider đầu tiên
func (r *Registry) Default() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.providers) == 0 {
		return nil, fmt.Errorf("no providers registered")
	}
	return r.providers[0], nil
}

// All trả về tất cả providers theo thứ tự đăng ký
func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

// FallbackCandidates trả về providers thay thế (bỏ qua failed)
// failedName có thể là name hoặc alias — resolve trước khi filter
func (r *Registry) FallbackCandidates(failedName string) []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resolvedName := strings.ToLower(failedName)
	if name, ok := r.aliases[resolvedName]; ok {
		resolvedName = name
	}

	var candidates []Provider
	for _, p := range r.providers {
		if strings.ToLower(p.Name()) != resolvedName {
			candidates = append(candidates, p)
		}
	}
	return candidates
}
