package provider

import "sync"

// Model định nghĩa một AI model cụ thể
type Model struct {
	ID              string
	Name            string
	MaxTokens       int
	InputCostPer1K  float64 // USD per 1000 input tokens
	OutputCostPer1K float64 // USD per 1000 output tokens
	SupportsStream  bool
}

// Provider định nghĩa một AI provider
type Provider struct {
	ID           string
	Name         string
	AuthType     string // "api_key" | "oauth" | "auto_detected"
	BaseURL      string
	Models       []Model
	DefaultModel string
}

// Registry lưu trữ danh sách providers đã biết
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// DefaultRegistry trả về registry với providers mặc định
func DefaultRegistry() *Registry {
	r := &Registry{providers: make(map[string]Provider)}

	r.providers["anthropic"] = Provider{
		ID:           "anthropic",
		Name:         "Anthropic (Claude)",
		AuthType:     "api_key",
		BaseURL:      "https://api.anthropic.com",
		DefaultModel: "claude-3-5-sonnet-20241022",
		Models: []Model{
			{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", MaxTokens: 8192, InputCostPer1K: 0.003, OutputCostPer1K: 0.015, SupportsStream: true},
			{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", MaxTokens: 8192, InputCostPer1K: 0.001, OutputCostPer1K: 0.005, SupportsStream: true},
			{ID: "claude-opus-4-5", Name: "Claude Opus 4.5", MaxTokens: 32000, InputCostPer1K: 0.015, OutputCostPer1K: 0.075, SupportsStream: true},
		},
	}

	r.providers["openai"] = Provider{
		ID:           "openai",
		Name:         "OpenAI (ChatGPT)",
		AuthType:     "api_key",
		BaseURL:      "https://api.openai.com",
		DefaultModel: "gpt-4o",
		Models: []Model{
			{ID: "gpt-4o", Name: "GPT-4o", MaxTokens: 4096, InputCostPer1K: 0.005, OutputCostPer1K: 0.015, SupportsStream: true},
			{ID: "gpt-4o-mini", Name: "GPT-4o mini", MaxTokens: 4096, InputCostPer1K: 0.00015, OutputCostPer1K: 0.0006, SupportsStream: true},
		},
	}

	r.providers["gemini"] = Provider{
		ID:           "gemini",
		Name:         "Google Gemini",
		AuthType:     "oauth",
		BaseURL:      "https://generativelanguage.googleapis.com",
		DefaultModel: "gemini-1.5-pro",
		Models: []Model{
			{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", MaxTokens: 8192, InputCostPer1K: 0.00125, OutputCostPer1K: 0.005, SupportsStream: true},
			{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", MaxTokens: 8192, InputCostPer1K: 0.000075, OutputCostPer1K: 0.0003, SupportsStream: true},
		},
	}

	r.providers["ollama"] = Provider{
		ID:           "ollama",
		Name:         "Ollama (Local)",
		AuthType:     "auto_detected",
		BaseURL:      "http://localhost:11434",
		DefaultModel: "llama3",
		Models: []Model{
			{ID: "llama3", Name: "Llama 3", MaxTokens: 4096, InputCostPer1K: 0, OutputCostPer1K: 0, SupportsStream: true},
			{ID: "mistral", Name: "Mistral 7B", MaxTokens: 4096, InputCostPer1K: 0, OutputCostPer1K: 0, SupportsStream: true},
			{ID: "codellama", Name: "CodeLlama", MaxTokens: 4096, InputCostPer1K: 0, OutputCostPer1K: 0, SupportsStream: true},
		},
	}

	r.providers["openrouter"] = Provider{
		ID:           "openrouter",
		Name:         "OpenRouter",
		AuthType:     "api_key",
		BaseURL:      "https://openrouter.ai/api",
		DefaultModel: "openai/gpt-4o",
		Models: []Model{
			{ID: "openai/gpt-4o", Name: "GPT-4o (via OpenRouter)", MaxTokens: 4096, InputCostPer1K: 0.005, OutputCostPer1K: 0.015, SupportsStream: true},
			{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet (via OR)", MaxTokens: 8192, InputCostPer1K: 0.003, OutputCostPer1K: 0.015, SupportsStream: true},
		},
	}

	return r
}

// Get trả về provider theo ID
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// List trả về tất cả providers
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

// GetModel trả về model trong một provider
func (r *Registry) GetModel(providerID, modelID string) (Model, bool) {
	p, ok := r.Get(providerID)
	if !ok {
		return Model{}, false
	}
	for _, m := range p.Models {
		if m.ID == modelID {
			return m, true
		}
	}
	return Model{}, false
}
