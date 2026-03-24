package provider_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestTokenManagerValidateKey(t *testing.T) {
	tm := provider.NewTokenManager(nil, nil, provider.DefaultRegistry())

	tests := []struct {
		providerID string
		key        string
		wantErr    bool
	}{
		{"anthropic", "sk-ant-api03-abcdefghijklmnopqrstuvwxyz012345", false},
		{"anthropic", "", true},
		{"anthropic", "invalid-no-prefix", true},
		{"anthropic", "sk-ant-short", true}, // too short
		{"openai", "sk-proj-abcdefghijklmnop", false},
		{"openai", "sk-short", true}, // too short
		{"openai", "", true},
		{"ollama", "", false},
		{"gemini", "AIzaSyA-abcdefghijklmnopqrstuvwx", false},
		{"gemini", "short", true},                  // too short
		{"gemini", "AIzaSyA-abc def invalid!", true}, // invalid chars
		{"copilot", "ghu_abcdefghij", false},
		{"copilot", "short", true},
	}

	for _, tt := range tests {
		err := tm.ValidateKeyFormat(tt.providerID, tt.key)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateKeyFormat(%q, %q) error = %v, wantErr %v",
				tt.providerID, tt.key, err, tt.wantErr)
		}
	}
}

func TestTokenManagerSaveTokenUnknownProvider(t *testing.T) {
	tm := provider.NewTokenManager(nil, nil, provider.DefaultRegistry())
	err := tm.SaveToken(1, "nonexistent_provider", "sk-ant-somekey")
	if err == nil {
		t.Error("phải trả về error khi provider không tồn tại trong registry")
	}
}
