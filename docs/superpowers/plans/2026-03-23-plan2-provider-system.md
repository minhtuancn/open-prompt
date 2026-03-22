# Provider System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Xây dựng hệ thống provider đầy đủ — keychain, auto-detect, registry, token manager, file watcher, model fallback, OpenAI/Ollama providers, API handlers và frontend ProvidersTab.

**Architecture:** Go engine mở rộng thêm package `provider/` xử lý toàn bộ vòng đời của API keys (lưu vào system keychain qua `zalando/go-keyring`, auto-detect từ env/config files, watch file changes). Package `model/` mở rộng thêm fallback chain đọc từ bảng `model_priority`. Frontend React thêm `ProvidersTab` để quản lý providers và kéo-thả ưu tiên model.

**Tech Stack:** Go 1.22+, `zalando/go-keyring` v0.2.x, `fsnotify/fsnotify` v1.7.x, React 18, `@dnd-kit/core` cho drag-and-drop

---

## Spec Reference
`docs/superpowers/specs/2026-03-22-open-prompt-design.md`

## Previous Plan
`docs/superpowers/plans/2026-03-22-phase1-foundation.md`

---

## File Map

### Go Engine — New Files
```
go-engine/
├── provider/
│   ├── keychain.go           ← CRUD lên system keychain via go-keyring
│   ├── keychain_test.go
│   ├── detector.go           ← scan env vars + config files + process
│   ├── detector_test.go
│   ├── registry.go           ← danh sách providers, models, cost table
│   ├── registry_test.go
│   ├── token_manager.go      ← validate/save/delete token, sync DB ↔ keychain
│   ├── token_manager_test.go
│   ├── watcher.go            ← fsnotify watcher → trigger re-detect
│   └── watcher_test.go
├── model/
│   ├── fallback.go           ← priority chain, sequential/latency/cost strategy
│   ├── fallback_test.go
│   └── providers/
│       ├── anthropic.go      ← (đã có)
│       ├── openai.go         ← OpenAI-compatible streaming provider
│       ├── openai_test.go
│       ├── ollama.go         ← Ollama local provider
│       └── ollama_test.go
├── api/
│   ├── router.go             ← MODIFY: thêm 4 route providers.*
│   └── handlers_providers.go ← NEW: handleProvidersList, Detect, Connect, SetPriority
└── db/
    └── repos/
        ├── provider_tokens_repo.go   ← NEW: CRUD cho provider_tokens table
        └── model_priority_repo.go    ← NEW: CRUD cho model_priority table
```

### Frontend — New Files
```
src/
└── components/
    └── settings/
        └── ProvidersTab.tsx   ← danh sách providers, manual add, drag-drop priority
```

---

## Task 1: Cài đặt dependencies

**Files:**
- Modify: `go-engine/go.mod`, `go-engine/go.sum`

- [ ] **Step 1.1: Thêm go-keyring và fsnotify**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go get github.com/zalando/go-keyring@v0.2.6 && \
  go get github.com/fsnotify/fsnotify@v1.7.0 && \
  go mod tidy
```
Expected output: dòng `go: added github.com/zalando/go-keyring v0.2.6` và `go: added github.com/fsnotify/fsnotify v1.7.0`

- [ ] **Step 1.2: Verify go.mod có đủ dependencies**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  grep -E "go-keyring|fsnotify" go.mod
```
Expected:
```
github.com/fsnotify/fsnotify v1.7.0
github.com/zalando/go-keyring v0.2.6
```

- [ ] **Step 1.3: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add go.mod go.sum && \
  git commit -m "chore: thêm go-keyring và fsnotify dependencies"
```

---

## Task 2: Provider Registry

**Files:**
- Create: `go-engine/provider/registry.go`
- Create: `go-engine/provider/registry_test.go`

- [ ] **Step 2.1: Viết failing test**

Tạo file `go-engine/provider/registry_test.go`:
```go
package provider_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestRegistryKnownProviders(t *testing.T) {
	// Registry phải có ít nhất anthropic, openai, ollama
	reg := provider.DefaultRegistry()
	for _, id := range []string{"anthropic", "openai", "ollama"} {
		p, ok := reg.Get(id)
		if !ok {
			t.Errorf("provider %q không tồn tại trong registry", id)
			continue
		}
		if len(p.Models) == 0 {
			t.Errorf("provider %q phải có ít nhất 1 model", id)
		}
	}
}

func TestRegistryModelCost(t *testing.T) {
	reg := provider.DefaultRegistry()
	p, _ := reg.Get("anthropic")
	// claude-3-5-sonnet phải có cost > 0
	found := false
	for _, m := range p.Models {
		if m.ID == "claude-3-5-sonnet-20241022" {
			found = true
			if m.InputCostPer1K <= 0 || m.OutputCostPer1K <= 0 {
				t.Errorf("model %q phải có cost > 0", m.ID)
			}
		}
	}
	if !found {
		t.Error("claude-3-5-sonnet-20241022 không có trong registry anthropic")
	}
}

func TestRegistryList(t *testing.T) {
	reg := provider.DefaultRegistry()
	list := reg.List()
	if len(list) < 3 {
		t.Errorf("List() phải trả về ít nhất 3 providers, got %d", len(list))
	}
}
```

- [ ] **Step 2.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestRegistry
```
Expected: FAIL với `cannot find package` hoặc `undefined: provider.DefaultRegistry`

- [ ] **Step 2.3: Implement registry.go**

Tạo file `go-engine/provider/registry.go`:
```go
package provider

// ModelInfo mô tả một model cụ thể của provider
type ModelInfo struct {
	ID               string
	Name             string
	ContextWindow    int
	InputCostPer1K   float64 // USD per 1K input tokens
	OutputCostPer1K  float64 // USD per 1K output tokens
	SupportsStreaming bool
}

// ProviderInfo mô tả một AI provider
type ProviderInfo struct {
	ID          string
	Name        string
	AuthType    string // "api_key" | "oauth" | "local"
	BaseURL     string
	Models      []ModelInfo
	EnvVarNames []string // biến môi trường chứa API key
}

// Registry lưu danh sách providers đã biết
type Registry struct {
	providers map[string]ProviderInfo
}

// DefaultRegistry trả về registry mặc định với providers phổ biến
func DefaultRegistry() *Registry {
	r := &Registry{providers: make(map[string]ProviderInfo)}

	r.providers["anthropic"] = ProviderInfo{
		ID:          "anthropic",
		Name:        "Anthropic Claude",
		AuthType:    "api_key",
		BaseURL:     "https://api.anthropic.com/v1",
		EnvVarNames: []string{"ANTHROPIC_API_KEY", "CLAUDE_API_KEY"},
		Models: []ModelInfo{
			{
				ID:               "claude-opus-4-5",
				Name:             "Claude Opus 4.5",
				ContextWindow:    200000,
				InputCostPer1K:   0.015,
				OutputCostPer1K:  0.075,
				SupportsStreaming: true,
			},
			{
				ID:               "claude-sonnet-4-5",
				Name:             "Claude Sonnet 4.5",
				ContextWindow:    200000,
				InputCostPer1K:   0.003,
				OutputCostPer1K:  0.015,
				SupportsStreaming: true,
			},
			{
				ID:               "claude-3-5-sonnet-20241022",
				Name:             "Claude 3.5 Sonnet",
				ContextWindow:    200000,
				InputCostPer1K:   0.003,
				OutputCostPer1K:  0.015,
				SupportsStreaming: true,
			},
			{
				ID:               "claude-3-haiku-20240307",
				Name:             "Claude 3 Haiku",
				ContextWindow:    200000,
				InputCostPer1K:   0.00025,
				OutputCostPer1K:  0.00125,
				SupportsStreaming: true,
			},
		},
	}

	r.providers["openai"] = ProviderInfo{
		ID:          "openai",
		Name:        "OpenAI",
		AuthType:    "api_key",
		BaseURL:     "https://api.openai.com/v1",
		EnvVarNames: []string{"OPENAI_API_KEY"},
		Models: []ModelInfo{
			{
				ID:               "gpt-4o",
				Name:             "GPT-4o",
				ContextWindow:    128000,
				InputCostPer1K:   0.005,
				OutputCostPer1K:  0.015,
				SupportsStreaming: true,
			},
			{
				ID:               "gpt-4o-mini",
				Name:             "GPT-4o Mini",
				ContextWindow:    128000,
				InputCostPer1K:   0.00015,
				OutputCostPer1K:  0.0006,
				SupportsStreaming: true,
			},
			{
				ID:               "gpt-4-turbo",
				Name:             "GPT-4 Turbo",
				ContextWindow:    128000,
				InputCostPer1K:   0.01,
				OutputCostPer1K:  0.03,
				SupportsStreaming: true,
			},
		},
	}

	r.providers["gemini"] = ProviderInfo{
		ID:          "gemini",
		Name:        "Google Gemini",
		AuthType:    "api_key",
		BaseURL:     "https://generativelanguage.googleapis.com/v1beta",
		EnvVarNames: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		Models: []ModelInfo{
			{
				ID:               "gemini-2.0-flash",
				Name:             "Gemini 2.0 Flash",
				ContextWindow:    1000000,
				InputCostPer1K:   0.000075,
				OutputCostPer1K:  0.0003,
				SupportsStreaming: true,
			},
			{
				ID:               "gemini-1.5-pro",
				Name:             "Gemini 1.5 Pro",
				ContextWindow:    2000000,
				InputCostPer1K:   0.00125,
				OutputCostPer1K:  0.005,
				SupportsStreaming: true,
			},
		},
	}

	r.providers["ollama"] = ProviderInfo{
		ID:          "ollama",
		Name:        "Ollama (Local)",
		AuthType:    "local",
		BaseURL:     "http://localhost:11434/v1",
		EnvVarNames: []string{"OLLAMA_HOST"},
		Models: []ModelInfo{
			{
				ID:               "llama3.2",
				Name:             "Llama 3.2",
				ContextWindow:    131072,
				InputCostPer1K:   0,
				OutputCostPer1K:  0,
				SupportsStreaming: true,
			},
			{
				ID:               "mistral",
				Name:             "Mistral 7B",
				ContextWindow:    32768,
				InputCostPer1K:   0,
				OutputCostPer1K:  0,
				SupportsStreaming: true,
			},
			{
				ID:               "qwen2.5",
				Name:             "Qwen 2.5",
				ContextWindow:    131072,
				InputCostPer1K:   0,
				OutputCostPer1K:  0,
				SupportsStreaming: true,
			},
		},
	}

	return r
}

// Get trả về provider theo ID
func (r *Registry) Get(id string) (ProviderInfo, bool) {
	p, ok := r.providers[id]
	return p, ok
}

// List trả về tất cả providers
func (r *Registry) List() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}
```

- [ ] **Step 2.4: Chạy test để xác nhận PASS**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestRegistry
```
Expected: `PASS` với 3 test cases

- [ ] **Step 2.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add provider/ && \
  git commit -m "feat: thêm provider registry với anthropic, openai, gemini, ollama"
```

---

## Task 3: Keychain Layer

**Files:**
- Create: `go-engine/provider/keychain.go`
- Create: `go-engine/provider/keychain_test.go`

- [ ] **Step 3.1: Viết failing test**

Tạo file `go-engine/provider/keychain_test.go`:
```go
package provider_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestKeychainSetGet(t *testing.T) {
	kc := provider.NewKeychain("open-prompt-test")

	// Lưu token
	err := kc.Set("test-provider", "user1", "sk-test-secret-key")
	if err != nil {
		t.Fatalf("Set() thất bại: %v", err)
	}

	// Đọc lại
	val, err := kc.Get("test-provider", "user1")
	if err != nil {
		t.Fatalf("Get() thất bại: %v", err)
	}
	if val != "sk-test-secret-key" {
		t.Errorf("Get() = %q, want %q", val, "sk-test-secret-key")
	}
}

func TestKeychainDelete(t *testing.T) {
	kc := provider.NewKeychain("open-prompt-test")

	kc.Set("del-provider", "user1", "token-to-delete")

	err := kc.Delete("del-provider", "user1")
	if err != nil {
		t.Fatalf("Delete() thất bại: %v", err)
	}

	_, err = kc.Get("del-provider", "user1")
	if err == nil {
		t.Error("Get() sau Delete() phải trả về error")
	}
}

func TestKeychainGetNotFound(t *testing.T) {
	kc := provider.NewKeychain("open-prompt-test")

	_, err := kc.Get("nonexistent-provider", "user1")
	if err == nil {
		t.Error("Get() với key không tồn tại phải trả về error")
	}
}

func TestKeychainBuildKey(t *testing.T) {
	kc := provider.NewKeychain("my-app")
	key := kc.BuildKey("anthropic", "user42")
	if key != "my-app:anthropic:user42" {
		t.Errorf("BuildKey() = %q, want %q", key, "my-app:anthropic:user42")
	}
}
```

- [ ] **Step 3.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestKeychain
```
Expected: FAIL với `undefined: provider.NewKeychain`

- [ ] **Step 3.3: Implement keychain.go**

Tạo file `go-engine/provider/keychain.go`:
```go
package provider

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

// Keychain wrap zalando/go-keyring để lưu API keys trên system keychain.
// Linux: libsecret/GNOME Keyring hoặc KWallet
// macOS: Keychain Services
// Windows: Windows Credential Manager
type Keychain struct {
	// appName là tên service trong keychain, dùng để namespace các keys
	appName string
}

// NewKeychain tạo keychain mới với app name làm namespace
func NewKeychain(appName string) *Keychain {
	return &Keychain{appName: appName}
}

// BuildKey tạo keychain key theo format "appName:providerID:userID"
func (k *Keychain) BuildKey(providerID, userID string) string {
	return fmt.Sprintf("%s:%s:%s", k.appName, providerID, userID)
}

// Set lưu API token vào system keychain
func (k *Keychain) Set(providerID, userID, token string) error {
	key := k.BuildKey(providerID, userID)
	if err := keyring.Set(k.appName, key, token); err != nil {
		return fmt.Errorf("keychain set %q: %w", key, err)
	}
	return nil
}

// Get đọc API token từ system keychain
func (k *Keychain) Get(providerID, userID string) (string, error) {
	key := k.BuildKey(providerID, userID)
	val, err := keyring.Get(k.appName, key)
	if err != nil {
		return "", fmt.Errorf("keychain get %q: %w", key, err)
	}
	return val, nil
}

// Delete xóa API token khỏi system keychain
func (k *Keychain) Delete(providerID, userID string) error {
	key := k.BuildKey(providerID, userID)
	if err := keyring.Delete(k.appName, key); err != nil {
		return fmt.Errorf("keychain delete %q: %w", key, err)
	}
	return nil
}
```

- [ ] **Step 3.4: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestKeychain
```
Expected: PASS (trên CI không có keyring daemon có thể skip bằng `-short`, xem note bên dưới)

> **Note:** Trên môi trường headless/CI không có dbus/libsecret, test keychain có thể fail với `dbus`. Thêm build tag hoặc mock nếu cần. Trong dev environment bình thường PASS.

- [ ] **Step 3.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add provider/keychain.go provider/keychain_test.go && \
  git commit -m "feat: thêm keychain layer dùng zalando/go-keyring"
```

---

## Task 4: Provider Auto-Detector

**Files:**
- Create: `go-engine/provider/detector.go`
- Create: `go-engine/provider/detector_test.go`

- [ ] **Step 4.1: Viết failing test**

Tạo file `go-engine/provider/detector_test.go`:
```go
package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestDetectFromEnvVar(t *testing.T) {
	// Set env var giả
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	detector := provider.NewDetector(provider.DefaultRegistry())
	results := detector.Detect()

	// Phải tìm thấy anthropic từ env
	found := false
	for _, r := range results {
		if r.ProviderID == "anthropic" && r.Source == provider.SourceEnvVar {
			found = true
			if r.Token != "sk-ant-test-key" {
				t.Errorf("token = %q, want %q", r.Token, "sk-ant-test-key")
			}
		}
	}
	if !found {
		t.Error("không detect được anthropic từ env var")
	}
}

func TestDetectFromClaudeCLI(t *testing.T) {
	// Tạo config file giả của Claude CLI
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	configFile := filepath.Join(claudeDir, "claude.json")
	os.WriteFile(configFile, []byte(`{"primaryApiKey":"sk-ant-from-cli-config"}`), 0600)

	detector := provider.NewDetector(provider.DefaultRegistry())
	detector.SetHomeDir(tmpDir) // override home dir cho test
	results := detector.Detect()

	found := false
	for _, r := range results {
		if r.ProviderID == "anthropic" && r.Source == provider.SourceConfigFile {
			found = true
			if r.Token != "sk-ant-from-cli-config" {
				t.Errorf("token = %q, want %q", r.Token, "sk-ant-from-cli-config")
			}
		}
	}
	if !found {
		t.Error("không detect được anthropic từ Claude CLI config")
	}
}

func TestDetectOllamaProcess(t *testing.T) {
	// Test detect Ollama — chỉ kiểm tra hàm không panic
	detector := provider.NewDetector(provider.DefaultRegistry())
	results := detector.Detect()
	// Kết quả tùy môi trường, chỉ cần không crash
	_ = results
}

func TestDetectionSource_String(t *testing.T) {
	tests := []struct {
		src  provider.DetectionSource
		want string
	}{
		{provider.SourceEnvVar, "env_var"},
		{provider.SourceConfigFile, "config_file"},
		{provider.SourceKeychain, "keychain"},
		{provider.SourceProcess, "process"},
	}
	for _, tt := range tests {
		if got := tt.src.String(); got != tt.want {
			t.Errorf("DetectionSource(%d).String() = %q, want %q", tt.src, got, tt.want)
		}
	}
}
```

- [ ] **Step 4.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestDetect
```
Expected: FAIL với `undefined: provider.NewDetector`

- [ ] **Step 4.3: Implement detector.go**

Tạo file `go-engine/provider/detector.go`:
```go
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DetectionSource chỉ ra nguồn phát hiện token
type DetectionSource int

const (
	SourceEnvVar     DetectionSource = iota // từ biến môi trường
	SourceConfigFile                        // từ file config
	SourceKeychain                          // từ system keychain
	SourceProcess                           // từ process đang chạy
)

// String trả về tên nguồn dạng string
func (s DetectionSource) String() string {
	switch s {
	case SourceEnvVar:
		return "env_var"
	case SourceConfigFile:
		return "config_file"
	case SourceKeychain:
		return "keychain"
	case SourceProcess:
		return "process"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// DetectionResult chứa thông tin về provider được phát hiện
type DetectionResult struct {
	ProviderID string
	Source     DetectionSource
	Token      string // rỗng nếu là local provider (ollama)
	ConfigPath string // đường dẫn file config nếu source là config_file
}

// Detector quét các nguồn để tìm provider đã cấu hình
type Detector struct {
	registry *Registry
	homeDir  string // override cho testing
}

// NewDetector tạo detector mới
func NewDetector(registry *Registry) *Detector {
	home, _ := os.UserHomeDir()
	return &Detector{
		registry: registry,
		homeDir:  home,
	}
}

// SetHomeDir cho phép override home directory (dùng trong test)
func (d *Detector) SetHomeDir(dir string) {
	d.homeDir = dir
}

// Detect quét tất cả nguồn và trả về danh sách providers đã tìm thấy.
// Thứ tự ưu tiên: env vars > config files > process detect
func (d *Detector) Detect() []DetectionResult {
	var results []DetectionResult

	// 1. Quét env vars
	results = append(results, d.detectFromEnvVars()...)

	// 2. Quét config files
	results = append(results, d.detectFromConfigFiles()...)

	// 3. Detect process đang chạy (Ollama)
	results = append(results, d.detectFromProcesses()...)

	return results
}

// detectFromEnvVars quét biến môi trường cho tất cả providers
func (d *Detector) detectFromEnvVars() []DetectionResult {
	var results []DetectionResult
	for _, p := range d.registry.List() {
		for _, envVar := range p.EnvVarNames {
			val := os.Getenv(envVar)
			if val != "" {
				results = append(results, DetectionResult{
					ProviderID: p.ID,
					Source:     SourceEnvVar,
					Token:      val,
				})
				break // chỉ lấy env var đầu tiên tìm thấy
			}
		}
	}
	return results
}

// detectFromConfigFiles quét các file config phổ biến
func (d *Detector) detectFromConfigFiles() []DetectionResult {
	var results []DetectionResult

	// Claude CLI: ~/.claude/claude.json
	claudeConfig := filepath.Join(d.homeDir, ".claude", "claude.json")
	if token, err := d.readClaudeCLIConfig(claudeConfig); err == nil && token != "" {
		results = append(results, DetectionResult{
			ProviderID: "anthropic",
			Source:     SourceConfigFile,
			Token:      token,
			ConfigPath: claudeConfig,
		})
	}

	// GitHub Copilot: ~/.config/gh/hosts.yml (dùng OAuth — chỉ detect, không lấy token)
	// Bỏ qua OAuth trong plan này

	// Gemini qua gcloud ADC: ~/.config/gcloud/application_default_credentials.json
	gcloudConfig := filepath.Join(d.homeDir, ".config", "gcloud", "application_default_credentials.json")
	if _, err := os.Stat(gcloudConfig); err == nil {
		// Chỉ detect sự tồn tại, không đọc token ADC vì cần oauth flow
		// Đánh dấu để UI biết gcloud đã cấu hình
		results = append(results, DetectionResult{
			ProviderID: "gemini",
			Source:     SourceConfigFile,
			Token:      "", // cần API key riêng, không dùng ADC
			ConfigPath: gcloudConfig,
		})
	}

	return results
}

// claudeCLIConfig là cấu trúc của ~/.claude/claude.json
type claudeCLIConfig struct {
	PrimaryAPIKey string `json:"primaryApiKey"`
}

// readClaudeCLIConfig đọc API key từ Claude CLI config file
func (d *Detector) readClaudeCLIConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var cfg claudeCLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse claude config: %w", err)
	}
	return cfg.PrimaryAPIKey, nil
}

// detectFromProcesses kiểm tra các process đang chạy
func (d *Detector) detectFromProcesses() []DetectionResult {
	var results []DetectionResult

	// Kiểm tra Ollama đang chạy
	if d.isOllamaRunning() {
		results = append(results, DetectionResult{
			ProviderID: "ollama",
			Source:     SourceProcess,
			Token:      "", // Ollama không cần API key
		})
	}

	return results
}

// isOllamaRunning kiểm tra xem Ollama có đang chạy không
func (d *Detector) isOllamaRunning() bool {
	// Thử connect đến Ollama HTTP API
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq ollama.exe")
		out, err := cmd.Output()
		if err != nil {
			return false
		}
		return len(out) > 100 // có output → process tồn tại
	default:
		// Linux/macOS: dùng pgrep
		err := exec.Command("pgrep", "-x", "ollama").Run()
		return err == nil
	}
}
```

- [ ] **Step 4.4: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run "TestDetect|TestDetectionSource"
```
Expected: PASS

- [ ] **Step 4.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add provider/detector.go provider/detector_test.go && \
  git commit -m "feat: thêm provider auto-detector từ env vars và config files"
```

---

## Task 5: Token Manager

**Files:**
- Create: `go-engine/provider/token_manager.go`
- Create: `go-engine/provider/token_manager_test.go`
- Create: `go-engine/db/repos/provider_tokens_repo.go`

- [ ] **Step 5.1: Tạo provider_tokens_repo.go**

Tạo file `go-engine/db/repos/provider_tokens_repo.go`:
```go
package repos

import (
	"database/sql"
	"fmt"
	"time"
)

// ProviderToken map với bảng provider_tokens trong DB
type ProviderToken struct {
	ID             int64
	UserID         int64
	ProviderID     string
	AuthType       string
	KeychainKey    string
	ExpiresAt      *time.Time
	DetectedAt     *time.Time
	LastRefreshed  *time.Time
	IsActive       bool
}

// ProviderTokenRepo thao tác với bảng provider_tokens
type ProviderTokenRepo struct {
	db *sql.DB
}

// NewProviderTokenRepo tạo repo mới
func NewProviderTokenRepo(db *sql.DB) *ProviderTokenRepo {
	return &ProviderTokenRepo{db: db}
}

// Upsert thêm hoặc cập nhật provider token
func (r *ProviderTokenRepo) Upsert(t ProviderToken) error {
	now := time.Now()
	_, err := r.db.Exec(`
		INSERT INTO provider_tokens (user_id, provider_id, auth_type, keychain_key, detected_at, is_active)
		VALUES (?, ?, ?, ?, ?, 1)
		ON CONFLICT(user_id, provider_id) DO UPDATE SET
			auth_type    = excluded.auth_type,
			keychain_key = excluded.keychain_key,
			detected_at  = excluded.detected_at,
			is_active    = 1
	`, t.UserID, t.ProviderID, t.AuthType, t.KeychainKey, now)
	if err != nil {
		return fmt.Errorf("upsert provider_token: %w", err)
	}
	return nil
}

// GetByUser trả về tất cả tokens của user
func (r *ProviderTokenRepo) GetByUser(userID int64) ([]ProviderToken, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, provider_id, auth_type, keychain_key, is_active
		FROM provider_tokens
		WHERE user_id = ? AND is_active = 1
		ORDER BY provider_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query provider_tokens: %w", err)
	}
	defer rows.Close()

	var tokens []ProviderToken
	for rows.Next() {
		var t ProviderToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.ProviderID, &t.AuthType, &t.KeychainKey, &t.IsActive); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// Delete đánh dấu token không còn active
func (r *ProviderTokenRepo) Delete(userID int64, providerID string) error {
	_, err := r.db.Exec(`
		UPDATE provider_tokens SET is_active = 0
		WHERE user_id = ? AND provider_id = ?
	`, userID, providerID)
	if err != nil {
		return fmt.Errorf("delete provider_token: %w", err)
	}
	return nil
}
```

- [ ] **Step 5.2: Viết failing test cho token manager**

Tạo file `go-engine/provider/token_manager_test.go`:
```go
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
		{"anthropic", "sk-ant-abc123", false},
		{"anthropic", "", true},
		{"anthropic", "invalid-no-prefix", true},
		{"openai", "sk-abc123", false},
		{"openai", "", true},
		{"ollama", "", false}, // ollama không cần key
	}

	for _, tt := range tests {
		err := tm.ValidateKeyFormat(tt.providerID, tt.key)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateKeyFormat(%q, %q) error = %v, wantErr %v",
				tt.providerID, tt.key, err, tt.wantErr)
		}
	}
}
```

- [ ] **Step 5.3: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestTokenManager
```
Expected: FAIL với `undefined: provider.NewTokenManager`

- [ ] **Step 5.4: Implement token_manager.go**

Tạo file `go-engine/provider/token_manager.go`:
```go
package provider

import (
	"fmt"
	"strings"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// TokenManager quản lý vòng đời của API tokens:
// validate format → lưu keychain → sync DB → xóa
type TokenManager struct {
	keychain  *Keychain
	tokenRepo *repos.ProviderTokenRepo
	registry  *Registry
}

// NewTokenManager tạo token manager (cho phép nil deps để test)
func NewTokenManager(kc *Keychain, repo *repos.ProviderTokenRepo, reg *Registry) *TokenManager {
	return &TokenManager{
		keychain:  kc,
		tokenRepo: repo,
		registry:  reg,
	}
}

// ValidateKeyFormat kiểm tra format API key trước khi lưu
func (tm *TokenManager) ValidateKeyFormat(providerID, key string) error {
	// Ollama không cần key
	if providerID == "ollama" {
		return nil
	}

	if key == "" {
		return fmt.Errorf("API key không được để trống")
	}

	switch providerID {
	case "anthropic":
		if !strings.HasPrefix(key, "sk-ant-") {
			return fmt.Errorf("Anthropic API key phải bắt đầu bằng 'sk-ant-'")
		}
	case "openai":
		if !strings.HasPrefix(key, "sk-") {
			return fmt.Errorf("OpenAI API key phải bắt đầu bằng 'sk-'")
		}
	case "gemini":
		if len(key) < 10 {
			return fmt.Errorf("Gemini API key quá ngắn")
		}
	}

	return nil
}

// SaveToken validate format, lưu vào keychain và sync vào DB
func (tm *TokenManager) SaveToken(userID int64, providerID, token string) error {
	if err := tm.ValidateKeyFormat(providerID, token); err != nil {
		return fmt.Errorf("validate key: %w", err)
	}

	// Xác định auth type
	_, exists := tm.registry.Get(providerID)
	if !exists {
		return fmt.Errorf("provider %q không tồn tại trong registry", providerID)
	}

	// Lưu vào keychain nếu có token
	keychainKey := ""
	if tm.keychain != nil && token != "" {
		userIDStr := fmt.Sprintf("%d", userID)
		if err := tm.keychain.Set(providerID, userIDStr, token); err != nil {
			return fmt.Errorf("lưu keychain: %w", err)
		}
		keychainKey = tm.keychain.BuildKey(providerID, userIDStr)
	}

	// Sync vào DB
	if tm.tokenRepo != nil {
		authType := "api_key"
		if providerID == "ollama" {
			authType = "local"
		}
		if err := tm.tokenRepo.Upsert(repos.ProviderToken{
			UserID:      userID,
			ProviderID:  providerID,
			AuthType:    authType,
			KeychainKey: keychainKey,
			IsActive:    true,
		}); err != nil {
			return fmt.Errorf("sync DB: %w", err)
		}
	}

	return nil
}

// GetToken đọc token từ keychain theo userID + providerID
func (tm *TokenManager) GetToken(userID int64, providerID string) (string, error) {
	if tm.keychain == nil {
		return "", fmt.Errorf("keychain chưa được khởi tạo")
	}
	userIDStr := fmt.Sprintf("%d", userID)
	return tm.keychain.Get(providerID, userIDStr)
}

// DeleteToken xóa token khỏi keychain và đánh dấu inactive trong DB
func (tm *TokenManager) DeleteToken(userID int64, providerID string) error {
	// Xóa khỏi keychain
	if tm.keychain != nil {
		userIDStr := fmt.Sprintf("%d", userID)
		// Bỏ qua lỗi nếu key không tồn tại trong keychain
		_ = tm.keychain.Delete(providerID, userIDStr)
	}

	// Đánh dấu inactive trong DB
	if tm.tokenRepo != nil {
		if err := tm.tokenRepo.Delete(userID, providerID); err != nil {
			return fmt.Errorf("xóa DB: %w", err)
		}
	}

	return nil
}
```

- [ ] **Step 5.5: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestTokenManager
```
Expected: PASS

- [ ] **Step 5.6: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add provider/token_manager.go provider/token_manager_test.go \
          db/repos/provider_tokens_repo.go && \
  git commit -m "feat: thêm token manager và provider_tokens repo"
```

---

## Task 6: File Watcher

**Files:**
- Create: `go-engine/provider/watcher.go`
- Create: `go-engine/provider/watcher_test.go`

- [ ] **Step 6.1: Viết failing test**

Tạo file `go-engine/provider/watcher_test.go`:
```go
package provider_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestWatcherTriggerOnChange(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	// Tạo file ban đầu
	os.WriteFile(configFile, []byte(`{"key":"v1"}`), 0644)

	changed := make(chan string, 1)
	w, err := provider.NewWatcher(func(path string) {
		changed <- path
	})
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	defer w.Close()

	if err := w.Watch(configFile); err != nil {
		t.Fatalf("Watch() error: %v", err)
	}

	// Sửa file để trigger event
	time.Sleep(50 * time.Millisecond)
	os.WriteFile(configFile, []byte(`{"key":"v2"}`), 0644)

	select {
	case path := <-changed:
		if path != configFile {
			t.Errorf("path = %q, want %q", path, configFile)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout: không nhận được event thay đổi file")
	}
}

func TestWatcherClose(t *testing.T) {
	w, err := provider.NewWatcher(func(path string) {})
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	// Phải không panic khi Close() được gọi nhiều lần
	w.Close()
	w.Close()
}
```

- [ ] **Step 6.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestWatcher
```
Expected: FAIL với `undefined: provider.NewWatcher`

- [ ] **Step 6.3: Implement watcher.go**

Tạo file `go-engine/provider/watcher.go`:
```go
package provider

import (
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher theo dõi các file config và gọi callback khi có thay đổi.
// Dùng fsnotify — hỗ trợ inotify (Linux), FSEvents (macOS), ReadDirectoryChangesW (Windows).
type Watcher struct {
	fw       *fsnotify.Watcher
	onChange func(path string) // callback khi file thay đổi
	once     sync.Once
	done     chan struct{}
}

// NewWatcher tạo watcher mới với callback
func NewWatcher(onChange func(path string)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fw:       fw,
		onChange: onChange,
		done:     make(chan struct{}),
	}

	go w.loop()
	return w, nil
}

// Watch thêm file vào danh sách theo dõi
func (w *Watcher) Watch(path string) error {
	return w.fw.Add(path)
}

// Unwatch bỏ theo dõi một file
func (w *Watcher) Unwatch(path string) error {
	return w.fw.Remove(path)
}

// Close dừng watcher, an toàn khi gọi nhiều lần
func (w *Watcher) Close() {
	w.once.Do(func() {
		close(w.done)
		w.fw.Close()
	})
}

// loop là goroutine xử lý các filesystem events
func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}
			// Chỉ quan tâm đến Write và Create events
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if w.onChange != nil {
					w.onChange(event.Name)
				}
			}
		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			log.Printf("[watcher] lỗi fsnotify: %v", err)
		}
	}
}
```

- [ ] **Step 6.4: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./provider/... -v -run TestWatcher -timeout 10s
```
Expected: PASS

- [ ] **Step 6.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add provider/watcher.go provider/watcher_test.go && \
  git commit -m "feat: thêm file watcher dùng fsnotify để re-detect khi config thay đổi"
```

---

## Task 7: Model Priority Repo

**Files:**
- Create: `go-engine/db/repos/model_priority_repo.go`

- [ ] **Step 7.1: Implement model_priority_repo.go**

Tạo file `go-engine/db/repos/model_priority_repo.go`:
```go
package repos

import (
	"database/sql"
	"fmt"
)

// ModelPriority map với bảng model_priority trong DB
type ModelPriority struct {
	ID        int64
	UserID    int64
	Priority  int    // thứ tự ưu tiên, số nhỏ = ưu tiên cao hơn
	Provider  string
	Model     string
	IsEnabled bool
}

// ModelPriorityRepo thao tác với bảng model_priority
type ModelPriorityRepo struct {
	db *sql.DB
}

// NewModelPriorityRepo tạo repo mới
func NewModelPriorityRepo(db *sql.DB) *ModelPriorityRepo {
	return &ModelPriorityRepo{db: db}
}

// GetByUser trả về priority chain theo thứ tự ưu tiên tăng dần
func (r *ModelPriorityRepo) GetByUser(userID int64) ([]ModelPriority, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, priority, provider, model, is_enabled
		FROM model_priority
		WHERE user_id = ?
		ORDER BY priority ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query model_priority: %w", err)
	}
	defer rows.Close()

	var items []ModelPriority
	for rows.Next() {
		var m ModelPriority
		if err := rows.Scan(&m.ID, &m.UserID, &m.Priority, &m.Provider, &m.Model, &m.IsEnabled); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// SetChain thay thế toàn bộ priority chain của user
func (r *ModelPriorityRepo) SetChain(userID int64, chain []ModelPriority) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Xóa chain cũ
	if _, err := tx.Exec(`DELETE FROM model_priority WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete old chain: %w", err)
	}

	// Chèn chain mới
	for i, m := range chain {
		if _, err := tx.Exec(`
			INSERT INTO model_priority (user_id, priority, provider, model, is_enabled)
			VALUES (?, ?, ?, ?, ?)
		`, userID, i+1, m.Provider, m.Model, m.IsEnabled); err != nil {
			return fmt.Errorf("insert priority %d: %w", i+1, err)
		}
	}

	return tx.Commit()
}

// DefaultChain trả về chain mặc định nếu user chưa cấu hình
func DefaultChain(userID int64) []ModelPriority {
	return []ModelPriority{
		{UserID: userID, Priority: 1, Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", IsEnabled: true},
		{UserID: userID, Priority: 2, Provider: "openai", Model: "gpt-4o-mini", IsEnabled: true},
		{UserID: userID, Priority: 3, Provider: "ollama", Model: "llama3.2", IsEnabled: true},
	}
}
```

- [ ] **Step 7.2: Build để xác nhận không có lỗi**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go build ./db/repos/...
```
Expected: không có output lỗi

- [ ] **Step 7.3: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add db/repos/model_priority_repo.go && \
  git commit -m "feat: thêm model priority repo với SetChain và DefaultChain"
```

---

## Task 8: Model Fallback

**Files:**
- Create: `go-engine/model/fallback.go`
- Create: `go-engine/model/fallback_test.go`

- [ ] **Step 8.1: Viết failing test**

Tạo file `go-engine/model/fallback_test.go`:
```go
package model_test

import (
	"context"
	"errors"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model"
)

// mockProvider là provider giả để test fallback
type mockProvider struct {
	name    string
	callErr error
	called  bool
}

func (m *mockProvider) StreamComplete(ctx context.Context, req model.StreamRequest, onChunk func(string)) error {
	m.called = true
	if m.callErr != nil {
		return m.callErr
	}
	onChunk("response from " + m.name)
	return nil
}

func TestFallbackChainSuccess(t *testing.T) {
	p1 := &mockProvider{name: "p1", callErr: errors.New("rate limit 429")}
	p2 := &mockProvider{name: "p2"}

	chain := model.NewFallbackChain([]model.NamedProvider{
		{Name: "p1", Provider: p1},
		{Name: "p2", Provider: p2},
	})

	var got string
	err := chain.StreamComplete(context.Background(), model.StreamRequest{Prompt: "hello"}, func(s string) {
		got = s
	})

	if err != nil {
		t.Fatalf("FallbackChain.StreamComplete() error = %v", err)
	}
	if !p1.called {
		t.Error("p1 phải được gọi trước")
	}
	if !p2.called {
		t.Error("p2 phải được gọi khi p1 thất bại")
	}
	if got != "response from p2" {
		t.Errorf("got = %q, want %q", got, "response from p2")
	}
}

func TestFallbackChainAllFail(t *testing.T) {
	p1 := &mockProvider{name: "p1", callErr: errors.New("error")}
	p2 := &mockProvider{name: "p2", callErr: errors.New("error")}

	chain := model.NewFallbackChain([]model.NamedProvider{
		{Name: "p1", Provider: p1},
		{Name: "p2", Provider: p2},
	})

	err := chain.StreamComplete(context.Background(), model.StreamRequest{}, func(s string) {})
	if err == nil {
		t.Error("phải trả về error khi tất cả providers thất bại")
	}
}

func TestIsFallbackError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{errors.New("rate limit 429"), true},
		{errors.New("HTTP 503 service unavailable"), true},
		{errors.New("timeout exceeded"), true},
		{errors.New("context deadline exceeded"), true},
		{errors.New("invalid api key"), false}, // lỗi auth không fallback
		{errors.New("bad request"), false},
	}

	for _, tt := range tests {
		got := model.IsFallbackError(tt.err)
		if got != tt.want {
			t.Errorf("IsFallbackError(%q) = %v, want %v", tt.err.Error(), got, tt.want)
		}
	}
}
```

- [ ] **Step 8.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./model/... -v -run "TestFallback|TestIsFallback"
```
Expected: FAIL với `undefined: model.NewFallbackChain`

- [ ] **Step 8.3: Implement fallback.go**

Tạo file `go-engine/model/fallback.go`:
```go
package model

import (
	"context"
	"fmt"
	"strings"
)

// StreamRequest là request chung cho tất cả providers
type StreamRequest struct {
	Model       string
	Prompt      string
	System      string
	Temperature float64
	MaxTokens   int
}

// StreamProvider là interface chung cho tất cả AI providers
type StreamProvider interface {
	StreamComplete(ctx context.Context, req StreamRequest, onChunk func(string)) error
}

// NamedProvider kết hợp tên và provider
type NamedProvider struct {
	Name     string
	Provider StreamProvider
}

// FallbackChain thực hiện fallback tuần tự qua danh sách providers.
// Khi provider thứ N lỗi với lỗi có thể fallback (429, 5xx, timeout),
// tự động thử provider thứ N+1.
type FallbackChain struct {
	providers []NamedProvider
}

// NewFallbackChain tạo fallback chain mới
func NewFallbackChain(providers []NamedProvider) *FallbackChain {
	return &FallbackChain{providers: providers}
}

// StreamComplete thực hiện streaming, fallback qua chain nếu cần
func (c *FallbackChain) StreamComplete(ctx context.Context, req StreamRequest, onChunk func(string)) error {
	if len(c.providers) == 0 {
		return fmt.Errorf("fallback chain rỗng — không có provider nào")
	}

	var lastErr error
	for _, np := range c.providers {
		err := np.Provider.StreamComplete(ctx, req, onChunk)
		if err == nil {
			return nil // thành công
		}

		// Kiểm tra xem có nên fallback không
		if IsFallbackError(err) {
			lastErr = fmt.Errorf("provider %q thất bại: %w", np.Name, err)
			continue // thử provider tiếp theo
		}

		// Lỗi không thể fallback (auth error, bad request...) → trả về ngay
		return fmt.Errorf("provider %q lỗi không thể fallback: %w", np.Name, err)
	}

	return fmt.Errorf("tất cả providers đều thất bại, lỗi cuối: %w", lastErr)
}

// IsFallbackError kiểm tra xem lỗi có nên trigger fallback không.
// Trigger: 429, 5xx, timeout. Không trigger: 4xx (trừ 429), auth errors.
func IsFallbackError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	// HTTP status codes kích hoạt fallback
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "service unavailable") ||
		strings.Contains(msg, "bad gateway") {
		return true
	}

	// Timeout kích hoạt fallback
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context canceled") {
		return true
	}

	return false
}
```

- [ ] **Step 8.4: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./model/... -v -run "TestFallback|TestIsFallback"
```
Expected: PASS với 3 test cases

- [ ] **Step 8.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add model/fallback.go model/fallback_test.go && \
  git commit -m "feat: thêm model fallback chain với sequential strategy"
```

---

## Task 9: OpenAI Provider

**Files:**
- Create: `go-engine/model/providers/openai.go`
- Create: `go-engine/model/providers/openai_test.go`

- [ ] **Step 9.1: Viết failing test**

Tạo file `go-engine/model/providers/openai_test.go`:
```go
package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func TestOpenAIProviderStreamComplete(t *testing.T) {
	// Mock server trả về SSE response dạng OpenAI
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Kiểm tra headers
		if r.Header.Get("Authorization") == "" {
			t.Error("thiếu Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type phải là application/json")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Gửi SSE events dạng OpenAI
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" World\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	p := providers.NewOpenAIProvider("sk-test-key", server.URL)
	var chunks []string
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "gpt-4o",
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
}

func TestOpenAIProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	p := providers.NewOpenAIProvider("sk-test-key", server.URL)
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "gpt-4o",
		Prompt: "hello",
	}, func(s string) {})

	if err == nil {
		t.Error("phải trả về error khi API trả về 429")
	}
}
```

- [ ] **Step 9.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./model/providers/... -v -run TestOpenAI
```
Expected: FAIL với `undefined: providers.NewOpenAIProvider`

- [ ] **Step 9.3: Implement openai.go**

Tạo file `go-engine/model/providers/openai.go`:
```go
package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIProvider gọi OpenAI Chat Completions API với streaming.
// Cũng tương thích với bất kỳ OpenAI-compatible API nào (vLLM, LM Studio...).
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider tạo provider mới.
// baseURL="" → dùng https://api.openai.com/v1
func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// StreamComplete gọi OpenAI Chat Completions với streaming SSE
func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	body := map[string]interface{}{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"stream":      true,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
	}
	if req.System != "" {
		// OpenAI dùng system message riêng
		body["messages"] = []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.Prompt},
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse Server-Sent Events theo format OpenAI
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			onChunk(event.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}
```

- [ ] **Step 9.4: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./model/providers/... -v -run TestOpenAI
```
Expected: PASS

- [ ] **Step 9.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add model/providers/openai.go model/providers/openai_test.go && \
  git commit -m "feat: thêm OpenAI-compatible streaming provider"
```

---

## Task 10: Ollama Provider

**Files:**
- Create: `go-engine/model/providers/ollama.go`
- Create: `go-engine/model/providers/ollama_test.go`

- [ ] **Step 10.1: Viết failing test**

Tạo file `go-engine/model/providers/ollama_test.go`:
```go
package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func TestOllamaProviderStreamComplete(t *testing.T) {
	// Ollama dùng /api/chat với NDJSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q, want /api/chat", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)

		// Gửi NDJSON chunks
		w.Write([]byte(`{"message":{"content":"Hi"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"content":" there"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"content":""},"done":true}` + "\n"))
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
	// Ollama không chạy → phải trả về lỗi rõ ràng
	p := providers.NewOllamaProvider("http://localhost:19999") // port không tồn tại
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "llama3.2",
		Prompt: "hello",
	}, func(s string) {})

	if err == nil {
		t.Error("phải trả về error khi Ollama không chạy")
	}
}
```

- [ ] **Step 10.2: Chạy test để xác nhận FAIL**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./model/providers/... -v -run TestOllama
```
Expected: FAIL với `undefined: providers.NewOllamaProvider`

- [ ] **Step 10.3: Implement ollama.go**

Tạo file `go-engine/model/providers/ollama.go`:
```go
package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOllamaBaseURL = "http://localhost:11434"

// OllamaProvider gọi Ollama local API với streaming NDJSON.
// Ollama chạy local nên không cần API key.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// NewOllamaProvider tạo provider mới.
// baseURL="" → dùng http://localhost:11434
func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	return &OllamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 120 * time.Second}, // model local cần thêm thời gian
	}
}

// StreamComplete gọi Ollama /api/chat với streaming
func (p *OllamaProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Prompt},
	}
	if req.System != "" {
		messages = append([]map[string]string{{"role": "system", "content": req.System}}, messages...)
	}

	body := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   true,
		"options": map[string]interface{}{
			"num_predict": req.MaxTokens,
			"temperature": req.Temperature,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("kết nối Ollama thất bại (Ollama có đang chạy không?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse NDJSON (Newline Delimited JSON) response của Ollama
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Done {
			break
		}
		if chunk.Message.Content != "" {
			onChunk(chunk.Message.Content)
		}
	}
	return scanner.Err()
}
```

- [ ] **Step 10.4: Chạy test**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./model/providers/... -v -run TestOllama
```
Expected: PASS

- [ ] **Step 10.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add model/providers/ollama.go model/providers/ollama_test.go && \
  git commit -m "feat: thêm Ollama local provider với NDJSON streaming"
```

---

## Task 11: API Handlers cho Providers

**Files:**
- Create: `go-engine/api/handlers_providers.go`
- Modify: `go-engine/api/router.go`

- [ ] **Step 11.1: Tạo handlers_providers.go**

Tạo file `go-engine/api/handlers_providers.go`:
```go
package api

import (
	"encoding/json"
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

// handleProvidersList trả về danh sách providers và trạng thái của chúng
func (r *Router) handleProvidersList(req *Request) (interface{}, *RPCError) {
	// Validate auth
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	// Lấy providers đã kết nối từ DB
	tokenRepo := repos.NewProviderTokenRepo(r.server.db)
	dbTokens, err := tokenRepo.GetByUser(claims.UserID)
	if err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("lỗi đọc DB: %v", err)}
	}

	// Map DB tokens theo providerID
	connected := make(map[string]bool)
	for _, t := range dbTokens {
		connected[t.ProviderID] = true
	}

	// Kết hợp với registry
	reg := provider.DefaultRegistry()
	type ProviderStatus struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		AuthType  string `json:"auth_type"`
		Connected bool   `json:"connected"`
		Models    []provider.ModelInfo `json:"models"`
	}

	var result []ProviderStatus
	for _, p := range reg.List() {
		result = append(result, ProviderStatus{
			ID:        p.ID,
			Name:      p.Name,
			AuthType:  p.AuthType,
			Connected: connected[p.ID],
			Models:    p.Models,
		})
	}

	return result, nil
}

// handleProvidersDetect chạy auto-detect và trả về kết quả
func (r *Router) handleProvidersDetect(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	reg := provider.DefaultRegistry()
	detector := provider.NewDetector(reg)
	results := detector.Detect()

	type DetectResult struct {
		ProviderID string `json:"provider_id"`
		Source     string `json:"source"`
		HasToken   bool   `json:"has_token"`
		ConfigPath string `json:"config_path,omitempty"`
	}

	var out []DetectResult
	for _, r := range results {
		out = append(out, DetectResult{
			ProviderID: r.ProviderID,
			Source:     r.Source.String(),
			HasToken:   r.Token != "",
			ConfigPath: r.ConfigPath,
		})
	}

	return out, nil
}

// handleProvidersConnect lưu API key thủ công
func (r *Router) handleProvidersConnect(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var params struct {
		ProviderID string `json:"provider_id"`
		APIKey     string `json:"api_key"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "params không hợp lệ"}
	}
	if params.ProviderID == "" {
		return nil, &RPCError{Code: -32602, Message: "provider_id không được để trống"}
	}

	kc := provider.NewKeychain("open-prompt")
	tokenRepo := repos.NewProviderTokenRepo(r.server.db)
	reg := provider.DefaultRegistry()
	tm := provider.NewTokenManager(kc, tokenRepo, reg)

	if err := tm.SaveToken(claims.UserID, params.ProviderID, params.APIKey); err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("lưu token thất bại: %v", err)}
	}

	return map[string]interface{}{"ok": true, "provider_id": params.ProviderID}, nil
}

// handleProvidersSetPriority cập nhật model priority chain
func (r *Router) handleProvidersSetPriority(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var params struct {
		Chain []struct {
			Provider  string `json:"provider"`
			Model     string `json:"model"`
			IsEnabled bool   `json:"is_enabled"`
		} `json:"chain"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "params không hợp lệ"}
	}

	prioRepo := repos.NewModelPriorityRepo(r.server.db)

	var chain []repos.ModelPriority
	for _, item := range params.Chain {
		chain = append(chain, repos.ModelPriority{
			UserID:    claims.UserID,
			Provider:  item.Provider,
			Model:     item.Model,
			IsEnabled: item.IsEnabled,
		})
	}

	if err := prioRepo.SetChain(claims.UserID, chain); err != nil {
		return nil, &RPCError{Code: -32000, Message: fmt.Sprintf("lỗi cập nhật priority: %v", err)}
	}

	return map[string]interface{}{"ok": true}, nil
}
```

- [ ] **Step 11.2: Thêm helper requireAuth vào router (nếu chưa có)**

Kiểm tra router.go có `requireAuth` không:
```bash
grep -n "requireAuth" /home/dev/open-prompt-code/open-prompt/go-engine/api/router.go
```

Nếu không có, thêm vào cuối `router.go`:
```go
// requireAuth validate JWT token từ request params
func (r *Router) requireAuth(req *Request) (*auth.Claims, *RPCError) {
	var p struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.Token == "" {
		return nil, &RPCError{Code: -32001, Message: "token bắt buộc"}
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, &RPCError{Code: -32001, Message: "token không hợp lệ"}
	}
	return claims, nil
}
```

- [ ] **Step 11.3: Modify router.go — thêm 4 routes mới**

Mở file `go-engine/api/router.go` và thêm vào switch trong hàm `dispatch`:
```go
	case "providers.list":
		return r.handleProvidersList(req)
	case "providers.detect":
		return r.handleProvidersDetect(req)
	case "providers.connect":
		return r.handleProvidersConnect(req)
	case "providers.set_priority":
		return r.handleProvidersSetPriority(req)
```

Thêm vào trước dòng `default:`.

- [ ] **Step 11.4: Build để verify**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go build ./...
```
Expected: không có lỗi

- [ ] **Step 11.5: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  git add api/handlers_providers.go api/router.go && \
  git commit -m "feat: thêm API handlers providers.list/detect/connect/set_priority"
```

---

## Task 12: Frontend ProvidersTab

**Files:**
- Create: `src/components/settings/ProvidersTab.tsx`

- [ ] **Step 12.1: Cài @dnd-kit cho drag-and-drop**
```bash
cd /home/dev/open-prompt-code/open-prompt && \
  npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
```
Expected: packages được thêm vào package.json

- [ ] **Step 12.2: Tạo ProvidersTab.tsx**

Tạo file `src/components/settings/ProvidersTab.tsx`:
```tsx
import { useState, useEffect } from "react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { invoke } from "@tauri-apps/api/core";

// Thông tin một provider từ API
interface ProviderStatus {
  id: string;
  name: string;
  auth_type: "api_key" | "local" | "oauth";
  connected: boolean;
  models: Array<{ id: string; name: string }>;
}

// Một item trong priority chain
interface PriorityItem {
  id: string; // dùng cho dnd-kit — "provider:model"
  provider: string;
  model: string;
  is_enabled: boolean;
}

// Component một item trong danh sách drag-drop
function SortableModelItem({
  item,
  onToggle,
}: {
  item: PriorityItem;
  onToggle: (id: string) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: item.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-3 p-3 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg mb-2"
    >
      {/* Handle kéo thả */}
      <button
        {...attributes}
        {...listeners}
        className="cursor-grab text-zinc-400 hover:text-zinc-600"
        aria-label="kéo để sắp xếp"
      >
        ⠿
      </button>

      {/* Toggle enable/disable */}
      <input
        type="checkbox"
        checked={item.is_enabled}
        onChange={() => onToggle(item.id)}
        className="w-4 h-4 accent-indigo-500"
      />

      {/* Provider + model */}
      <div className="flex-1">
        <span className="text-sm font-medium text-zinc-800 dark:text-zinc-100">
          {item.model}
        </span>
        <span className="ml-2 text-xs text-zinc-500">{item.provider}</span>
      </div>
    </div>
  );
}

// Component thêm API key thủ công
function ConnectProviderForm({
  providers,
  onConnected,
}: {
  providers: ProviderStatus[];
  onConnected: () => void;
}) {
  const [selectedProvider, setSelectedProvider] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const apiKeyProviders = providers.filter((p) => p.auth_type === "api_key");

  const handleConnect = async () => {
    if (!selectedProvider || !apiKey) return;
    setLoading(true);
    setError("");
    try {
      await invoke("rpc", {
        method: "providers.connect",
        params: { provider_id: selectedProvider, api_key: apiKey },
      });
      setApiKey("");
      onConnected();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-4 border border-zinc-200 dark:border-zinc-700 rounded-lg">
      <h3 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200 mb-3">
        Thêm API Key thủ công
      </h3>
      <div className="flex flex-col gap-2">
        <select
          value={selectedProvider}
          onChange={(e) => setSelectedProvider(e.target.value)}
          className="px-3 py-2 text-sm rounded-md border border-zinc-300 dark:border-zinc-600 bg-white dark:bg-zinc-800 text-zinc-800 dark:text-zinc-100"
        >
          <option value="">Chọn provider...</option>
          {apiKeyProviders.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>

        <input
          type="password"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          placeholder="Nhập API key..."
          className="px-3 py-2 text-sm rounded-md border border-zinc-300 dark:border-zinc-600 bg-white dark:bg-zinc-800 text-zinc-800 dark:text-zinc-100 font-mono"
        />

        {error && <p className="text-xs text-red-500">{error}</p>}

        <button
          onClick={handleConnect}
          disabled={loading || !selectedProvider || !apiKey}
          className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 rounded-md transition-colors"
        >
          {loading ? "Đang lưu..." : "Kết nối"}
        </button>
      </div>
    </div>
  );
}

// Tab chính quản lý providers
export function ProvidersTab({ token }: { token: string }) {
  const [providers, setProviders] = useState<ProviderStatus[]>([]);
  const [priorityChain, setPriorityChain] = useState<PriorityItem[]>([]);
  const [detecting, setDetecting] = useState(false);
  const [savingPriority, setSavingPriority] = useState(false);
  const [loading, setLoading] = useState(true);

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // Load danh sách providers
  const loadProviders = async () => {
    try {
      const result = await invoke<ProviderStatus[]>("rpc", {
        method: "providers.list",
        params: { token },
      });
      setProviders(result || []);
    } catch (e) {
      console.error("Lỗi load providers:", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProviders();
  }, []);

  // Auto-detect providers
  const handleDetect = async () => {
    setDetecting(true);
    try {
      await invoke("rpc", {
        method: "providers.detect",
        params: { token },
      });
      await loadProviders();
    } catch (e) {
      console.error("Lỗi detect providers:", e);
    } finally {
      setDetecting(false);
    }
  };

  // Xử lý drag-drop kết thúc
  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    setPriorityChain((items) => {
      const oldIndex = items.findIndex((i) => i.id === active.id);
      const newIndex = items.findIndex((i) => i.id === over.id);
      return arrayMove(items, oldIndex, newIndex);
    });
  };

  // Toggle enable/disable một model
  const handleToggle = (id: string) => {
    setPriorityChain((items) =>
      items.map((item) =>
        item.id === id ? { ...item, is_enabled: !item.is_enabled } : item
      )
    );
  };

  // Lưu priority chain
  const handleSavePriority = async () => {
    setSavingPriority(true);
    try {
      await invoke("rpc", {
        method: "providers.set_priority",
        params: {
          token,
          chain: priorityChain.map((item) => ({
            provider: item.provider,
            model: item.model,
            is_enabled: item.is_enabled,
          })),
        },
      });
    } catch (e) {
      console.error("Lỗi lưu priority:", e);
    } finally {
      setSavingPriority(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-40 text-zinc-400 text-sm">
        Đang tải...
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-base font-semibold text-zinc-800 dark:text-zinc-100">
          Quản lý AI Providers
        </h2>
        <button
          onClick={handleDetect}
          disabled={detecting}
          className="px-3 py-1.5 text-xs font-medium text-indigo-600 dark:text-indigo-400 border border-indigo-300 dark:border-indigo-600 rounded-md hover:bg-indigo-50 dark:hover:bg-indigo-900/20 disabled:opacity-50 transition-colors"
        >
          {detecting ? "Đang quét..." : "Tự động phát hiện"}
        </button>
      </div>

      {/* Danh sách providers đã kết nối */}
      <div>
        <h3 className="text-xs font-semibold text-zinc-500 uppercase tracking-wide mb-3">
          Trạng thái Providers
        </h3>
        <div className="grid gap-2">
          {providers.map((p) => (
            <div
              key={p.id}
              className="flex items-center gap-3 p-3 rounded-lg border border-zinc-200 dark:border-zinc-700"
            >
              {/* Trạng thái kết nối */}
              <div
                className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  p.connected ? "bg-green-500" : "bg-zinc-300 dark:bg-zinc-600"
                }`}
              />
              <div className="flex-1">
                <span className="text-sm font-medium text-zinc-800 dark:text-zinc-100">
                  {p.name}
                </span>
                <span className="ml-2 text-xs text-zinc-400">
                  {p.auth_type === "local" ? "Local" : `${p.models.length} models`}
                </span>
              </div>
              <span
                className={`text-xs px-2 py-0.5 rounded-full ${
                  p.connected
                    ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                    : "bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
                }`}
              >
                {p.connected ? "Đã kết nối" : "Chưa kết nối"}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Form thêm API key */}
      <ConnectProviderForm providers={providers} onConnected={loadProviders} />

      {/* Priority chain drag-drop */}
      {priorityChain.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-xs font-semibold text-zinc-500 uppercase tracking-wide">
              Thứ tự ưu tiên Model
            </h3>
            <button
              onClick={handleSavePriority}
              disabled={savingPriority}
              className="px-3 py-1 text-xs font-medium text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 rounded-md transition-colors"
            >
              {savingPriority ? "Đang lưu..." : "Lưu thứ tự"}
            </button>
          </div>
          <p className="text-xs text-zinc-400 mb-3">
            Kéo để sắp xếp. Khi model đầu tiên lỗi (429, timeout), tự động dùng model tiếp theo.
          </p>

          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={priorityChain.map((i) => i.id)}
              strategy={verticalListSortingStrategy}
            >
              {priorityChain.map((item) => (
                <SortableModelItem key={item.id} item={item} onToggle={handleToggle} />
              ))}
            </SortableContext>
          </DndContext>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 12.3: Build frontend để xác nhận không có lỗi TypeScript**
```bash
cd /home/dev/open-prompt-code/open-prompt && \
  npm run build 2>&1 | tail -20
```
Expected: build thành công, không có TypeScript error

- [ ] **Step 12.4: Commit**
```bash
cd /home/dev/open-prompt-code/open-prompt && \
  git add src/components/settings/ProvidersTab.tsx package.json package-lock.json && \
  git commit -m "feat: thêm ProvidersTab với auto-detect, manual connect và drag-drop priority"
```

---

## Task 13: Integration — Build toàn bộ

- [ ] **Step 13.1: Chạy tất cả Go tests**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go test ./... -v -timeout 30s 2>&1 | tail -40
```
Expected: tất cả tests PASS (keychain test có thể skip trên headless CI)

- [ ] **Step 13.2: Build Go engine**
```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine && \
  go build -o bin/go-engine ./...
```
Expected: binary được tạo thành công

- [ ] **Step 13.3: Build Tauri app**
```bash
cd /home/dev/open-prompt-code/open-prompt && \
  npm run tauri build 2>&1 | tail -20
```
Expected: build thành công

- [ ] **Step 13.4: Commit tổng kết**
```bash
cd /home/dev/open-prompt-code/open-prompt && \
  git add -A && \
  git commit -m "feat: hoàn thành Plan 2 — Provider System đầy đủ"
```

---

## Tóm tắt

| Task | Files tạo | Mục đích |
|------|-----------|----------|
| 1 | go.mod | Cài go-keyring, fsnotify |
| 2 | provider/registry.go | Danh sách providers + cost table |
| 3 | provider/keychain.go | CRUD keychain cross-platform |
| 4 | provider/detector.go | Auto-detect từ env/config/process |
| 5 | provider/token_manager.go, db/repos/provider_tokens_repo.go | Validate + save token |
| 6 | provider/watcher.go | Watch config files, trigger re-detect |
| 7 | db/repos/model_priority_repo.go | CRUD model priority chain |
| 8 | model/fallback.go | Fallback chain sequential strategy |
| 9 | model/providers/openai.go | OpenAI-compatible streaming |
| 10 | model/providers/ollama.go | Ollama local streaming |
| 11 | api/handlers_providers.go, api/router.go | 4 API endpoints |
| 12 | src/components/settings/ProvidersTab.tsx | Frontend quản lý providers |
| 13 | — | Integration build + test toàn bộ |
