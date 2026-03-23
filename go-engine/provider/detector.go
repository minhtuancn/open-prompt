package provider

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
	ScanFiles   bool     // có quét config files không
	ClaudeJSON  string   // override path ~/.claude/claude.json (cho test)
	GHHostsYAML string   // override path ~/.config/gh/hosts.yml (cho test)
	LocalPorts  []string // override ports cho test (ví dụ ["localhost:12345"])
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

	results = append(results, d.detectFromEnv()...)
	results = append(results, d.detectFromCLI()...)
	if d.config.ScanFiles {
		results = append(results, d.detectFromFiles()...)
	}
	results = append(results, d.detectFromProcesses()...)
	results = append(results, d.detectFromLocalPorts()...)

	return results
}

// detectFromEnv phát hiện từ environment variables
func (d *Detector) detectFromEnv() []DetectedProvider {
	var results []DetectedProvider

	envMap := map[string]string{
		"ANTHROPIC_API_KEY":  "anthropic",
		"OPENAI_API_KEY":     "openai",
		"GOOGLE_API_KEY":     "gemini",
		"GEMINI_API_KEY":     "gemini",
		"AI_STUDIO_KEY":      "gemini",
		"GITHUB_TOKEN":       "copilot",
		"OPENROUTER_API_KEY": "openrouter",
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

// detectFromCLI phát hiện từ CLI tools (gh, gcloud)
func (d *Detector) detectFromCLI() []DetectedProvider {
	var results []DetectedProvider

	// gh auth token → GitHub Copilot
	if out, err := exec.Command("gh", "auth", "token").Output(); err == nil {
		token := strings.TrimSpace(string(out))
		if token != "" {
			results = append(results, DetectedProvider{
				ProviderID: "copilot",
				Token:      token,
				Source:     "cli",
			})
		}
	}

	return results
}

// detectFromLocalPorts phát hiện local AI servers qua TCP
func (d *Detector) detectFromLocalPorts() []DetectedProvider {
	var results []DetectedProvider

	ports := d.config.LocalPorts
	if len(ports) == 0 {
		ports = []string{"localhost:11434", "localhost:4000", "localhost:8000"}
	}

	portNames := map[string]string{
		"11434": "ollama",
		"4000":  "litellm",
		"8000":  "vllm",
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	for _, addr := range ports {
		// Thử OpenAI-compat endpoint trước
		resp, err := client.Get("http://" + addr + "/v1/models")
		if err != nil {
			// Thử Ollama native endpoint
			resp, err = client.Get("http://" + addr + "/api/tags")
		}
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			name := "gateway"
			parts := strings.Split(addr, ":")
			if len(parts) == 2 {
				if n, ok := portNames[parts[1]]; ok {
					name = n
				}
			}
			results = append(results, DetectedProvider{
				ProviderID: name,
				Token:      "",
				Source:     "localport",
				FilePath:   addr,
			})
		}
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
