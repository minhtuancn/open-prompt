package api_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/api"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func TestQueryStreamMentionSmoke(t *testing.T) {
	// Smoke test: @mention không gây crash khi registry rỗng
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "mentiontest", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "query.stream", map[string]interface{}{
		"token": token,
		"input": "@claude hello",
	})
	// Không có provider configured → error hoặc nil, nhưng không crash
	t.Logf("Response: error=%v", resp.Error)
}

func TestQueryStreamWithProviderParam(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "provuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "query.stream", map[string]interface{}{
		"token":    token,
		"input":    "hello",
		"provider": "anthropic",
	})
	t.Logf("Response: error=%v", resp.Error)
}

func TestParseMentionWithRegistryIntegration(t *testing.T) {
	// Verify ParseMention + Registry.Route integration
	reg := providers.NewRegistry()
	reg.Register(providers.NewAnthropicProviderWithBaseURL("test-key", "http://localhost"))
	reg.Register(providers.NewOpenAIProvider("test-key", "http://localhost"))

	tests := []struct {
		input     string
		wantProv  string
		wantClean string
	}{
		{"@claude viết email", "anthropic", "viết email"},
		{"@gpt4 hello world", "openai", "hello world"},
		{"hello world", "", "hello world"},
	}

	for _, tt := range tests {
		alias, clean := api.ParseMention(tt.input)
		if tt.wantProv == "" {
			if alias != "" {
				t.Errorf("input=%q: got alias=%q, want empty", tt.input, alias)
			}
			continue
		}

		prov, err := reg.Route(alias)
		if err != nil {
			t.Errorf("input=%q: Route(%q) error: %v", tt.input, alias, err)
			continue
		}
		if prov.Name() != tt.wantProv {
			t.Errorf("input=%q: got provider=%q, want %q", tt.input, prov.Name(), tt.wantProv)
		}
		if clean != tt.wantClean {
			t.Errorf("input=%q: got clean=%q, want %q", tt.input, clean, tt.wantClean)
		}
	}
}
