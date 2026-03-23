package model

import (
	"context"
	"fmt"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// testProvider implements providers.Provider cho test
type testProvider struct {
	name   string
	err    error
	called bool
}

func (t *testProvider) Name() string                              { return t.name }
func (t *testProvider) DisplayName() string                       { return t.name }
func (t *testProvider) Aliases() []string                         { return nil }
func (t *testProvider) GetAuthType() providers.AuthType           { return providers.AuthNone }
func (t *testProvider) Validate(_ context.Context) error          { return nil }
func (t *testProvider) Models(_ context.Context) ([]string, error) { return nil, nil }
func (t *testProvider) StreamComplete(_ context.Context, _ providers.CompletionRequest, onChunk func(string)) error {
	t.called = true
	if t.err != nil {
		return t.err
	}
	onChunk("hello from " + t.name)
	return nil
}

func TestFallbackChainSuccess(t *testing.T) {
	p1 := &testProvider{name: "p1", err: fmt.Errorf("HTTP 429 rate limit")}
	p2 := &testProvider{name: "p2"}

	chain := NewFallbackChain([]providers.Provider{p1, p2})
	var chunks []string
	err := chain.StreamComplete(context.Background(), providers.CompletionRequest{}, func(s string) {
		chunks = append(chunks, s)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p1.called || !p2.called {
		t.Error("both providers should be called")
	}
	if len(chunks) == 0 || chunks[0] != "hello from p2" {
		t.Errorf("got chunks=%v, want ['hello from p2']", chunks)
	}
}

func TestFallbackChainAllFail(t *testing.T) {
	p1 := &testProvider{name: "p1", err: fmt.Errorf("HTTP 503")}
	p2 := &testProvider{name: "p2", err: fmt.Errorf("HTTP 502")}

	chain := NewFallbackChain([]providers.Provider{p1, p2})
	err := chain.StreamComplete(context.Background(), providers.CompletionRequest{}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsFallbackError(t *testing.T) {
	tests := []struct {
		err  string
		want bool
	}{
		{"HTTP 429 rate limit", true},
		{"HTTP 503", true},
		{"timeout", true},
		{"context deadline exceeded", true},
		{"invalid api key", false},
		{"bad request", false},
	}
	for _, tt := range tests {
		got := IsFallbackError(fmt.Errorf("%s", tt.err))
		if got != tt.want {
			t.Errorf("IsFallbackError(%q)=%v, want %v", tt.err, got, tt.want)
		}
	}
}
