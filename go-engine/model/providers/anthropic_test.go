package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// TestAnthropicProviderMock kiểm tra SSE parsing không cần API key thật
func TestAnthropicProviderMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("thiếu x-api-key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("thiếu anthropic-version header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type phải là application/json")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\"}\n\n"))
		w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\" World\"}}\n\n"))
		w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	p := providers.NewAnthropicProviderWithBaseURL("sk-ant-test", server.URL)
	var chunks []string
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
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
	if strings.Join(chunks, "") != "Hello World" {
		t.Errorf("response = %q, want %q", strings.Join(chunks, ""), "Hello World")
	}
}

// TestAnthropicProvider401 kiểm tra xử lý lỗi 401 từ API
func TestAnthropicProvider401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
	}))
	defer server.Close()

	p := providers.NewAnthropicProviderWithBaseURL("sk-ant-invalid", server.URL)
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
		Prompt: "hello",
	}, func(s string) {})

	if err == nil {
		t.Error("phải trả về error khi API trả về 401")
	}
}

// TestAnthropicProvider429 kiểm tra xử lý rate limit
func TestAnthropicProvider429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	p := providers.NewAnthropicProviderWithBaseURL("sk-ant-test", server.URL)
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
		Prompt: "hello",
	}, func(s string) {})

	if err == nil {
		t.Error("phải trả về error khi API trả về 429")
	}
}

// TestAnthropicProviderLive chỉ chạy khi có API key thật và đủ credits
func TestAnthropicProviderLive(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	p := providers.NewAnthropicProvider(apiKey)
	var sb strings.Builder

	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
		Prompt: "Say hello in one word.",
	}, func(chunk string) {
		sb.WriteString(chunk)
	})

	if err != nil {
		t.Skipf("live test skipped: %v", err)
	}
	if sb.Len() == 0 {
		t.Error("expected non-empty response")
	}
	t.Logf("response: %q", sb.String())
}
