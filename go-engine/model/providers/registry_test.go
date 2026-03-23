package providers

import (
	"context"
	"testing"
)

type mockProvider struct {
	name        string
	displayName string
	aliases     []string
	authType    AuthType
}

func (m *mockProvider) Name() string                                  { return m.name }
func (m *mockProvider) DisplayName() string                           { return m.displayName }
func (m *mockProvider) Aliases() []string                             { return m.aliases }
func (m *mockProvider) GetAuthType() AuthType                         { return m.authType }
func (m *mockProvider) Validate(_ context.Context) error              { return nil }
func (m *mockProvider) Models(_ context.Context) ([]string, error)    { return nil, nil }
func (m *mockProvider) StreamComplete(_ context.Context, _ CompletionRequest, onChunk func(string)) error {
	onChunk("mock")
	return nil
}

func TestRegistryRouteByName(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "sonnet", "anthropic"}})

	got, err := r.Route("anthropic")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteByAlias(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "sonnet", "anthropic"}})

	got, err := r.Route("claude")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteCaseInsensitive(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude"}})

	got, err := r.Route("CLAUDE")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteStripAt(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude"}})

	got, err := r.Route("@claude")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Route("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryDefault(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic"})
	r.Register(&mockProvider{name: "openai"})

	got, err := r.Default()
	if err != nil {
		t.Fatalf("Default error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryDefaultEmpty(t *testing.T) {
	r := NewRegistry()
	_, err := r.Default()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "a"})
	r.Register(&mockProvider{name: "b"})

	if got := len(r.All()); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
}

func TestRegistryFallbackCandidates(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "anthropic"}})
	r.Register(&mockProvider{name: "openai", aliases: []string{"gpt4", "openai"}})
	r.Register(&mockProvider{name: "ollama", aliases: []string{"local", "ollama"}})

	candidates := r.FallbackCandidates("anthropic")
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}
	for _, c := range candidates {
		if c.Name() == "anthropic" {
			t.Error("should not include failed provider")
		}
	}
}

func TestRegistryFallbackByAlias(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "anthropic"}})
	r.Register(&mockProvider{name: "openai"})

	candidates := r.FallbackCandidates("claude")
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].Name() != "openai" {
		t.Errorf("got %q, want 'openai'", candidates[0].Name())
	}
}
