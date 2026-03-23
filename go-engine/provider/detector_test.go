package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	for _, key := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GEMINI_API_KEY", "AI_STUDIO_KEY", "GITHUB_TOKEN", "OPENROUTER_API_KEY"} {
		os.Unsetenv(key)
	}

	d := provider.NewDetector(provider.DetectorConfig{ScanFiles: false})
	// Không crash, có thể trả về empty slice
	results := d.Detect()
	_ = results // 0 hoặc nhiều hơn — chỉ verify không panic
}

func TestDetector_CLIScannerNoCrash(t *testing.T) {
	// CLI scanner không crash khi `gh` không tồn tại
	d := provider.NewDetector(provider.DetectorConfig{})
	results := d.Detect()
	t.Logf("Detect results: %d providers", len(results))
}

func TestDetector_LocalPortScanner(t *testing.T) {
	// Mock server giả local AI server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"models":[{"name":"llama3"}]}`)
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	d := provider.NewDetector(provider.DetectorConfig{LocalPorts: []string{addr}})
	results := d.Detect()

	found := false
	for _, r := range results {
		if r.Source == "localport" {
			found = true
		}
	}
	if !found {
		t.Error("expected localport detection from mock server")
	}
}

func TestDetector_GeminiEnvKey(t *testing.T) {
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	d := provider.NewDetector(provider.DetectorConfig{})
	results := d.Detect()

	found := false
	for _, r := range results {
		if r.ProviderID == "gemini" && r.Source == "env" {
			found = true
		}
	}
	if !found {
		t.Error("expected gemini from GEMINI_API_KEY")
	}
}
