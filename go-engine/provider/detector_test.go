package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestDetector_DetectFromEnv(t *testing.T) {
	// Setup env
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	d := provider.NewDetector(provider.DetectorConfig{ScanFiles: false})
	results := d.Detect()

	found := false
	for _, r := range results {
		if r.ProviderID == "anthropic" && r.Source == "env" {
			found = true
			if r.Token == "" {
				t.Error("Token phải không rỗng khi detect từ env")
			}
		}
	}
	if !found {
		t.Error("Phải detect được anthropic từ ANTHROPIC_API_KEY")
	}
}

func TestDetector_DetectFromFile(t *testing.T) {
	// Tạo temp file giả lập ~/.claude/claude.json
	dir := t.TempDir()
	claudeJSON := filepath.Join(dir, "claude.json")
	os.WriteFile(claudeJSON, []byte(`{"api_key":"sk-ant-from-file"}`), 0600)

	d := provider.NewDetector(provider.DetectorConfig{
		ScanFiles:  true,
		ClaudeJSON: claudeJSON,
	})
	results := d.Detect()

	found := false
	for _, r := range results {
		if r.ProviderID == "anthropic" && r.Source == "file" {
			found = true
		}
	}
	if !found {
		t.Errorf("Phải detect được anthropic từ file, got: %v", results)
	}
}

func TestDetector_EmptyWhenNoProviders(t *testing.T) {
	// Clear tất cả env vars liên quan
	for _, key := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GITHUB_TOKEN"} {
		os.Unsetenv(key)
	}

	d := provider.NewDetector(provider.DetectorConfig{ScanFiles: false})
	// Không crash, có thể trả về empty slice
	results := d.Detect()
	_ = results // 0 hoặc nhiều hơn — chỉ verify không panic
}
