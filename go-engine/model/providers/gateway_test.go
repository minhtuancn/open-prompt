package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayImplementsProvider(t *testing.T) {
	var _ Provider = (*GatewayProvider)(nil)
}

func TestGatewayStreamComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path=%q, want /chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	gw := NewGatewayProvider("test-gw", "Test Gateway", srv.URL, "", "llama3", nil)
	var chunks []string
	err := gw.StreamComplete(context.Background(), CompletionRequest{
		Model: "llama3", Prompt: "hello",
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(chunks) != 2 || chunks[0] != "hello" || chunks[1] != " world" {
		t.Errorf("chunks=%v, want [hello, ' world']", chunks)
	}
}

func TestGatewayWithAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	gw := NewGatewayProvider("or", "OpenRouter", srv.URL, "sk-or-key", "gpt-4o", nil)
	_ = gw.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})

	if gotAuth != "Bearer sk-or-key" {
		t.Errorf("auth=%q, want 'Bearer sk-or-key'", gotAuth)
	}
}

func TestGatewayNoAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	gw := NewGatewayProvider("ollama", "Ollama", srv.URL, "", "llama3", nil)
	_ = gw.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})

	if gotAuth != "" {
		t.Errorf("auth=%q, want empty", gotAuth)
	}
}

func TestGatewayPresets(t *testing.T) {
	presets := GatewayPresets()
	if len(presets) < 4 {
		t.Fatalf("got %d presets, want >= 4", len(presets))
	}
	found := false
	for _, p := range presets {
		if p.Name == "ollama" {
			found = true
			if p.BaseURL != "http://localhost:11434/v1" {
				t.Errorf("ollama baseURL=%q", p.BaseURL)
			}
		}
	}
	if !found {
		t.Error("ollama preset not found")
	}
}

func TestGatewayValidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"llama3"}]}`)
	}))
	defer srv.Close()

	gw := NewGatewayProvider("test", "Test", srv.URL, "", "llama3", nil)
	if err := gw.Validate(context.Background()); err != nil {
		t.Fatalf("validate error: %v", err)
	}
}
