package model_test

import (
	"context"
	"errors"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model"
)

type mockProvider struct {
	name    string
	callErr error
	called  bool
}

func (m *mockProvider) StreamComplete(ctx context.Context, req model.StreamRequest, onChunk func(string)) error {
	m.called = true
	if m.callErr != nil {
		return m.callErr
	}
	onChunk("response from " + m.name)
	return nil
}

func TestFallbackChainSuccess(t *testing.T) {
	p1 := &mockProvider{name: "p1", callErr: errors.New("rate limit 429")}
	p2 := &mockProvider{name: "p2"}

	chain := model.NewFallbackChain([]model.NamedProvider{
		{Name: "p1", Provider: p1},
		{Name: "p2", Provider: p2},
	})

	var got string
	err := chain.StreamComplete(context.Background(), model.StreamRequest{Prompt: "hello"}, func(s string) {
		got = s
	})

	if err != nil {
		t.Fatalf("FallbackChain.StreamComplete() error = %v", err)
	}
	if !p1.called {
		t.Error("p1 phải được gọi trước")
	}
	if !p2.called {
		t.Error("p2 phải được gọi khi p1 thất bại")
	}
	if got != "response from p2" {
		t.Errorf("got = %q, want %q", got, "response from p2")
	}
}

func TestFallbackChainAllFail(t *testing.T) {
	p1 := &mockProvider{name: "p1", callErr: errors.New("error")}
	p2 := &mockProvider{name: "p2", callErr: errors.New("error")}

	chain := model.NewFallbackChain([]model.NamedProvider{
		{Name: "p1", Provider: p1},
		{Name: "p2", Provider: p2},
	})

	err := chain.StreamComplete(context.Background(), model.StreamRequest{}, func(s string) {})
	if err == nil {
		t.Error("phải trả về error khi tất cả providers thất bại")
	}
}

func TestIsFallbackError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{errors.New("rate limit 429"), true},
		{errors.New("HTTP 503 service unavailable"), true},
		{errors.New("timeout exceeded"), true},
		{errors.New("context deadline exceeded"), true},
		{errors.New("invalid api key"), false},
		{errors.New("bad request"), false},
	}
	for _, tt := range tests {
		got := model.IsFallbackError(tt.err)
		if got != tt.want {
			t.Errorf("IsFallbackError(%q) = %v, want %v", tt.err.Error(), got, tt.want)
		}
	}
}
