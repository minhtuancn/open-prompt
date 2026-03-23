package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func TestOllamaProviderStreamComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"message\":{\"content\":\"Hello\"},\"done\":false}\n"))
		w.Write([]byte("{\"message\":{\"content\":\" World\"},\"done\":false}\n"))
		w.Write([]byte("{\"message\":{\"content\":\"\"},\"done\":true}\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	p := providers.NewOllamaProvider(server.URL)
	var chunks []string
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "llama3.2",
		Prompt: "hello",
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

func TestOllamaProviderConnectionRefused(t *testing.T) {
	p := providers.NewOllamaProvider("http://localhost:19999")
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "llama3.2",
		Prompt: "hello",
	}, func(s string) {})

	if err == nil {
		t.Error("phải trả về error khi Ollama không chạy")
	}
}
