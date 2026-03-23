package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func TestOpenAIProviderStreamComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("thiếu Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type phải là application/json")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" World\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	p := providers.NewOpenAIProvider("sk-test-key", server.URL)
	var chunks []string
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "gpt-4o",
		Prompt: "say hello",
	}, func(s string) {
		chunks = append(chunks, s)
	})

	if err != nil {
		t.Fatalf("StreamComplete() error: %v", err)
	}
	if len(chunks) != 2 {
		t.Errorf("len(chunks) = %d, want 2", len(chunks))
	}
}

func TestOpenAIProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	p := providers.NewOpenAIProvider("sk-test-key", server.URL)
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "gpt-4o",
		Prompt: "hello",
	}, func(s string) {})

	if err == nil {
		t.Error("phải trả về error khi API trả về 429")
	}
}
