package provider

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectedProvider chứa thông tin một provider được detect
type DetectedProvider struct {
	ProviderID string // "anthropic", "openai", v.v.
	Token      string // API key hoặc access token
	Source     string // "env" | "file" | "process"
	FilePath   string // đường dẫn file nguồn (nếu Source == "file")
}

// DetectorConfig cấu hình cho Detector
type DetectorConfig struct {
	ScanFiles   bool   // có quét config files không
	ClaudeJSON  string // override path ~/.claude/claude.json (cho test)
	GHHostsYAML string // override path ~/.config/gh/hosts.yml (cho test)
}

// Detector phát hiện các AI providers đã cài trên máy
type Detector struct {
	config DetectorConfig
}

// NewDetector tạo Detector mới
func NewDetector(config DetectorConfig) *Detector {
	return &Detector{config: config}
}

// Detect chạy toàn bộ quy trình phát hiện provider
func (d *Detector) Detect() []DetectedProvider {
	var results []DetectedProvider

	// 1. Env vars (ưu tiên cao nhất)
	results = append(results, d.detectFromEnv()...)

	// 2. Config files (chỉ khi được bật)
	if d.config.ScanFiles {
		results = append(results, d.detectFromFiles()...)
	}

	// 3. Running processes (Ollama)
	results = append(results, d.detectFromProcesses()...)

	return results
}

// detectFromEnv phát hiện từ environment variables
func (d *Detector) detectFromEnv() []DetectedProvider {
	var results []DetectedProvider

	envMap := map[string]string{
		"ANTHROPIC_API_KEY": "anthropic",
		"OPENAI_API_KEY":    "openai",
		"GOOGLE_API_KEY":    "gemini",
		"GITHUB_TOKEN":      "github_copilot",
	}

	for envKey, providerID := range envMap {
		if val := os.Getenv(envKey); val != "" {
			results = append(results, DetectedProvider{
				ProviderID: providerID,
				Token:      val,
				Source:     "env",
			})
		}
	}
	return results
}

// detectFromFiles phát hiện từ config files
func (d *Detector) detectFromFiles() []DetectedProvider {
	var results []DetectedProvider

	// ~/.claude/claude.json hoặc ~/.claude.json
	claudePath := d.config.ClaudeJSON
	if claudePath == "" {
		home, _ := os.UserHomeDir()
		// Thử cả hai path
		for _, p := range []string{
			filepath.Join(home, ".claude", "claude.json"),
			filepath.Join(home, ".claude.json"),
		} {
			if _, err := os.Stat(p); err == nil {
				claudePath = p
				break
			}
		}
	}
	if claudePath != "" {
		if token := extractJSONField(claudePath, "api_key"); token != "" {
			results = append(results, DetectedProvider{
				ProviderID: "anthropic",
				Token:      token,
				Source:     "file",
				FilePath:   claudePath,
			})
		}
	}

	// ~/.config/gh/hosts.yml (GitHub Copilot)
	ghPath := d.config.GHHostsYAML
	if ghPath == "" {
		home, _ := os.UserHomeDir()
		ghPath = filepath.Join(home, ".config", "gh", "hosts.yml")
	}
	if _, err := os.Stat(ghPath); err == nil {
		if token := extractGHToken(ghPath); token != "" {
			results = append(results, DetectedProvider{
				ProviderID: "github_copilot",
				Token:      token,
				Source:     "file",
				FilePath:   ghPath,
			})
		}
	}

	return results
}

// detectFromProcesses phát hiện từ running processes
func (d *Detector) detectFromProcesses() []DetectedProvider {
	var results []DetectedProvider

	// Kiểm tra Ollama đang chạy
	if isProcessRunning("ollama") {
		results = append(results, DetectedProvider{
			ProviderID: "ollama",
			Token:      "", // Ollama không cần token
			Source:     "process",
		})
	}
	return results
}

// extractJSONField đọc field từ JSON file
func extractJSONField(path, field string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return ""
	}
	if val, ok := obj[field].(string); ok {
		return val
	}
	return ""
}

// extractGHToken đọc oauth_token từ ~/.config/gh/hosts.yml
func extractGHToken(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "oauth_token:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// isProcessRunning kiểm tra process có đang chạy không
func isProcessRunning(name string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq "+name+".exe")
	default:
		cmd = exec.Command("pgrep", "-x", name)
	}
	return cmd.Run() == nil
}
