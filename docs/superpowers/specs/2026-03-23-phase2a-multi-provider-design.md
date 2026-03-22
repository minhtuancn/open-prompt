# Open Prompt — Phase 2A: Multi-Provider Engine

**Ngày:** 2026-03-23
**Trạng thái:** Draft
**Phụ thuộc:** Phase 1 Foundation (hoàn thành)

---

## 1. Mục tiêu

Mở rộng Open Prompt từ 1 provider cứng (Anthropic) sang hệ thống đa provider linh hoạt:

- Tự động phát hiện token/key từ môi trường (env vars, CLI tools, config files, local servers)
- Hỗ trợ OAuth từ trình duyệt hoặc WebView nhúng
- Hỗ trợ gateway routers tương thích OpenAI (Ollama, vLLM, LiteLLM, OpenRouter)
- Routing query qua `@mention` trong prompt hoặc `Ctrl+M` quick-switch
- Fallback tương tác khi provider thất bại

---

## 2. Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────────────────────┐
│  React Overlay                                                  │
│  "@claude viết email" → parseMention() → alias="claude"        │
│  Ctrl+M → model picker → overlayStore.setProvider()            │
│  Fallback dialog → user chọn provider thay thế                 │
└────────────────────┬────────────────────────────────────────────┘
                     │ query.stream {prompt, provider?}
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│  Go Engine — api/handlers_query.go                              │
│  parseMention(prompt) → alias → Registry.Route(alias)          │
│  → provider.StreamComplete() → stream.chunk notifications       │
│  → on error: stream.chunk {done:true, error, fallback_providers}│
└────────────────────┬────────────────────────────────────────────┘
                     │
         ┌───────────▼───────────┐
         │  ProviderRegistry     │
         │  Route(alias)         │
         │  Default()            │
         │  All() → UI list      │
         └───────────┬───────────┘
                     │
    ┌────────────────┼────────────────────────────┐
    ▼                ▼              ▼              ▼
Anthropic        OpenAI         Gemini        Gateway
(refactor)    (+ Codex)      (OAuth/key)   (OpenAI-compat)
                                              Ollama/vLLM
                                              LiteLLM/OpenRouter
```

---

## 3. Provider Interface

### 3.1 Interface Go

**File:** `go-engine/model/providers/interface.go`

```go
package providers

import "context"

// AuthType phân loại cơ chế xác thực của provider
type AuthType string

const (
    AuthAPIKey   AuthType = "api_key"
    AuthOAuth    AuthType = "oauth"
    AuthCLIToken AuthType = "cli_token"
    AuthNone     AuthType = "none" // local models không cần auth
)

// Provider là interface chung cho tất cả AI providers
type Provider interface {
    // Name trả về alias chính (dùng cho @mention và DB key)
    Name() string
    // DisplayName trả về tên hiển thị cho UI
    DisplayName() string
    // Aliases trả về danh sách alias phụ ("@gpt4", "@openai", "@o1")
    Aliases() []string
    // AuthType trả về loại xác thực
    AuthType() AuthType
    // StreamComplete gửi request và stream kết quả qua onChunk callback
    StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error
    // Validate kiểm tra kết nối và xác thực, trả về nil nếu OK
    Validate(ctx context.Context) error
    // Models trả về danh sách model IDs available (có thể rỗng nếu provider không hỗ trợ)
    Models(ctx context.Context) ([]string, error)
}
```

### 3.2 ProviderRegistry

**File:** `go-engine/model/providers/registry.go`

```go
package providers

// Registry quản lý tất cả providers đã đăng ký
type Registry struct {
    mu        sync.RWMutex
    providers map[string]Provider // key = primary name
    aliases   map[string]string   // alias → primary name
    priority  []string            // thứ tự fallback
}

// Register thêm provider vào registry và map tất cả alias của nó
func (r *Registry) Register(p Provider)

// Route tìm provider theo alias (case-insensitive, strip "@")
// Trả về ErrProviderNotFound nếu không tìm thấy
func (r *Registry) Route(alias string) (Provider, error)

// Default trả về provider đầu tiên trong priority list
func (r *Registry) Default() (Provider, error)

// All trả về danh sách tất cả providers theo thứ tự priority
func (r *Registry) All() []Provider

// SetPriority cập nhật thứ tự ưu tiên (persist vào model_priority table)
func (r *Registry) SetPriority(names []string)

// FallbackCandidates trả về providers thay thế khi provider chỉ định thất bại
func (r *Registry) FallbackCandidates(failedName string) []Provider
```

---

## 4. Providers

### 4.1 Anthropic (refactor từ Phase 1)

**File:** `go-engine/model/providers/anthropic.go`

Refactor `AnthropicProvider` hiện có để implement `Provider` interface. Thêm `Name()`, `DisplayName()`, `Aliases()`, `AuthType()`, `Validate()`, `Models()`. Logic `StreamComplete` giữ nguyên.

Aliases: `["claude", "sonnet", "opus", "haiku", "anthropic"]`

### 4.2 OpenAI

**File:** `go-engine/model/providers/openai.go`

- Auth: API key (`OPENAI_API_KEY` hoặc nhập thủ công)
- Endpoint: `https://api.openai.com/v1` (chuẩn OpenAI)
- Aliases: `["gpt4", "gpt", "openai", "o1", "o3", "codex"]`
- Stream: SSE chuẩn OpenAI (`data: {"choices":[{"delta":{"content":"..."}}]}`)
- Azure OpenAI: cùng file, config thêm `azure_endpoint` + `api_version`

### 4.3 Google Gemini

**File:** `go-engine/model/providers/gemini.go`

- Auth: OAuth2 (primary) hoặc API key (`GEMINI_API_KEY`, `AI_STUDIO_KEY`)
- Endpoint: `https://generativelanguage.googleapis.com/v1beta`
- Stream: SSE format riêng của Google
- Aliases: `["gemini", "google", "bard"]`
- Token refresh: dùng `refresh_token` lưu trong `provider_tokens` khi access token hết hạn

### 4.4 GitHub Copilot

**File:** `go-engine/model/providers/copilot.go`

- Auth: Bearer token từ `gh auth token` (CLIToken)
- Endpoint: `https://api.githubcopilot.com/chat/completions`
- Header đặc biệt: `Editor-Version`, `Copilot-Integration-Id`
- Token exchange: GitHub OAuth token → Copilot session token (có expiry, cần refresh)
- Aliases: `["copilot", "gh", "github"]`

### 4.5 Gateway (OpenAI-compatible)

**File:** `go-engine/model/providers/gateway.go`

Provider generic cho tất cả server tương thích OpenAI API:

```go
type GatewayProvider struct {
    name        string   // user-defined, ví dụ "my-ollama"
    displayName string
    baseURL     string   // "http://localhost:11434/v1"
    apiKey      string   // optional
    aliases     []string // user-defined, ví dụ ["@local", "@ollama"]
    model       string   // default model name
}
```

**Preset templates** (Base URL + model mặc định):

| Template | Base URL | Default Model |
|---|---|---|
| Ollama | `http://localhost:11434/v1` | `llama3.2` |
| LiteLLM | `http://localhost:4000/v1` | `gpt-4o` |
| OpenRouter | `https://openrouter.ai/api/v1` | `openai/gpt-4o` |
| vLLM | `http://localhost:8000/v1` | (từ `/v1/models`) |
| Custom | (user nhập) | (user nhập) |

---

## 5. Auto-Detector

**File:** `go-engine/model/providers/detector/detector.go`

Chạy khi app khởi động và khi user nhấn "Re-scan". Tất cả scanners chạy song song với `errgroup` + context timeout 5 giây.

```go
type DetectedProvider struct {
    Alias   string   // key provider: "claude", "copilot"...
    Token   string   // API key hoặc bearer token (không log)
    Source  string   // "env:ANTHROPIC_API_KEY", "cli:gh", "file:~/.config/gh/hosts.yml"
    Valid   bool     // đã Validate() thành công
    Error   string   // lý do nếu invalid
}

// RunAll chạy tất cả scanners song song, trả về kết quả hợp nhất
func RunAll(ctx context.Context) []DetectedProvider
```

### 5.1 Env Scanner

**File:** `go-engine/model/providers/detector/env.go`

```
ANTHROPIC_API_KEY        → claude
OPENAI_API_KEY           → gpt4
GEMINI_API_KEY           → gemini
AI_STUDIO_KEY            → gemini
GITHUB_TOKEN             → copilot (nếu có Copilot subscription)
OPENROUTER_API_KEY       → openrouter gateway
COHERE_API_KEY           → (placeholder Phase 2B)
```

### 5.2 CLI Scanner

**File:** `go-engine/model/providers/detector/cli.go`

```
gh auth token                        → copilot token
gcloud auth print-access-token       → gemini OAuth token
claude config get api_key            → claude key
anthropic config get api_key         → claude key (alternate)
```

Mỗi lệnh chạy với timeout 3 giây. Nếu CLI không tồn tại → skip (không error).

### 5.3 Config File Scanner

**File:** `go-engine/model/providers/detector/configfile.go`

```
~/.config/gh/hosts.yml              → GitHub OAuth token → copilot
~/.claude/config.json               → Anthropic API key
~/.config/litellm/config.yaml       → LiteLLM gateway URL
~/.ollama/                          → Ollama installed (→ thêm local gateway)
```

### 5.4 Local Port Scanner

**File:** `go-engine/model/providers/detector/localport.go`

TCP dial với timeout 500ms:

```
localhost:11434  → Ollama   (GET /api/tags)
localhost:4000   → LiteLLM  (GET /models)
localhost:8000   → vLLM     (GET /v1/models)
localhost:8080   → generic  (GET /v1/models)
```

Nếu port mở và `/models` trả về danh sách → tự động tạo GatewayProvider.

---

## 6. OAuth Flow

### 6.1 Embedded WebView (primary)

**File:** `src-tauri/src/oauth.rs`

```rust
#[command]
pub async fn start_oauth(app: AppHandle, provider: String) -> Result<String, String> {
    // 1. Lấy authorization URL từ Go engine (provider.oauth_start)
    // 2. Mở WebView window 400×600 với URL đó
    // 3. Monitor navigation events, detect redirect về "open-prompt://oauth?code=..."
    // 4. Intercept code, đóng WebView window
    // 5. Gọi Go engine: provider.oauth_finish {provider, code}
    // 6. Go engine exchange code → access_token + refresh_token → lưu vào provider_tokens
}
```

Tauri WebView intercept URL scheme `open-prompt://` — không cần mở browser ngoài.

### 6.2 Browser + Localhost Callback (fallback)

Khi provider không hỗ trợ custom URL scheme (redirect URI phải là `http://`):

1. Go engine start HTTP server tạm `localhost:random_port/callback`
2. Tauri mở URL login trong browser hệ thống
3. User login, provider redirect về `http://localhost:PORT/callback?code=xxx`
4. Go engine nhận code, exchange token, đóng HTTP server
5. Emit Tauri event `oauth-complete` về React

### 6.3 GitHub Copilot — Device Flow

GitHub Copilot dùng OAuth Device Flow (không cần redirect URI):

1. Go engine request device code từ `github.com/login/device/code`
2. Hiện cho user: "Mở github.com/login/device, nhập code: **ABCD-1234**"
3. Poll `github.com/login/oauth/access_token` mỗi 5 giây
4. Khi user approve → nhận token → exchange sang Copilot session token

---

## 7. @Mention Routing

### 7.1 Parsing

**File:** `go-engine/api/mention.go`

```go
// ParseMention tách alias và prompt sạch
// "@claude viết email" → ("claude", "viết email")
// "viết email @gpt4"   → ("gpt4", "viết email")
// "viết email"         → ("", "viết email") → dùng Default()
func ParseMention(prompt string) (alias, cleanPrompt string)
```

Alias không phân biệt hoa thường. Strip ký tự `@` trước khi Route.

### 7.2 UI Hint

Khi user gõ `@` trong `CommandInput.tsx` → hiện dropdown nhỏ gợi ý providers đang active (lấy từ `provider.list` RPC call khi overlay mở).

### 7.3 Ctrl+M Quick-Switch

`CommandInput.tsx` xử lý `Ctrl+M` → mở `ModelPicker.tsx`:

```
┌─────────────────────────────────┐
│ Chọn model              [ESC]   │
│ ● Claude 3.5 Sonnet  [default] │
│   GPT-4o                        │
│   Gemini 1.5 Pro                │
│   Ollama (llama3.2)             │
└─────────────────────────────────┘
```

Selection lưu vào `overlayStore.activeProvider` (per-session, reset khi đóng overlay).

---

## 8. Interactive Fallback

Khi provider thất bại, Go engine gửi `stream.chunk` cuối với metadata fallback:

```json
{
  "method": "stream.chunk",
  "params": {
    "delta": "",
    "done": true,
    "error": "rate_limit",
    "error_message": "Claude: 429 Too Many Requests",
    "fallback_providers": ["gpt-4o", "gemini-pro"]
  }
}
```

React `ResponsePanel.tsx` detect `error + fallback_providers` → hiện `FallbackDialog.tsx`:

```
┌────────────────────────────────────────┐
│ ⚠ Claude gặp lỗi: rate limit           │
│                                        │
│ Thử lại với:                           │
│  [GPT-4o]  [Gemini Pro]  [Hủy]        │
└────────────────────────────────────────┘
```

User chọn → `query.stream` gửi lại với `provider: "gpt-4o"` và prompt gốc — không cần gõ lại.

---

## 9. DB Schema bổ sung

```sql
-- provider_tokens đã có từ Phase 1, thêm cột aliases
ALTER TABLE provider_tokens ADD COLUMN aliases TEXT DEFAULT '[]'; -- JSON array

-- Custom alias mapping per user
CREATE TABLE IF NOT EXISTS model_aliases (
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alias       TEXT    NOT NULL, -- "@mylocal"
    provider_id TEXT    NOT NULL, -- "my-ollama"
    PRIMARY KEY (user_id, alias)
);

-- Custom gateways
CREATE TABLE IF NOT EXISTS custom_gateways (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT    NOT NULL, -- "my-ollama"
    display_name TEXT   NOT NULL,
    base_url    TEXT    NOT NULL,
    api_key     TEXT    DEFAULT '',
    default_model TEXT  DEFAULT '',
    aliases     TEXT    DEFAULT '[]', -- JSON array
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 10. JSON-RPC Methods mới

| Method | Params | Response |
|---|---|---|
| `provider.list` | `{token}` | `[{name, display_name, aliases, auth_type, valid, source}]` |
| `provider.detect` | `{token}` | `[DetectedProvider]` — chạy lại auto-detect |
| `provider.add_gateway` | `{token, name, base_url, api_key, aliases}` | `{ok}` |
| `provider.remove` | `{token, name}` | `{ok}` |
| `provider.set_default` | `{token, name}` | `{ok}` |
| `provider.validate` | `{token, name}` | `{valid, latency_ms, models}` |
| `provider.oauth_start` | `{token, provider}` | `{url, method: "webview"\|"browser"\|"device_flow", device_code?}` |
| `provider.oauth_finish` | `{token, provider, code}` | `{ok}` |
| `provider.oauth_poll` | `{token, provider, device_code}` | `{done, token?}` — Device Flow polling |
| `query.stream` | `{token, prompt, provider?, system?, max_tokens?}` | stream notifications |

`query.stream` thêm optional field `provider` — nếu có thì override alias từ `@mention`.

---

## 11. File Map

### Go Engine (mới + sửa)

```
go-engine/model/providers/
├── interface.go        ← NEW: Provider interface, AuthType
├── registry.go         ← NEW: ProviderRegistry
├── anthropic.go        ← REFACTOR: implement Provider interface
├── openai.go           ← NEW: OpenAI + Azure OpenAI
├── gemini.go           ← NEW: Google Gemini (OAuth + API key)
├── copilot.go          ← NEW: GitHub Copilot (Device Flow)
├── gateway.go          ← NEW: Generic OpenAI-compat gateway
└── detector/
    ├── detector.go     ← NEW: RunAll() song song
    ├── env.go          ← NEW: env var scanner
    ← NEW: CLI scanner
    ├── configfile.go   ← NEW: config file scanner
    └── localport.go    ← NEW: TCP port scanner

go-engine/api/
├── mention.go          ← NEW: ParseMention()
├── handlers_query.go   ← MODIFY: dùng Registry, parseMention, fallback metadata
├── handlers_provider.go ← NEW: provider.list/detect/add_gateway/remove/set_default/validate/oauth_*
└── router.go           ← MODIFY: thêm provider.* routes
```

### Tauri (mới)

```
src-tauri/src/
└── oauth.rs            ← NEW: start_oauth command (WebView + browser fallback)
```

### React (mới + sửa)

```
src/components/
├── overlay/
│   ├── CommandInput.tsx    ← MODIFY: Ctrl+M, @ dropdown hint
│   ├── ModelPicker.tsx     ← NEW: quick-switch model list
│   ├── ResponsePanel.tsx   ← MODIFY: detect fallback_providers
│   └── FallbackDialog.tsx  ← NEW: interactive fallback UI
└── settings/
    ├── ProvidersPanel.tsx  ← NEW: danh sách providers + status
    └── GatewayForm.tsx     ← NEW: thêm custom gateway
```

---

## 12. Testing Strategy

- **Unit tests Go:** Mỗi provider có test mock HTTP server (không cần key thật)
- **Unit tests detector:** Mock env, mock CLI output, mock TCP server
- **Unit tests registry:** Route, Default, FallbackCandidates
- **Unit tests mention:** ParseMention với các edge cases
- **Integration test:** `TestProviderRegistry_Anthropic` dùng `ANTHROPIC_API_KEY` nếu có, skip nếu không
- **React:** Không có unit test UI (Phase 1 precedent), nhưng TypeScript type safety

---

## 13. Out of Scope (Phase 2B, 2C)

- Prompt Library + History → Phase 2B
- Text injection → Phase 2C
- Keychain OS storage (lưu token) → Phase 2B (cùng security improvements)
- Claude OAuth (Anthropic chưa release public endpoint) → placeholder
- Mobile companion → Phase 3

---

## 14. Định nghĩa Hoàn thành

Phase 2A hoàn thành khi:
1. `@claude`, `@gpt4`, `@gemini` routing hoạt động trong overlay
2. Auto-detect tìm được ít nhất 1 provider từ env/CLI/config trên máy dev
3. Thêm Ollama gateway qua Settings UI và chat được
4. `Ctrl+M` mở model picker, switch provider, query thành công
5. Khi provider trả lỗi rate_limit, FallbackDialog hiện và retry thành công
6. All Go unit tests pass
