package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiImplementsProvider(t *testing.T) {
	var _ Provider = (*GeminiProvider)(nil)
}

func TestGeminiStreamComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("missing API key in query")
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Errorf("missing alt=sse")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]}}]}\n\n")
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" World\"}]}}]}\n\n")
	}))
	defer srv.Close()

	gm := NewGeminiProviderWithBaseURL("test-key", srv.URL)
	var chunks []string
	err := gm.StreamComplete(context.Background(), CompletionRequest{
		Model: "gemini-1.5-flash", Prompt: "hello",
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(chunks) != 2 || chunks[0] != "Hello" || chunks[1] != " World" {
		t.Errorf("chunks=%v, want [Hello, ' World']", chunks)
	}
}

func TestGeminiError401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"message":"invalid key"}}`)
	}))
	defer srv.Close()

	gm := NewGeminiProviderWithBaseURL("bad-key", srv.URL)
	err := gm.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
