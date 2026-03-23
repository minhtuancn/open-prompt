# Phase 2A2: New Providers + Detector — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Thêm 3 providers mới (Gemini, Copilot, Gateway), mở rộng detector, thêm API handlers, và auto-register providers vào ProviderRegistry khi khởi động.

**Architecture:** Gemini dùng API key (OAuth ở A3). Copilot dùng CLIToken qua `gh auth token` (Device Flow ở A3). Gateway reuse SSE parser từ OpenAI (OpenAI-compatible). Detector mở rộng thêm CLI scanner và localport scanner, chạy parallel với errgroup. `newRouter()` auto-register providers dựa trên detected tokens + saved keys. 5 API handlers mới: `providers.add_gateway`, `providers.remove`, `providers.validate`, `providers.set_default`, `providers.rescan`.

**Tech Stack:** Go 1.22+, `golang.org/x/sync/errgroup`, modernc.org/sqlite

**Lưu ý quan trọng:**
- `provider/detector.go` (package `provider`) — mở rộng thêm CLI scanner + localport scanner
- `model/providers/` — 3 files mới: gemini.go, copilot.go, gateway.go
- `api/router.go` — `newRouter()` auto-register providers thay vì registry rỗng
- Test patterns: `setupServer(t)` → `(*api.Server, string)`, `callRPC(t, addr, secret, method, params)`, `registerAndLogin(t, addr, username, password)`, password >= 8 chars
- Package test: `package api_test` (external tests)

---

## Spec Reference

- `docs/superpowers/specs/2026-03-23-phase2a-multi-provider-design.md` (sections 4.3-4.5, 5, 10)
- `docs/superpowers/specs/2026-03-23-phase2a-implementation-approach.md`

---

## File Map

### Go Engine — New Files

| File | Trách nhiệm |
|------|-------------|
| `go-engine/model/providers/gemini.go` | Google Gemini provider (API key) |
| `go-engine/model/providers/gemini_test.go` | Mock SSE test |
| `go-engine/model/providers/copilot.go` | GitHub Copilot provider (CLIToken) |
| `go-engine/model/providers/copilot_test.go` | Mock SSE test |
| `go-engine/model/providers/gateway.go` | Generic OpenAI-compat provider + presets |
| `go-engine/model/providers/gateway_test.go` | Mock SSE test |

### Go Engine — Modified Files

| File | Thay đổi |
|------|----------|
| `go-engine/provider/detector.go` | Thêm CLI scanner, localport scanner, parallel execution |
| `go-engine/provider/detector_test.go` | Tests cho CLI + localport |
| `go-engine/api/handlers_providers.go` | 5 handlers mới |
| `go-engine/api/handlers_providers_test.go` | Tests cho handlers mới |
| `go-engine/api/router.go` | Auto-register providers, thêm routes |

---

## Task 1: GatewayProvider (OpenAI-compatible)

Gateway là đơn giản nhất — reuse SSE pattern từ OpenAI. Triển khai trước vì dùng cho Ollama/LiteLLM/OpenRouter/vLLM.

**Files:**
- Create: `go-engine/model/providers/gateway.go`
- Create: `go-engine/model/providers/gateway_test.go`

- [ ] **Step 1.1: Viết test cho GatewayProvider**

`go-engine/model/providers/gateway_test.go`:
```go
package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayImplementsProvider(t *testing.T) {
	var _ Provider = (*GatewayProvider)(nil)
}

func TestGatewayStreamComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path=%q, want /chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	gw := NewGatewayProvider("test-gw", "Test Gateway", srv.URL, "", "llama3", nil)
	var chunks []string
	err := gw.StreamComplete(context.Background(), CompletionRequest{
		Model:  "llama3",
		Prompt: "hello",
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(chunks) != 2 || chunks[0] != "hello" || chunks[1] != " world" {
		t.Errorf("chunks=%v, want [hello, ' world']", chunks)
	}
}

func TestGatewayWithAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	gw := NewGatewayProvider("or", "OpenRouter", srv.URL, "sk-or-key", "gpt-4o", nil)
	_ = gw.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})

	if gotAuth != "Bearer sk-or-key" {
		t.Errorf("auth=%q, want 'Bearer sk-or-key'", gotAuth)
	}
}

func TestGatewayNoAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	gw := NewGatewayProvider("ollama", "Ollama", srv.URL, "", "llama3", nil)
	_ = gw.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})

	if gotAuth != "" {
		t.Errorf("auth=%q, want empty (no key)", gotAuth)
	}
}

func TestGatewayPresets(t *testing.T) {
	presets := GatewayPresets()
	if len(presets) < 4 {
		t.Fatalf("got %d presets, want >= 4", len(presets))
	}
	// Verify Ollama preset
	found := false
	for _, p := range presets {
		if p.Name == "ollama" {
			found = true
			if p.BaseURL != "http://localhost:11434/v1" {
				t.Errorf("ollama baseURL=%q", p.BaseURL)
			}
		}
	}
	if !found {
		t.Error("ollama preset not found")
	}
}

func TestGatewayValidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("validate path=%q, want /models", r.URL.Path)
		}
		fmt.Fprint(w, `{"data":[{"id":"llama3"}]}`)
	}))
	defer srv.Close()

	gw := NewGatewayProvider("test", "Test", srv.URL, "", "llama3", nil)
	if err := gw.Validate(context.Background()); err != nil {
		t.Fatalf("validate error: %v", err)
	}
}
```

- [ ] **Step 1.2: Chạy test — verify FAIL**

```bash
cd go-engine && go test ./model/providers/ -run "TestGateway" -v
```

Expected: FAIL — `GatewayProvider` chưa tồn tại.

- [ ] **Step 1.3: Implement GatewayProvider**

`go-engine/model/providers/gateway.go`:
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

// GatewayPreset là template cho các gateway phổ biến
type GatewayPreset struct {
	Name         string
	DisplayName  string
	BaseURL      string
	DefaultModel string
}

// GatewayPresets trả về danh sách preset templates
func GatewayPresets() []GatewayPreset {
	return []GatewayPreset{
		{Name: "ollama", DisplayName: "Ollama (Local)", BaseURL: "http://localhost:11434/v1", DefaultModel: "llama3.2"},
		{Name: "litellm", DisplayName: "LiteLLM", BaseURL: "http://localhost:4000/v1", DefaultModel: "gpt-4o"},
		{Name: "openrouter", DisplayName: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", DefaultModel: "openai/gpt-4o"},
		{Name: "vllm", DisplayName: "vLLM", BaseURL: "http://localhost:8000/v1", DefaultModel: ""},
	}
}

// GatewayProvider gọi bất kỳ server nào tương thích OpenAI API
type GatewayProvider struct {
	name         string
	displayName  string
	baseURL      string
	apiKey       string
	defaultModel string
	aliases      []string
	client       *http.Client
}

// NewGatewayProvider tạo gateway provider mới
func NewGatewayProvider(name, displayName, baseURL, apiKey, defaultModel string, aliases []string) *GatewayProvider {
	if aliases == nil {
		aliases = []string{name}
	}
	return &GatewayProvider{
		name:         name,
		displayName:  displayName,
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiKey:       apiKey,
		defaultModel: defaultModel,
		aliases:      aliases,
		client:       &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GatewayProvider) Name() string        { return p.name }
func (p *GatewayProvider) DisplayName() string  { return p.displayName }
func (p *GatewayProvider) Aliases() []string    { return p.aliases }
func (p *GatewayProvider) GetAuthType() AuthType {
	if p.apiKey != "" {
		return AuthAPIKey
	}
	return AuthNone
}

// StreamComplete gọi /chat/completions với SSE (chuẩn OpenAI)
func (p *GatewayProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	modelName := req.Model
	if modelName == "" {
		modelName = p.defaultModel
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Prompt},
	}
	if req.System != "" {
		messages = append([]map[string]string{{"role": "system", "content": req.System}}, messages...)
	}

	body := map[string]interface{}{
		"model":       modelName,
		"messages":    messages,
		"stream":      true,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gateway %s request: %w", p.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway %s API error %d: %s", p.name, resp.StatusCode, string(respBody))
	}

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
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}

// Validate kiểm tra gateway có đang chạy
func (p *GatewayProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gateway %s validate: %w", p.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway %s validate: HTTP %d", p.name, resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách models từ /models endpoint
func (p *GatewayProvider) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway %s models: %w", p.name, err)
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	names := make([]string, len(result.Data))
	for i, m := range result.Data {
		names[i] = m.ID
	}
	return names, nil
}
```

- [ ] **Step 1.4: Chạy tests**

```bash
cd go-engine && go test ./model/providers/ -run "TestGateway" -v
```

Expected: Tất cả PASS (6 tests).

- [ ] **Step 1.5: Commit**

```bash
git add go-engine/model/providers/gateway.go go-engine/model/providers/gateway_test.go
git commit -m "feat: thêm GatewayProvider (OpenAI-compatible) + preset templates"
```

---

## Task 2: GeminiProvider (API key)

**Files:**
- Create: `go-engine/model/providers/gemini.go`
- Create: `go-engine/model/providers/gemini_test.go`

- [ ] **Step 2.1: Viết test cho GeminiProvider**

`go-engine/model/providers/gemini_test.go`:
```go
package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiImplementsProvider(t *testing.T) {
	var _ Provider = (*GeminiProvider)(nil)
}

func TestGeminiStreamComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key trong query param
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("missing API key in query")
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Errorf("missing alt=sse")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		// Gemini SSE format
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]}}]}\n\n")
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" World\"}]}}]}\n\n")
	}))
	defer srv.Close()

	gm := NewGeminiProviderWithBaseURL("test-key", srv.URL)
	var chunks []string
	err := gm.StreamComplete(context.Background(), CompletionRequest{
		Model:  "gemini-1.5-flash",
		Prompt: "hello",
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(chunks) != 2 || chunks[0] != "Hello" || chunks[1] != " World" {
		t.Errorf("chunks=%v, want [Hello, ' World']", chunks)
	}
}

func TestGeminiError401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"message":"invalid key"}}`)
	}))
	defer srv.Close()

	gm := NewGeminiProviderWithBaseURL("bad-key", srv.URL)
	err := gm.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2.2: Implement GeminiProvider**

`go-engine/model/providers/gemini.go`:
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

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// GeminiProvider gọi Google Gemini API với streaming SSE
type GeminiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGeminiProvider tạo provider mới
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return NewGeminiProviderWithBaseURL(apiKey, defaultGeminiBaseURL)
}

// NewGeminiProviderWithBaseURL tạo provider với custom base URL (dùng cho test)
func NewGeminiProviderWithBaseURL(apiKey, baseURL string) *GeminiProvider {
	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *GeminiProvider) Name() string         { return "gemini" }
func (p *GeminiProvider) DisplayName() string   { return "Google Gemini" }
func (p *GeminiProvider) Aliases() []string     { return []string{"gemini", "google", "bard"} }
func (p *GeminiProvider) GetAuthType() AuthType { return AuthAPIKey }

// StreamComplete gọi Gemini streamGenerateContent API
func (p *GeminiProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 8192
	}
	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	// Build Gemini request body
	contents := []map[string]interface{}{
		{"role": "user", "parts": []map[string]string{{"text": req.Prompt}}},
	}
	body := map[string]interface{}{
		"contents":         contents,
		"generationConfig": map[string]interface{}{"maxOutputTokens": req.MaxTokens},
	}
	if req.System != "" {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]string{{"text": req.System}},
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// Gemini API: POST /models/{model}:streamGenerateContent?key={key}&alt=sse
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.baseURL, modelName, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse Gemini SSE
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
			text := event.Candidates[0].Content.Parts[0].Text
			if text != "" {
				onChunk(text)
			}
		}
	}
	return scanner.Err()
}

// Validate kiểm tra API key
func (p *GeminiProvider) Validate(ctx context.Context) error {
	url := fmt.Sprintf("%s/models?key=%s", p.baseURL, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("gemini validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gemini validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs
func (p *GeminiProvider) Models(_ context.Context) ([]string, error) {
	return []string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-2.0-flash"}, nil
}
```

- [ ] **Step 2.3: Chạy tests**

```bash
cd go-engine && go test ./model/providers/ -run "TestGemini" -v
```

Expected: Tất cả PASS.

- [ ] **Step 2.4: Commit**

```bash
git add go-engine/model/providers/gemini.go go-engine/model/providers/gemini_test.go
git commit -m "feat: thêm GeminiProvider (API key, SSE streaming)"
```

---

## Task 3: CopilotProvider (CLIToken)

**Files:**
- Create: `go-engine/model/providers/copilot.go`
- Create: `go-engine/model/providers/copilot_test.go`

- [ ] **Step 3.1: Viết test cho CopilotProvider**

`go-engine/model/providers/copilot_test.go`:
```go
package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCopilotImplementsProvider(t *testing.T) {
	var _ Provider = (*CopilotProvider)(nil)
}

func TestCopilotStreamComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Copilot headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth=%q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Editor-Version") == "" {
			t.Error("missing Editor-Version header")
		}
		// OpenAI-compatible response
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"copilot says hi\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	cp := NewCopilotProviderWithBaseURL("test-token", srv.URL)
	var chunks []string
	err := cp.StreamComplete(context.Background(), CompletionRequest{
		Prompt: "hello",
	}, func(s string) { chunks = append(chunks, s) })

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(chunks) != 1 || chunks[0] != "copilot says hi" {
		t.Errorf("chunks=%v", chunks)
	}
}

func TestCopilotError403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `{"error":"no copilot subscription"}`)
	}))
	defer srv.Close()

	cp := NewCopilotProviderWithBaseURL("bad-token", srv.URL)
	err := cp.StreamComplete(context.Background(), CompletionRequest{Prompt: "hi"}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 3.2: Implement CopilotProvider**

`go-engine/model/providers/copilot.go`:
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

const defaultCopilotBaseURL = "https://api.githubcopilot.com"

// CopilotProvider gọi GitHub Copilot Chat API
type CopilotProvider struct {
	token   string
	baseURL string
	client  *http.Client
}

// NewCopilotProvider tạo provider mới
func NewCopilotProvider(token string) *CopilotProvider {
	return NewCopilotProviderWithBaseURL(token, defaultCopilotBaseURL)
}

// NewCopilotProviderWithBaseURL tạo provider với custom URL (dùng cho test)
func NewCopilotProviderWithBaseURL(token, baseURL string) *CopilotProvider {
	return &CopilotProvider{
		token:   token,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *CopilotProvider) Name() string         { return "copilot" }
func (p *CopilotProvider) DisplayName() string   { return "GitHub Copilot" }
func (p *CopilotProvider) Aliases() []string     { return []string{"copilot", "gh", "github"} }
func (p *CopilotProvider) GetAuthType() AuthType { return AuthCLIToken }

// StreamComplete gọi Copilot /chat/completions (OpenAI-compatible)
func (p *CopilotProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	modelName := req.Model
	if modelName == "" {
		modelName = "gpt-4o"
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Prompt},
	}
	if req.System != "" {
		messages = append([]map[string]string{{"role": "system", "content": req.System}}, messages...)
	}

	body := map[string]interface{}{
		"model":       modelName,
		"messages":    messages,
		"stream":      true,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.token)
	httpReq.Header.Set("Editor-Version", "open-prompt/0.1.0")
	httpReq.Header.Set("Copilot-Integration-Id", "open-prompt")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("copilot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("copilot API error %d: %s", resp.StatusCode, string(respBody))
	}

	// OpenAI-compatible SSE
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
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}

// Validate kiểm tra token hợp lệ
func (p *CopilotProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Editor-Version", "open-prompt/0.1.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("copilot validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("copilot validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs
func (p *CopilotProvider) Models(_ context.Context) ([]string, error) {
	return []string{"gpt-4o", "gpt-4o-mini", "o1-mini"}, nil
}
```

- [ ] **Step 3.3: Chạy tests**

```bash
cd go-engine && go test ./model/providers/ -run "TestCopilot" -v
```

Expected: Tất cả PASS.

- [ ] **Step 3.4: Commit**

```bash
git add go-engine/model/providers/copilot.go go-engine/model/providers/copilot_test.go
git commit -m "feat: thêm CopilotProvider (CLIToken, OpenAI-compat SSE)"
```

---

## Task 4: Mở rộng Detector — CLI scanner + localport scanner

**Files:**
- Modify: `go-engine/provider/detector.go`
- Modify: `go-engine/provider/detector_test.go`

- [ ] **Step 4.1: Viết tests cho CLI + localport scanners**

Thêm vào cuối `go-engine/provider/detector_test.go`:

```go
func TestDetector_DetectFromCLI_GH(t *testing.T) {
	// Test rằng CLI scanner không crash khi `gh` không tồn tại
	d := NewDetector(DetectorConfig{})
	results := d.detectFromCLI()
	// Không assert kết quả cụ thể vì phụ thuộc vào environment
	t.Logf("CLI scan results: %d providers", len(results))
}

func TestDetector_DetectFromLocalPorts(t *testing.T) {
	// Mock HTTP server giả Ollama trên random port
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"models":[{"name":"llama3"}]}`)
	}))
	defer srv.Close()

	// Extract port from srv.URL
	addr := strings.TrimPrefix(srv.URL, "http://")
	d := NewDetector(DetectorConfig{LocalPorts: []string{addr}})
	results := d.detectFromLocalPorts()

	found := false
	for _, r := range results {
		if r.Source == "localport" {
			found = true
		}
	}
	if !found {
		t.Error("expected localport detection")
	}
}
```

Thêm imports cần thiết:
```go
import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)
```

- [ ] **Step 4.2: Mở rộng DetectorConfig và Detect()**

Trong `go-engine/provider/detector.go`:

1. Thêm field `LocalPorts []string` vào `DetectorConfig` (cho test override ports):
```go
type DetectorConfig struct {
	ScanFiles   bool
	ClaudeJSON  string
	GHHostsYAML string
	LocalPorts  []string // override ports cho test
}
```

2. Thêm `detectFromCLI()` method:
```go
// detectFromCLI phát hiện từ CLI tools
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
```

3. Thêm `detectFromLocalPorts()` method:
```go
// detectFromLocalPorts phát hiện local AI servers
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

	for _, addr := range ports {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		resp, err := client.Get("http://" + addr + "/v1/models")
		if err != nil {
			// Thử Ollama endpoint
			resp, err = client.Get("http://" + addr + "/api/tags")
		}
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			// Extract port để xác định tên
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
```

4. Cập nhật `Detect()` để gọi cả CLI và localport:
```go
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
```

Thêm imports: `"net/http"`, `"time"`

- [ ] **Step 4.3: Chạy tests**

```bash
cd go-engine && go test ./provider/ -v
```

Expected: Tất cả tests PASS (cũ + mới).

- [ ] **Step 4.4: Commit**

```bash
git add go-engine/provider/detector.go go-engine/provider/detector_test.go
git commit -m "feat: detector thêm CLI scanner và localport scanner"
```

---

## Task 5: Auto-register providers trong newRouter()

**Files:**
- Modify: `go-engine/api/router.go`

- [ ] **Step 5.1: Cập nhật newRouter() để register providers**

Trong `go-engine/api/router.go`, thay block `providerReg := providers.NewRegistry()`:

```go
// Provider routing registry
providerReg := providers.NewRegistry()

// Auto-register providers từ saved tokens
if tokens, err := tokenRepo.GetByUser(0); err == nil {
	for _, tok := range tokens {
		switch tok.ProviderID {
		case "anthropic":
			providerReg.Register(providers.NewAnthropicProvider(tok.KeychainKey))
		case "openai":
			providerReg.Register(providers.NewOpenAIProvider(tok.KeychainKey, ""))
		case "gemini":
			providerReg.Register(providers.NewGeminiProvider(tok.KeychainKey))
		case "copilot":
			providerReg.Register(providers.NewCopilotProvider(tok.KeychainKey))
		case "ollama":
			providerReg.Register(providers.NewOllamaProvider(""))
		}
	}
}

// Auto-register từ env vars (nếu chưa có trong DB)
if _, err := providerReg.Route("anthropic"); err != nil {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		providerReg.Register(providers.NewAnthropicProvider(key))
	}
}
if _, err := providerReg.Route("openai"); err != nil {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providerReg.Register(providers.NewOpenAIProvider(key, ""))
	}
}
if _, err := providerReg.Route("gemini"); err != nil {
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		providerReg.Register(providers.NewGeminiProvider(key))
	}
}
```

Thêm import `"os"`.

- [ ] **Step 5.2: Chạy build và tests**

```bash
cd go-engine && go build ./... && go test ./... 2>&1 | tail -12
```

Expected: Compile + ALL PASS.

- [ ] **Step 5.3: Commit**

```bash
git add go-engine/api/router.go
git commit -m "feat: auto-register providers từ DB tokens + env vars"
```

---

## Task 6: Thêm API handlers (add_gateway, remove, validate, rescan)

**Files:**
- Modify: `go-engine/api/handlers_providers.go`
- Modify: `go-engine/api/router.go` (thêm routes)

- [ ] **Step 6.1: Thêm handlers**

Thêm vào cuối `go-engine/api/handlers_providers.go`:

```go
// handleProvidersAddGateway thêm custom gateway
func (r *Router) handleProvidersAddGateway(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token        string   `json:"token"`
		Name         string   `json:"name"`
		DisplayName  string   `json:"display_name"`
		BaseURL      string   `json:"base_url"`
		APIKey       string   `json:"api_key"`
		DefaultModel string   `json:"default_model"`
		Aliases      []string `json:"aliases"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" || p.BaseURL == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name và base_url bắt buộc"}
	}

	displayName := p.DisplayName
	if displayName == "" {
		displayName = p.Name
	}

	// Register vào runtime registry
	gw := providers.NewGatewayProvider(p.Name, displayName, p.BaseURL, p.APIKey, p.DefaultModel, p.Aliases)
	r.providerRegistry.Register(gw)

	// Lưu vào DB custom_gateways
	_, err := r.server.db.Exec(
		`INSERT INTO custom_gateways (user_id, name, display_name, base_url, api_key, default_model, aliases)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		claims.UserID, p.Name, displayName, p.BaseURL, p.APIKey, p.DefaultModel, "[]",
	)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal.Code, Message: fmt.Sprintf("save gateway: %v", err)}
	}

	return map[string]interface{}{"ok": true, "name": p.Name}, nil
}

// handleProvidersValidate kiểm tra provider có hoạt động không
func (r *Router) handleProvidersValidate(req *Request) (interface{}, *RPCError) {
	_, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name bắt buộc"}
	}

	prov, err := r.providerRegistry.Route(p.Name)
	if err != nil {
		return nil, &RPCError{Code: ErrProviderNotFound.Code, Message: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	validateErr := prov.Validate(ctx)
	latency := time.Since(start).Milliseconds()

	result := map[string]interface{}{
		"name":       prov.Name(),
		"valid":      validateErr == nil,
		"latency_ms": latency,
	}
	if validateErr != nil {
		result["error"] = validateErr.Error()
	}

	// Lấy models nếu valid
	if validateErr == nil {
		if models, err := prov.Models(ctx); err == nil {
			result["models"] = models
		}
	}

	return result, nil
}

// handleProvidersRemove xóa provider khỏi registry
func (r *Router) handleProvidersRemove(req *Request) (interface{}, *RPCError) {
	claims, rpcErr := r.requireAuth(req)
	if rpcErr != nil {
		return nil, rpcErr
	}

	var p struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := decodeParams(req.Params, &p); err != nil || p.Name == "" {
		return nil, &RPCError{Code: ErrInvalidParams.Code, Message: "name bắt buộc"}
	}

	// Soft-delete token trong DB
	_ = r.tokenRepo.Delete(claims.UserID, p.Name)
	// Xóa custom gateway nếu có
	_, _ = r.server.db.Exec("DELETE FROM custom_gateways WHERE user_id = ? AND name = ?", claims.UserID, p.Name)

	return map[string]interface{}{"ok": true}, nil
}
```

Thêm imports: `"context"`, `"time"`, `"github.com/minhtuancn/open-prompt/go-engine/model/providers"`

- [ ] **Step 6.2: Thêm routes vào dispatch**

Trong `go-engine/api/router.go`, function `dispatch()`, thêm trước `default:`:

```go
case "providers.add_gateway":
	return r.handleProvidersAddGateway(req)
case "providers.validate":
	return r.handleProvidersValidate(req)
case "providers.remove":
	return r.handleProvidersRemove(req)
```

- [ ] **Step 6.3: Chạy build + tests**

```bash
cd go-engine && go build ./... && go test ./... 2>&1 | tail -12
```

Expected: Compile + ALL PASS.

- [ ] **Step 6.4: Commit**

```bash
git add go-engine/api/handlers_providers.go go-engine/api/router.go
git commit -m "feat: thêm providers.add_gateway, providers.validate, providers.remove handlers"
```

---

## Task 7: Tests cho handlers mới

**Files:**
- Modify: `go-engine/api/handlers_providers_test.go`

- [ ] **Step 7.1: Thêm tests**

Thêm vào cuối `go-engine/api/handlers_providers_test.go`:

```go
func TestProvidersAddGateway(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "gwuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.add_gateway", map[string]interface{}{
		"token":         token,
		"name":          "my-ollama",
		"display_name":  "My Ollama",
		"base_url":      "http://localhost:11434/v1",
		"default_model": "llama3",
	})
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["ok"] != true {
		t.Errorf("ok=%v, want true", result["ok"])
	}
}

func TestProvidersAddGatewayMissingName(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "gwuser2", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.add_gateway", map[string]interface{}{
		"token":    token,
		"base_url": "http://localhost:11434/v1",
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Errorf("expected error -32602, got %v", resp.Error)
	}
}

func TestProvidersValidateNotFound(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "valuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.validate", map[string]interface{}{
		"token": token,
		"name":  "nonexistent",
	})
	if resp.Error == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestProvidersRemove(t *testing.T) {
	_, addr := setupServer(t)
	token := registerAndLogin(t, addr, "rmuser", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "providers.remove", map[string]interface{}{
		"token": token,
		"name":  "anthropic",
	})
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["ok"] != true {
		t.Errorf("ok=%v, want true", result["ok"])
	}
}
```

- [ ] **Step 7.2: Chạy tests**

```bash
cd go-engine && go test ./api/ -run "TestProviders" -v 2>&1 | tail -20
```

Expected: Tất cả PASS.

- [ ] **Step 7.3: Chạy full test suite**

```bash
cd go-engine && go test ./... -count=1 2>&1 | tail -12
```

Expected: ALL PASS.

- [ ] **Step 7.4: Commit**

```bash
git add go-engine/api/handlers_providers_test.go
git commit -m "test: thêm tests cho add_gateway, validate, remove handlers"
```

---

## Task 8: Merge và Push

- [ ] **Step 8.1: Chạy full test suite**

```bash
cd go-engine && go test ./... -count=1 -v 2>&1 | tail -40
```

- [ ] **Step 8.2: Merge vào main và push**

```bash
git checkout main && git merge --no-edit && git push origin main
```

- [ ] **Step 8.3: Cập nhật spec status**

Cập nhật `docs/superpowers/specs/2026-03-23-phase2a-implementation-approach.md`: Sub-phase A2 → Completed.
