# Phase 2A1: Provider Interface + Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Thiết lập Provider interface chuẩn, refactor 3 providers hiện có (Anthropic, OpenAI, Ollama) để implement interface, tạo ProviderRegistry mới với Route/Default/Fallback, thêm ParseMention routing, và refactor query.stream để dùng registry thay vì hardcode Anthropic.

**Architecture:** Package `model/providers/` đã có `CompletionRequest` (trong `anthropic.go`) và `StreamProvider` interface (trong `model/fallback.go`). Ta tạo `Provider` interface mới trong `model/providers/provider.go` — dùng `CompletionRequest` đã có (giữ nguyên field `Temperature`). Refactor `model/fallback.go` để dùng `Provider` interface thay vì `StreamProvider`. `model/router.go` hiện hardcode Anthropic → refactor dùng `ProviderRegistry`. `api/handlers_query.go` sẽ parse `@mention`, route qua registry, và trả `fallback_providers` metadata khi lỗi.

**Tech Stack:** Go 1.22+, modernc.org/sqlite, React 18 + Zustand + TailwindCSS, TypeScript

**Lưu ý quan trọng:**
- Method `GetAuthType()` thay vì `AuthType()` (tránh conflict với type `AuthType`)
- `Name()` trả về "anthropic"/"openai"/"ollama" (khớp với DB key), aliases chứa "claude"/"gpt4"/etc
- `provider.Registry` (package `provider/`) = metadata (models, costs) — giữ nguyên cho `handlers_providers.go`
- `providers.Registry` (package `model/providers/`) = routing (Route, Default, Fallback) — mới
- Migration là 003 (vì 002_seed.sql đã tồn tại)
- Code hiện tại dùng `decodeParams()`, `claims.UserID`, `r.history.Insert(repos.InsertHistoryInput{})`, `SendNotification()` — plan giữ nguyên patterns này

---

## Spec Reference

- `docs/superpowers/specs/2026-03-23-phase2a-multi-provider-design.md`
- `docs/superpowers/specs/2026-03-23-phase2a-implementation-approach.md`

---

## File Map

### Go Engine — New Files

| File | Trách nhiệm |
|------|-------------|
| `go-engine/model/providers/provider.go` | Provider interface, AuthType enum |
| `go-engine/model/providers/provider_test.go` | Test interface compliance |
| `go-engine/model/providers/registry.go` | ProviderRegistry: Register, Route, Default, All, FallbackCandidates |
| `go-engine/model/providers/registry_test.go` | Test registry routing, fallback, alias resolution |
| `go-engine/api/mention.go` | ParseMention() tách @alias từ prompt |
| `go-engine/api/mention_test.go` | Test ParseMention edge cases |
| `go-engine/db/migrations/003_multi_provider.sql` | Bảng model_aliases, custom_gateways |

### Go Engine — Modified Files

| File | Thay đổi |
|------|----------|
| `go-engine/model/providers/anthropic.go` | Thêm Name/DisplayName/Aliases/GetAuthType/Validate/Models |
| `go-engine/model/providers/openai.go` | Thêm Name/DisplayName/Aliases/GetAuthType/Validate/Models |
| `go-engine/model/providers/ollama.go` | Thêm Name/DisplayName/Aliases/GetAuthType/Validate/Models |
| `go-engine/model/fallback.go` | Refactor dùng `providers.Provider` thay vì `StreamProvider` |
| `go-engine/model/router.go` | Refactor dùng ProviderRegistry |
| `go-engine/api/handlers_query.go` | ParseMention + registry.Route + fallback metadata |
| `go-engine/api/router.go` | Thêm `providerRegistry *providers.Registry` vào Router struct |
| `go-engine/db/sqlite.go` | Embed và chạy migration 003 |

---

## Task 1: Tạo Provider Interface

**Files:**
- Create: `go-engine/model/providers/provider.go`
- Create: `go-engine/model/providers/provider_test.go`

- [ ] **Step 1.1: Tạo file provider.go**

`go-engine/model/providers/provider.go`:
```go
package providers

import "context"

// AuthType phân loại cơ chế xác thực của provider
type AuthType string

const (
	AuthAPIKey   AuthType = "api_key"
	AuthOAuth    AuthType = "oauth"
	AuthCLIToken AuthType = "cli_token"
	AuthNone     AuthType = "none"
)

// Provider là interface chung cho tất cả AI providers.
// CompletionRequest đã tồn tại ở anthropic.go với đầy đủ fields
// bao gồm Temperature — giữ nguyên, không tạo lại.
type Provider interface {
	// Name trả về tên chính (khớp DB key: "anthropic", "openai", "ollama")
	Name() string
	// DisplayName trả về tên hiển thị cho UI
	DisplayName() string
	// Aliases trả về tất cả alias (bao gồm cả Name)
	// Ví dụ: anthropic → ["claude", "sonnet", "opus", "haiku", "anthropic"]
	Aliases() []string
	// GetAuthType trả về loại xác thực
	// Tên GetAuthType thay vì AuthType để tránh conflict với type AuthType
	GetAuthType() AuthType
	// StreamComplete gửi request và stream kết quả qua onChunk callback
	StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error
	// Validate kiểm tra kết nối và xác thực
	Validate(ctx context.Context) error
	// Models trả về danh sách model IDs available
	// Anthropic/OpenAI hardcode (ít thay đổi, tránh API call không cần thiết)
	// Ollama query API /api/tags (models thay đổi theo user)
	Models(ctx context.Context) ([]string, error)
}
```

- [ ] **Step 1.2: Tạo compliance tests**

`go-engine/model/providers/provider_test.go`:
```go
package providers

import "testing"

func TestAnthropicImplementsProvider(t *testing.T) {
	var _ Provider = (*AnthropicProvider)(nil)
}

func TestOpenAIImplementsProvider(t *testing.T) {
	var _ Provider = (*OpenAIProvider)(nil)
}

func TestOllamaImplementsProvider(t *testing.T) {
	var _ Provider = (*OllamaProvider)(nil)
}
```

- [ ] **Step 1.3: Chạy test — verify FAIL**

```bash
cd go-engine && go test ./model/providers/ -run TestAnthropicImplementsProvider -v
```

Expected: FAIL — thiếu Name, DisplayName, Aliases, GetAuthType, Validate, Models.

- [ ] **Step 1.4: Commit**

```bash
git add go-engine/model/providers/provider.go go-engine/model/providers/provider_test.go
git commit -m "feat: thêm Provider interface và compliance tests (failing)"
```

---

## Task 2: AnthropicProvider implement Provider interface

**Files:**
- Modify: `go-engine/model/providers/anthropic.go`

- [ ] **Step 2.1: Thêm methods vào AnthropicProvider**

Thêm vào cuối `go-engine/model/providers/anthropic.go` (sau function `StreamComplete`):

```go
// Name trả về tên chính (khớp DB key)
func (p *AnthropicProvider) Name() string { return "anthropic" }

// DisplayName trả về tên hiển thị
func (p *AnthropicProvider) DisplayName() string { return "Anthropic (Claude)" }

// Aliases trả về tất cả alias cho @mention routing
func (p *AnthropicProvider) Aliases() []string {
	return []string{"claude", "sonnet", "opus", "haiku", "anthropic"}
}

// GetAuthType trả về loại xác thực
func (p *AnthropicProvider) GetAuthType() AuthType { return AuthAPIKey }

// Validate kiểm tra API key hợp lệ
func (p *AnthropicProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anthropic validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs (hardcode — ít thay đổi)
func (p *AnthropicProvider) Models(_ context.Context) ([]string, error) {
	return []string{
		"claude-sonnet-4-5-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-opus-4-5",
	}, nil
}
```

- [ ] **Step 2.2: Chạy tests**

```bash
cd go-engine && go test ./model/providers/ -run "TestAnthropic" -v
```

Expected: `TestAnthropicImplementsProvider` PASS, existing tests PASS.

- [ ] **Step 2.3: Commit**

```bash
git add go-engine/model/providers/anthropic.go
git commit -m "feat: AnthropicProvider implement Provider interface"
```

---

## Task 3: OpenAIProvider implement Provider interface

**Files:**
- Modify: `go-engine/model/providers/openai.go`

- [ ] **Step 3.1: Thêm methods vào OpenAIProvider**

Thêm vào cuối `go-engine/model/providers/openai.go`:

```go
// Name trả về tên chính
func (p *OpenAIProvider) Name() string { return "openai" }

// DisplayName trả về tên hiển thị
func (p *OpenAIProvider) DisplayName() string { return "OpenAI (ChatGPT)" }

// Aliases trả về tất cả alias
func (p *OpenAIProvider) Aliases() []string {
	return []string{"gpt4", "gpt", "openai", "o1", "o3", "codex"}
}

// GetAuthType trả về loại xác thực
func (p *OpenAIProvider) GetAuthType() AuthType { return AuthAPIKey }

// Validate kiểm tra API key
func (p *OpenAIProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("openai validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openai validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách model IDs (hardcode)
func (p *OpenAIProvider) Models(_ context.Context) ([]string, error) {
	return []string{"gpt-4o", "gpt-4o-mini", "o1", "o3-mini"}, nil
}
```

- [ ] **Step 3.2: Chạy tests**

```bash
cd go-engine && go test ./model/providers/ -run "TestOpenAI" -v
```

Expected: Tất cả PASS.

- [ ] **Step 3.3: Commit**

```bash
git add go-engine/model/providers/openai.go
git commit -m "feat: OpenAIProvider implement Provider interface"
```

---

## Task 4: OllamaProvider implement Provider interface

**Files:**
- Modify: `go-engine/model/providers/ollama.go`

- [ ] **Step 4.1: Thêm methods vào OllamaProvider**

Thêm vào cuối `go-engine/model/providers/ollama.go`:

```go
// Name trả về tên chính
func (p *OllamaProvider) Name() string { return "ollama" }

// DisplayName trả về tên hiển thị
func (p *OllamaProvider) DisplayName() string { return "Ollama (Local)" }

// Aliases trả về tất cả alias
func (p *OllamaProvider) Aliases() []string {
	return []string{"ollama", "local", "llama"}
}

// GetAuthType trả về loại xác thực
func (p *OllamaProvider) GetAuthType() AuthType { return AuthNone }

// Validate kiểm tra Ollama có đang chạy
func (p *OllamaProvider) Validate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama validate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama validate: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Models trả về danh sách models từ Ollama API (dynamic)
func (p *OllamaProvider) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama models: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	names := make([]string, len(result.Models))
	for i, m := range result.Models {
		names[i] = m.Name
	}
	return names, nil
}
```

- [ ] **Step 4.2: Chạy tất cả provider tests**

```bash
cd go-engine && go test ./model/providers/ -v
```

Expected: 3 compliance tests + 7 existing tests — tất cả PASS.

- [ ] **Step 4.3: Commit**

```bash
git add go-engine/model/providers/ollama.go
git commit -m "feat: OllamaProvider implement Provider interface"
```

---

## Task 5: Tạo ProviderRegistry

**Files:**
- Create: `go-engine/model/providers/registry.go`
- Create: `go-engine/model/providers/registry_test.go`

- [ ] **Step 5.1: Viết tests cho ProviderRegistry**

`go-engine/model/providers/registry_test.go`:
```go
package providers

import (
	"context"
	"testing"
)

// mockProvider dùng cho test registry
type mockProvider struct {
	name        string
	displayName string
	aliases     []string
	authType    AuthType
}

func (m *mockProvider) Name() string                           { return m.name }
func (m *mockProvider) DisplayName() string                    { return m.displayName }
func (m *mockProvider) Aliases() []string                      { return m.aliases }
func (m *mockProvider) GetAuthType() AuthType                  { return m.authType }
func (m *mockProvider) Validate(_ context.Context) error       { return nil }
func (m *mockProvider) Models(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockProvider) StreamComplete(_ context.Context, _ CompletionRequest, onChunk func(string)) error {
	onChunk("mock")
	return nil
}

func TestRegistryRouteByName(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "sonnet", "anthropic"}})

	got, err := r.Route("anthropic")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteByAlias(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "sonnet", "anthropic"}})

	got, err := r.Route("claude")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteCaseInsensitive(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude"}})

	got, err := r.Route("CLAUDE")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteStripAt(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude"}})

	got, err := r.Route("@claude")
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryRouteNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Route("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryDefault(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic"})
	r.Register(&mockProvider{name: "openai"})

	got, err := r.Default()
	if err != nil {
		t.Fatalf("Default error: %v", err)
	}
	if got.Name() != "anthropic" {
		t.Errorf("got %q, want 'anthropic'", got.Name())
	}
}

func TestRegistryDefaultEmpty(t *testing.T) {
	r := NewRegistry()
	_, err := r.Default()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "a"})
	r.Register(&mockProvider{name: "b"})

	if got := len(r.All()); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
}

func TestRegistryFallbackCandidates(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "anthropic"}})
	r.Register(&mockProvider{name: "openai", aliases: []string{"gpt4", "openai"}})
	r.Register(&mockProvider{name: "ollama", aliases: []string{"local", "ollama"}})

	candidates := r.FallbackCandidates("anthropic")
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(candidates))
	}
	for _, c := range candidates {
		if c.Name() == "anthropic" {
			t.Error("should not include failed provider")
		}
	}
}

func TestRegistryFallbackByAlias(t *testing.T) {
	// FallbackCandidates nhận alias, resolve sang name trước khi filter
	r := NewRegistry()
	r.Register(&mockProvider{name: "anthropic", aliases: []string{"claude", "anthropic"}})
	r.Register(&mockProvider{name: "openai"})

	candidates := r.FallbackCandidates("claude")
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0].Name() != "openai" {
		t.Errorf("got %q, want 'openai'", candidates[0].Name())
	}
}
```

- [ ] **Step 5.2: Chạy test — verify FAIL**

```bash
cd go-engine && go test ./model/providers/ -run "TestRegistry" -v
```

Expected: FAIL — `NewRegistry` chưa tồn tại.

- [ ] **Step 5.3: Implement ProviderRegistry**

`go-engine/model/providers/registry.go`:
```go
package providers

import (
	"fmt"
	"strings"
	"sync"
)

// Registry quản lý tất cả providers đã đăng ký
type Registry struct {
	mu        sync.RWMutex
	providers []Provider         // giữ thứ tự đăng ký
	byName    map[string]Provider // name → provider
	aliases   map[string]string   // alias → name
}

// NewRegistry tạo registry rỗng
func NewRegistry() *Registry {
	return &Registry{
		byName:  make(map[string]Provider),
		aliases: make(map[string]string),
	}
}

// Register thêm provider và map tất cả alias
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(p.Name())
	r.providers = append(r.providers, p)
	r.byName[name] = p
	// Map name và aliases → name
	r.aliases[name] = name
	for _, alias := range p.Aliases() {
		r.aliases[strings.ToLower(alias)] = name
	}
}

// Route tìm provider theo alias (case-insensitive, strip "@")
func (r *Registry) Route(alias string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alias = strings.ToLower(strings.TrimPrefix(alias, "@"))
	name, ok := r.aliases[alias]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", alias)
	}
	return r.byName[name], nil
}

// Default trả về provider đầu tiên
func (r *Registry) Default() (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.providers) == 0 {
		return nil, fmt.Errorf("no providers registered")
	}
	return r.providers[0], nil
}

// All trả về tất cả providers theo thứ tự đăng ký
func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

// FallbackCandidates trả về providers thay thế (bỏ qua failed)
// failedName có thể là name hoặc alias — resolve trước khi filter
func (r *Registry) FallbackCandidates(failedName string) []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Resolve alias → name
	resolvedName := strings.ToLower(failedName)
	if name, ok := r.aliases[resolvedName]; ok {
		resolvedName = name
	}

	var candidates []Provider
	for _, p := range r.providers {
		if strings.ToLower(p.Name()) != resolvedName {
			candidates = append(candidates, p)
		}
	}
	return candidates
}
```

- [ ] **Step 5.4: Chạy tests**

```bash
cd go-engine && go test ./model/providers/ -run "TestRegistry" -v
```

Expected: Tất cả PASS (10 tests).

- [ ] **Step 5.5: Commit**

```bash
git add go-engine/model/providers/registry.go go-engine/model/providers/registry_test.go
git commit -m "feat: thêm ProviderRegistry với Route, Default, FallbackCandidates"
```

---

## Task 6: Refactor model/fallback.go dùng providers.Provider

**Files:**
- Modify: `go-engine/model/fallback.go`
- Modify: `go-engine/model/fallback_test.go`

Hiện tại `fallback.go` dùng `StreamProvider` interface riêng với `StreamRequest` (package `model`). Refactor để dùng `providers.Provider` interface — loại bỏ `StreamProvider`, `StreamRequest`, `NamedProvider` (redundant với `Provider.Name()` mới).

- [ ] **Step 6.1: Refactor fallback.go**

Thay toàn bộ `go-engine/model/fallback.go`:

```go
package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// FallbackChain thực hiện fallback tuần tự qua danh sách providers
type FallbackChain struct {
	providers []providers.Provider
}

// NewFallbackChain tạo fallback chain mới
func NewFallbackChain(providerList []providers.Provider) *FallbackChain {
	return &FallbackChain{providers: providerList}
}

// StreamComplete thực hiện streaming, fallback qua chain nếu cần
func (c *FallbackChain) StreamComplete(ctx context.Context, req providers.CompletionRequest, onChunk func(string)) error {
	if len(c.providers) == 0 {
		return fmt.Errorf("fallback chain rỗng — không có provider nào")
	}
	var lastErr error
	for _, p := range c.providers {
		err := p.StreamComplete(ctx, req, onChunk)
		if err == nil {
			return nil
		}
		if IsFallbackError(err) {
			lastErr = fmt.Errorf("provider %q thất bại: %w", p.Name(), err)
			continue
		}
		return fmt.Errorf("provider %q lỗi không thể fallback: %w", p.Name(), err)
	}
	return fmt.Errorf("tất cả providers đều thất bại, lỗi cuối: %w", lastErr)
}

// IsFallbackError kiểm tra xem lỗi có nên trigger fallback không
func IsFallbackError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "503") || strings.Contains(msg, "502") ||
		strings.Contains(msg, "504") || strings.Contains(msg, "500") ||
		strings.Contains(msg, "service unavailable") || strings.Contains(msg, "bad gateway") {
		return true
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context canceled") {
		return true
	}
	return false
}
```

- [ ] **Step 6.2: Cập nhật fallback_test.go**

Thay toàn bộ `go-engine/model/fallback_test.go`:

```go
package model

import (
	"context"
	"fmt"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// testProvider implements providers.Provider cho test
type testProvider struct {
	name   string
	err    error
	called bool
}

func (t *testProvider) Name() string                           { return t.name }
func (t *testProvider) DisplayName() string                    { return t.name }
func (t *testProvider) Aliases() []string                      { return nil }
func (t *testProvider) GetAuthType() providers.AuthType        { return providers.AuthNone }
func (t *testProvider) Validate(_ context.Context) error       { return nil }
func (t *testProvider) Models(_ context.Context) ([]string, error) { return nil, nil }
func (t *testProvider) StreamComplete(_ context.Context, _ providers.CompletionRequest, onChunk func(string)) error {
	t.called = true
	if t.err != nil {
		return t.err
	}
	onChunk("hello from " + t.name)
	return nil
}

func TestFallbackChainSuccess(t *testing.T) {
	p1 := &testProvider{name: "p1", err: fmt.Errorf("HTTP 429 rate limit")}
	p2 := &testProvider{name: "p2"}

	chain := NewFallbackChain([]providers.Provider{p1, p2})
	var chunks []string
	err := chain.StreamComplete(context.Background(), providers.CompletionRequest{}, func(s string) {
		chunks = append(chunks, s)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p1.called || !p2.called {
		t.Error("both providers should be called")
	}
	if len(chunks) == 0 || chunks[0] != "hello from p2" {
		t.Errorf("got chunks=%v, want ['hello from p2']", chunks)
	}
}

func TestFallbackChainAllFail(t *testing.T) {
	p1 := &testProvider{name: "p1", err: fmt.Errorf("HTTP 503")}
	p2 := &testProvider{name: "p2", err: fmt.Errorf("HTTP 502")}

	chain := NewFallbackChain([]providers.Provider{p1, p2})
	err := chain.StreamComplete(context.Background(), providers.CompletionRequest{}, func(string) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsFallbackError(t *testing.T) {
	tests := []struct {
		err  string
		want bool
	}{
		{"HTTP 429 rate limit", true},
		{"HTTP 503", true},
		{"timeout", true},
		{"context deadline exceeded", true},
		{"invalid api key", false},
		{"bad request", false},
	}
	for _, tt := range tests {
		got := IsFallbackError(fmt.Errorf(tt.err))
		if got != tt.want {
			t.Errorf("IsFallbackError(%q)=%v, want %v", tt.err, got, tt.want)
		}
	}
}
```

- [ ] **Step 6.3: Chạy tests**

```bash
cd go-engine && go test ./model/ -v
```

Expected: `TestFallbackChainSuccess`, `TestFallbackChainAllFail`, `TestIsFallbackError` — tất cả PASS.

- [ ] **Step 6.4: Commit**

```bash
git add go-engine/model/fallback.go go-engine/model/fallback_test.go
git commit -m "refactor: fallback.go dùng providers.Provider thay vì StreamProvider"
```

---

## Task 7: ParseMention

**Files:**
- Create: `go-engine/api/mention.go`
- Create: `go-engine/api/mention_test.go`

- [ ] **Step 7.1: Viết tests**

`go-engine/api/mention_test.go`:
```go
package api

import "testing"

func TestParseMentionAtStart(t *testing.T) {
	alias, clean := ParseMention("@claude viết email")
	if alias != "claude" || clean != "viết email" {
		t.Errorf("got (%q, %q), want ('claude', 'viết email')", alias, clean)
	}
}

func TestParseMentionAtEnd(t *testing.T) {
	alias, clean := ParseMention("viết email @gpt4")
	if alias != "gpt4" || clean != "viết email" {
		t.Errorf("got (%q, %q), want ('gpt4', 'viết email')", alias, clean)
	}
}

func TestParseMentionNoMention(t *testing.T) {
	alias, clean := ParseMention("viết email")
	if alias != "" || clean != "viết email" {
		t.Errorf("got (%q, %q), want ('', 'viết email')", alias, clean)
	}
}

func TestParseMentionMiddle(t *testing.T) {
	alias, clean := ParseMention("hãy @claude viết email")
	if alias != "claude" || clean != "hãy viết email" {
		t.Errorf("got (%q, %q), want ('claude', 'hãy viết email')", alias, clean)
	}
}

func TestParseMentionEmpty(t *testing.T) {
	alias, clean := ParseMention("")
	if alias != "" || clean != "" {
		t.Errorf("got (%q, %q), want ('', '')", alias, clean)
	}
}

func TestParseMentionOnlyAlias(t *testing.T) {
	alias, clean := ParseMention("@claude")
	if alias != "claude" || clean != "" {
		t.Errorf("got (%q, %q), want ('claude', '')", alias, clean)
	}
}

func TestParseMentionEmail(t *testing.T) {
	alias, clean := ParseMention("gửi cho user@example.com")
	if alias != "" {
		t.Errorf("email mistaken as mention: alias=%q", alias)
	}
}
```

- [ ] **Step 7.2: Chạy test — verify FAIL**

```bash
cd go-engine && go test ./api/ -run "TestParseMention" -v
```

- [ ] **Step 7.3: Implement ParseMention**

`go-engine/api/mention.go`:
```go
package api

import (
	"regexp"
	"strings"
)

// mentionRegex match @alias — không match email (ký tự word trước @)
var mentionRegex = regexp.MustCompile(`(?:^|\s)@([a-zA-Z0-9_-]+)`)

// ParseMention tách @alias và prompt sạch
func ParseMention(prompt string) (alias, cleanPrompt string) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", ""
	}

	loc := mentionRegex.FindStringSubmatchIndex(prompt)
	if loc == nil {
		return "", prompt
	}

	alias = strings.ToLower(prompt[loc[2]:loc[3]])

	// Xóa match khỏi prompt
	cleanPrompt = prompt[:loc[0]] + prompt[loc[1]:]
	cleanPrompt = strings.TrimSpace(cleanPrompt)
	cleanPrompt = strings.Join(strings.Fields(cleanPrompt), " ")

	return alias, cleanPrompt
}
```

- [ ] **Step 7.4: Chạy tests**

```bash
cd go-engine && go test ./api/ -run "TestParseMention" -v
```

Expected: Tất cả PASS.

- [ ] **Step 7.5: Commit**

```bash
git add go-engine/api/mention.go go-engine/api/mention_test.go
git commit -m "feat: thêm ParseMention() cho @mention routing"
```

---

## Task 8: Refactor router.go + handlers_query.go (gộp — tránh compile break)

**Files:**
- Modify: `go-engine/model/router.go`
- Modify: `go-engine/api/router.go`
- Modify: `go-engine/api/handlers_query.go`

**Lưu ý:** Task này gộp refactor router.go và handlers_query.go để commit cùng lúc — tránh trạng thái code không compile giữa 2 tasks.

- [ ] **Step 8.1: Refactor model/router.go**

Thay toàn bộ `go-engine/model/router.go`:

```go
package model

import (
	"context"
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// Router route request đến đúng provider qua Registry
type Router struct {
	registry *providers.Registry
}

// NewRouter tạo router mới
func NewRouter(registry *providers.Registry) *Router {
	return &Router{registry: registry}
}

// Stream gửi request đến provider theo alias
// alias="" → dùng Default provider
func (r *Router) Stream(ctx context.Context, alias string, req providers.CompletionRequest, onChunk func(string)) error {
	var p providers.Provider
	var err error

	if alias != "" {
		p, err = r.registry.Route(alias)
	} else {
		p, err = r.registry.Default()
	}
	if err != nil {
		return fmt.Errorf("route provider: %w", err)
	}

	return p.StreamComplete(ctx, req, onChunk)
}

// Registry trả về underlying registry
func (r *Router) Registry() *providers.Registry {
	return r.registry
}
```

- [ ] **Step 8.2: Thêm providerRegistry vào api/router.go**

Trong `go-engine/api/router.go`:

1. Thêm import:
```go
"github.com/minhtuancn/open-prompt/go-engine/model/providers"
```

2. Thêm field vào Router struct (sau dòng 27 `registry`):
```go
providerRegistry *providers.Registry
```

3. Trong `newRouter()`, sau dòng 38 (`registry := provider.DefaultRegistry()`), thêm:
```go
// Tạo provider routing registry (khác với metadata registry ở trên)
providerReg := providers.NewRegistry()
```

4. Trong return statement, thêm:
```go
providerRegistry: providerReg,
```

- [ ] **Step 8.3: Refactor handleQueryStream**

Thay toàn bộ function `handleQueryStream` trong `go-engine/api/handlers_query.go`:

```go
package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
	"github.com/minhtuancn/open-prompt/go-engine/engine"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func (r *Router) handleQueryStream(conn net.Conn, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token     string            `json:"token"`
		Input     string            `json:"input"`
		Model     string            `json:"model"`
		System    string            `json:"system"`
		Provider  string            `json:"provider"`
		SlashName string            `json:"slash_name"`
		ExtraVars map[string]string `json:"extra_vars"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, copyErr(ErrInvalidParams)
	}

	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, copyErr(ErrUnauthorized)
	}

	// Resolve slash command nếu có
	finalInput := p.Input
	if p.SlashName != "" {
		builder := engine.NewPromptBuilder()
		resolver := engine.NewCommandResolver(r.prompts, builder)
		resolved, resolveErr := resolver.Resolve(claims.UserID, p.SlashName, p.Input, p.ExtraVars)
		if resolveErr != nil {
			return nil, &RPCError{Code: -32002, Message: resolveErr.Error()}
		}
		if resolved.NeedsVars {
			return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("slash command cần thêm biến: %v", resolved.RequiredVars)}
		}
		finalInput = resolved.RenderedPrompt
	}

	// Xác định provider: explicit param > @mention > default
	alias := p.Provider
	if alias == "" {
		var cleanInput string
		alias, cleanInput = ParseMention(finalInput)
		if alias != "" {
			finalInput = cleanInput
		}
	}

	// Route đến provider
	var prov providers.Provider
	if alias != "" {
		prov, err = r.providerRegistry.Route(alias)
	} else {
		prov, err = r.providerRegistry.Default()
	}

	// Fallback: nếu registry rỗng, thử lấy API key từ settings (tương thích Phase 1)
	if err != nil {
		apiKey, _ := r.settings.Get(claims.UserID, "anthropic_api_key")
		if apiKey != "" {
			prov = providers.NewAnthropicProvider(apiKey)
		} else {
			return nil, &RPCError{Code: ErrProviderNotFound.Code, Message: err.Error()}
		}
	}

	modelName := p.Model
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}

	// Stream response
	start := time.Now()
	var sb strings.Builder

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	streamErr := prov.StreamComplete(ctx, providers.CompletionRequest{
		Model:  modelName,
		Prompt: finalInput,
		System: p.System,
	}, func(chunk string) {
		sb.WriteString(chunk)
		_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	latency := time.Since(start).Milliseconds()
	providerName := prov.Name()

	if streamErr != nil {
		// Thêm fallback_providers khi lỗi
		doneParams := map[string]interface{}{
			"delta":         "",
			"done":          true,
			"error":         fmt.Sprintf("%v", streamErr),
			"error_message": fmt.Sprintf("%s: %v", providerName, streamErr),
		}
		candidates := r.providerRegistry.FallbackCandidates(providerName)
		if len(candidates) > 0 {
			names := make([]string, len(candidates))
			for i, c := range candidates {
				names[i] = c.Name()
			}
			doneParams["fallback_providers"] = names
		}
		_ = SendNotification(conn, "stream.chunk", doneParams)

		_ = r.history.Insert(repos.InsertHistoryInput{
			UserID:    claims.UserID,
			Query:     finalInput,
			Provider:  providerName,
			Model:     modelName,
			LatencyMs: latency,
			Status:    repos.HistoryStatusError,
		})
		return nil, nil
	}

	// Done notification
	_ = SendNotification(conn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})

	_ = r.history.Insert(repos.InsertHistoryInput{
		UserID:    claims.UserID,
		Query:     finalInput,
		Response:  sb.String(),
		Provider:  providerName,
		Model:     modelName,
		LatencyMs: latency,
		Status:    repos.HistoryStatusSuccess,
	})

	return nil, nil
}
```

- [ ] **Step 8.4: Chạy build**

```bash
cd go-engine && go build ./...
```

Expected: Compile success.

- [ ] **Step 8.5: Chạy tất cả tests**

```bash
cd go-engine && go test ./... 2>&1 | tail -30
```

Expected: Tất cả PASS. Test `TestQueryStream` cũ sẽ sử dụng fallback path (API key từ settings) vì registry rỗng.

- [ ] **Step 8.6: Commit**

```bash
git add go-engine/model/router.go go-engine/api/router.go go-engine/api/handlers_query.go
git commit -m "refactor: query.stream dùng ProviderRegistry + ParseMention + fallback metadata"
```

---

## Task 9: DB Migration 003

**Files:**
- Create: `go-engine/db/migrations/003_multi_provider.sql`
- Modify: `go-engine/db/sqlite.go`

- [ ] **Step 9.1: Tạo migration file**

`go-engine/db/migrations/003_multi_provider.sql`:
```sql
-- Migration 003: Phase 2A Multi-Provider
-- Lưu ý: spec gốc ghi migration 002 nhưng thực tế là 003 (002_seed.sql đã tồn tại)

CREATE TABLE IF NOT EXISTS model_aliases (
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alias       TEXT    NOT NULL,
    provider_id TEXT    NOT NULL,
    PRIMARY KEY (user_id, alias)
);

CREATE TABLE IF NOT EXISTS custom_gateways (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    display_name TEXT    NOT NULL,
    base_url     TEXT    NOT NULL,
    api_key      TEXT    DEFAULT '',
    default_model TEXT   DEFAULT '',
    aliases      TEXT    DEFAULT '[]',
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_custom_gateways_user ON custom_gateways(user_id);
```

- [ ] **Step 9.2: Embed migration trong sqlite.go**

Trong `go-engine/db/sqlite.go`, thêm sau dòng 15 (`var initSQL string`):

```go
//go:embed migrations/003_multi_provider.sql
var multiProviderSQL string
```

Trong function `Migrate()`, thêm sau `db.Exec(initSQL)`:

```go
if _, err := db.Exec(multiProviderSQL); err != nil {
	return fmt.Errorf("migration 003: %w", err)
}
```

- [ ] **Step 9.3: Chạy tests**

```bash
cd go-engine && go test ./db/ -v
```

Expected: PASS.

- [ ] **Step 9.4: Commit**

```bash
git add go-engine/db/migrations/003_multi_provider.sql go-engine/db/sqlite.go
git commit -m "feat: thêm migration 003 — model_aliases, custom_gateways"
```

---

## Task 10: Integration Test — @mention Routing với Mock Providers

**Files:**
- Create: `go-engine/api/mention_integration_test.go`

- [ ] **Step 10.1: Viết integration test với mock HTTP servers**

`go-engine/api/mention_integration_test.go`:

**Lưu ý:** Dùng `package api_test` (khớp với tests hiện tại). `setupServer(t)` trả về `(*api.Server, string)`. `registerAndLogin` nhận 4 params (không có secret). Password >= 8 ký tự.

```go
package api_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/api"
	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

func TestQueryStreamMentionSmoke(t *testing.T) {
	// Smoke test: @mention không gây crash khi registry rỗng
	// setupServer tạo registry rỗng → fallback path lấy từ settings
	srv, addr := setupServer(t)
	_ = srv
	token := registerAndLogin(t, addr, "mentiontest", "pass12345678")

	resp := callRPC(t, addr, "test-secret-16chars", "query.stream", map[string]interface{}{
		"token": token,
		"input": "@claude hello",
	})
	// Không có provider configured → error, nhưng không crash
	t.Logf("Response: error=%v", resp.Error)
}

func TestQueryStreamWithProviderParam(t *testing.T) {
	srv, addr := setupServer(t)
	_ = srv
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
	// Dùng api.ParseMention (exported function)
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
```

- [ ] **Step 10.2: Chạy tests**

```bash
cd go-engine && go test ./api/ -run "TestQueryStreamMention|TestParseMentionIntegration" -v
```

Expected: PASS.

- [ ] **Step 10.3: Chạy TOÀN BỘ test suite**

```bash
cd go-engine && go test ./... -count=1 2>&1 | tail -20
```

Expected: Tất cả PASS.

- [ ] **Step 10.4: Commit**

```bash
git add go-engine/api/mention_integration_test.go
git commit -m "test: thêm integration test cho @mention routing + ParseMention"
```

---

## Task 11: Merge và Push

- [ ] **Step 11.1: Chạy full test suite**

```bash
cd go-engine && go test ./... -count=1 -v 2>&1 | tail -40
```

- [ ] **Step 11.2: Merge vào main và push**

```bash
git checkout main && git merge --no-edit && git push origin main
```

- [ ] **Step 11.3: Cập nhật spec status**

Cập nhật `docs/superpowers/specs/2026-03-23-phase2a-implementation-approach.md`: Sub-phase A1 → Completed.
