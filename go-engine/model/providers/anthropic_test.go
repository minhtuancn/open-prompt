package providers_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// TestAnthropicProvider chỉ chạy khi có API key thật
func TestAnthropicProvider(t *testing.T) {
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
		t.Fatalf("stream complete: %v", err)
	}
	if sb.Len() == 0 {
		t.Error("expected non-empty response")
	}
	t.Logf("response: %q", sb.String())
}

func TestAnthropicProviderBadKey(t *testing.T) {
	p := providers.NewAnthropicProvider("sk-bad-key")
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
		Prompt: "hello",
	}, func(chunk string) {})

	if err == nil {
		t.Error("expected error for bad API key")
	}
}
