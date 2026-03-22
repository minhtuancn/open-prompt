# Open Prompt — Design Specification

**Date:** 2026-03-22
**Status:** Approved
**Repo (private):** `minhtuancn/open-prompt`
**Repo (public/release):** `minhtuancn/open-prompt-release`

---

## 1. Overview

Open Prompt là một desktop AI assistant chạy trên Windows, Linux và macOS, hoạt động như một "system-wide AI copilot". Người dùng gọi ứng dụng bằng global hotkey, nhập query hoặc slash command trong overlay UI nổi, nhận response từ AI và insert vào bất kỳ ô input nào trên hệ thống.

### Goals

- System-wide AI assistant với hotkey kích hoạt tức thì
- Hỗ trợ nhiều AI provider với smart fallback tự động
- Quản lý prompt, skill, project cho multi-user trên cùng máy
- Auto-detect provider đã đăng nhập (Claude CLI, Copilot, Gemini, v.v.)
- Usage analytics theo provider và model
- Production-ready: cross-platform, secure, auto-update

---

## 2. Architecture

### Pattern: Tauri v2 (Rust) + Go Engine (sidecar)

```
┌─────────────────────────────────────────────────────┐
│                   USER MACHINE                      │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │           TAURI v2 (Rust core)               │   │
│  │  • Global hotkey listener                    │   │
│  │  • System tray                               │   │
│  │  • Overlay window management                 │   │
│  │  • Input injection (enigo crate)             │   │
│  │  • Auto-updater (tauri-plugin-updater)       │   │
│  │  • IPC bridge → Go Engine                    │   │
│  │                                              │   │
│  │  ┌────────────────────────────────────────┐  │   │
│  │  │     React + TailwindCSS (WebView)      │  │   │
│  │  │  • Overlay UI                          │  │   │
│  │  │  • Slash command palette               │  │   │
│  │  │  • Prompt/Skill manager                │  │   │
│  │  │  • Settings + Analytics panel          │  │   │
│  │  └────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────┘   │
│              ↕ Unix Socket / Named Pipe              │
│  ┌──────────────────────────────────────────────┐   │
│  │              GO ENGINE (sidecar)             │   │
│  │  • Prompt builder & Skill engine             │   │
│  │  • Model router + priority fallback chain    │   │
│  │  • Provider auto-detector                    │   │
│  │  • OAuth flow (GitHub/Google)                │   │
│  │  • Token expiry watcher                      │   │
│  │  • SQLite (users, prompts, history)          │   │
│  │  • Auth (bcrypt, multi-user)                 │   │
│  │  • Usage analytics aggregator               │   │
│  │  • Context detector (active app)             │   │
│  │  • i18n data provider (7 ngôn ngữ)          │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
                        ↕ HTTPS
          ┌─────────────────────────────┐
          │  External AI Providers      │
          │  Claude / Gemini / Copilot  │
          │  OpenAI / OpenRouter /      │
          │  LiteLLM / Ollama (local)   │
          └─────────────────────────────┘
```

**Giao tiếp Rust ↔ Go:** Unix socket (Linux/macOS), Named Pipe (Windows). Go Engine expose JSON-RPC 2.0 API. Không expose ra ngoài máy.

**Socket authentication:** Khi spawn Go sidecar, Tauri truyền một `shared_secret` (32-byte random) qua environment variable `OP_SOCKET_SECRET`. Mỗi JSON-RPC request phải có header `X-Secret: <shared_secret>`. Go Engine từ chối request không có header đúng. Chống process khác trên máy connect vào socket.

**Startup flow:** Tauri spawn Go sidecar (với `OP_SOCKET_SECRET`) → Go khởi tạo SQLite + migrations → Go listen socket → Go gửi `ready` signal qua stdout → Tauri ready → nhận hotkey.

### JSON-RPC 2.0 API Contract

**Streaming:** Go Engine dùng JSON-RPC notifications (`method: "stream.chunk"`) để push từng token. Rust nhận qua socket read loop và forward lên React qua Tauri event.

**Core methods:**

```jsonc
// Auth
{ "method": "auth.login",    "params": { "username": string, "password": string } }
// → { "token": string, "user": User }

{ "method": "auth.logout",   "params": { "token": string } }
{ "method": "auth.me",       "params": { "token": string } }

// Query (streaming)
{ "method": "query.stream",  "params": { "token": string, "input": string, "skill_id"?: int, "slash_name"?: string } }
// → notifications: { "method": "stream.chunk",  "params": { "delta": string, "done": false } }
// → notification:  { "method": "stream.chunk",  "params": { "delta": "",     "done": true, "usage": Usage } }

// Query (non-streaming)
{ "method": "query.run",     "params": { "token": string, "input": string, "skill_id"?: int } }
// → { "response": string, "usage": Usage }

// Providers
{ "method": "providers.list",   "params": { "token": string } }
{ "method": "providers.detect", "params": { "token": string } }
{ "method": "providers.connect_oauth", "params": { "token": string, "provider_id": string } }

// Prompts
{ "method": "prompts.list",   "params": { "token": string, "search"?: string, "category"?: string } }
{ "method": "prompts.create", "params": { "token": string, "prompt": PromptInput } }
{ "method": "prompts.update", "params": { "token": string, "id": int, "prompt": PromptInput } }
{ "method": "prompts.delete", "params": { "token": string, "id": int } }

// Skills
{ "method": "skills.list",   "params": { "token": string } }
{ "method": "skills.create", "params": { "token": string, "skill": SkillInput } }
{ "method": "skills.update", "params": { "token": string, "id": int, "skill": SkillInput } }
{ "method": "skills.delete", "params": { "token": string, "id": int } }

// Slash commands
{ "method": "commands.list",    "params": { "token": string, "query"?: string } }
{ "method": "commands.resolve", "params": { "token": string, "slash_name": string } }

// Settings
{ "method": "settings.get", "params": { "token": string, "key": string } }
{ "method": "settings.set", "params": { "token": string, "key": string, "value": any } }

// Analytics
{ "method": "analytics.summary", "params": { "token": string, "period": "7d"|"30d"|"90d" } }
{ "method": "analytics.by_provider", "params": { "token": string, "period": string } }
{ "method": "analytics.token_expiry","params": { "token": string } }

// i18n
{ "method": "i18n.locale", "params": { "lang": string } }
// → { "messages": Record<string, string> }
```

**Error format:**
```jsonc
{ "error": { "code": -32001, "message": "unauthorized" } }
{ "error": { "code": -32002, "message": "provider_not_found" } }
{ "error": { "code": -32003, "message": "all_providers_failed", "data": { "attempts": [...] } } }
```

**Startup flow:** Tauri spawn Go sidecar (với `OP_SOCKET_SECRET`) → Go khởi tạo SQLite + migrations → Go listen socket → Go gửi `ready` signal qua stdout → Tauri ready → nhận hotkey.

---

## 3. Platform Support

| Platform | Hotkey | Input Injection | System Tray | Keychain |
|----------|--------|----------------|-------------|----------|
| Windows | Win32 RegisterHotKey | SendInput / Clipboard | WinAPI | Windows Credential Manager |
| Linux | X11/Wayland libxdo | xdotool / Clipboard | libappindicator | libsecret / KWallet |
| macOS | CGEventTap | AXUIElement (Accessibility) | NSStatusBar | macOS Keychain |

CI/CD build tất cả 3 platform song song từ ngày đầu.

---

## 4. Release Workflow

- `minhtuancn/open-prompt` — private repo, chứa toàn bộ source code
- `minhtuancn/open-prompt-release` — public repo, là git submodule trong private repo
- Submodule path: `open-prompt/open-prompt-release/`
- Public repo chứa: CHANGELOG.md, README.md, release notes (không track binary trong git)
- **Binaries:** upload lên GitHub Releases assets (không commit vào git repo)
- Quy trình: tag `vX.Y.Z` trên private → GitHub Actions build 3 platform → sign binaries → upload lên public repo GitHub Releases

**Code signing:**
- Windows: Authenticode signature (self-signed cho dev, certificate cho production)
- macOS: Apple Developer ID + notarization (bắt buộc từ macOS 10.15+)
- Linux: GPG signature của tarball

**Tauri updater config:**
```json
{
  "updater": {
    "endpoints": ["https://github.com/minhtuancn/open-prompt-release/releases/latest/download/update-manifest.json"],
    "pubkey": "<ed25519 public key>"
  }
}
```

---

## 5. Project Structure

```
open-prompt/                          ← private repo (source)
├── .github/
│   └── workflows/
│       ├── build.yml                 ← CI: build Win+Linux+macOS
│       ├── release.yml               ← CD: tag → GitHub Release
│       └── test.yml                  ← unit + integration tests
│
├── open-prompt-release/              ← git submodule (public repo)
│   ├── CHANGELOG.md
│   ├── README.md
│   └── releases/
│
├── src-tauri/                        ← Tauri v2 (Rust)
│   ├── Cargo.toml
│   ├── tauri.conf.json
│   ├── capabilities/
│   └── src/
│       ├── main.rs
│       ├── lib.rs
│       ├── hotkey.rs
│       ├── tray.rs
│       ├── window.rs
│       ├── injection.rs              ← enigo crate
│       ├── sidecar.rs                ← spawn + manage Go engine
│       └── ipc.rs                    ← Tauri ↔ Go bridge
│
├── src/                              ← React frontend
│   ├── main.tsx
│   ├── App.tsx
│   ├── components/
│   │   ├── overlay/
│   │   │   ├── CommandInput.tsx
│   │   │   ├── SlashMenu.tsx
│   │   │   └── ResponsePanel.tsx
│   │   ├── settings/
│   │   │   ├── SettingsLayout.tsx
│   │   │   ├── ProvidersTab.tsx
│   │   │   ├── HotkeyTab.tsx
│   │   │   ├── AppearanceTab.tsx
│   │   │   └── LanguageTab.tsx
│   │   ├── prompts/
│   │   │   ├── PromptList.tsx
│   │   │   ├── PromptEditor.tsx
│   │   │   └── PromptFilter.tsx
│   │   ├── skills/
│   │   │   ├── SkillList.tsx
│   │   │   └── SkillEditor.tsx
│   │   ├── analytics/
│   │   │   └── UsageStats.tsx
│   │   └── shared/
│   │       ├── FuzzySearch.tsx
│   │       ├── KeyboardNav.tsx
│   │       └── Avatar.tsx
│   ├── hooks/
│   │   ├── useOverlay.ts
│   │   ├── useSlashCommand.ts
│   │   ├── useStream.ts
│   │   └── useI18n.ts
│   ├── i18n/
│   │   ├── index.ts
│   │   └── locales/
│   │       ├── en.json
│   │       ├── vi.json
│   │       ├── fr.json
│   │       ├── zh-CN.json
│   │       ├── th.json
│   │       ├── lo.json
│   │       └── ru.json
│   ├── store/
│   │   ├── overlayStore.ts
│   │   ├── authStore.ts
│   │   └── settingsStore.ts
│   └── styles/
│       └── globals.css
│
├── go-engine/
│   ├── main.go
│   ├── go.mod
│   ├── Makefile
│   ├── api/
│   │   ├── server.go                 ← JSON-RPC 2.0 over socket
│   │   ├── handlers.go
│   │   └── middleware.go
│   ├── auth/
│   │   ├── service.go                ← bcrypt, multi-user
│   │   ├── session.go
│   │   └── oauth/
│   │       ├── server.go             ← local callback HTTP server
│   │       ├── pkce.go               ← PKCE flow
│   │       ├── browser.go            ← open system browser
│   │       ├── github.go             ← Copilot Device Flow
│   │       ├── google.go             ← Gemini OAuth 2.0
│   │       └── anthropic.go          ← Claude API key flow
│   ├── provider/
│   │   ├── detector.go               ← auto-detect từ file/env/keychain/process
│   │   ├── registry.go               ← provider definitions
│   │   ├── keychain.go               ← system keychain cross-platform
│   │   ├── token_manager.go          ← refresh, validate, rotate
│   │   ├── watcher.go                ← file system watcher
│   │   └── health_checker.go         ← ping providers mỗi 5 phút
│   ├── model/
│   │   ├── router.go                 ← route request → provider
│   │   ├── fallback.go               ← priority chain + fallback logic
│   │   ├── stream.go                 ← SSE streaming
│   │   └── providers/
│   │       ├── litellm.go
│   │       ├── openrouter.go
│   │       ├── anthropic.go
│   │       ├── openai.go
│   │       ├── google.go
│   │       ├── copilot.go
│   │       ├── ollama.go             ← optional, pluggable
│   │       └── custom.go
│   ├── engine/
│   │   ├── prompt_builder.go
│   │   ├── skill_engine.go
│   │   ├── command_resolver.go
│   │   └── context_detector.go
│   ├── analytics/
│   │   ├── collector.go              ← ghi usage sau mỗi request
│   │   ├── aggregator.go             ← daily rollup
│   │   └── reporter.go               ← query stats cho UI
│   ├── db/
│   │   ├── sqlite.go
│   │   ├── migrations/
│   │   │   ├── 001_init.sql
│   │   │   ├── 002_skills.sql
│   │   │   └── 003_analytics.sql
│   │   └── repos/
│   │       ├── user_repo.go
│   │       ├── prompt_repo.go
│   │       ├── skill_repo.go
│   │       ├── project_repo.go
│   │       ├── history_repo.go
│   │       ├── analytics_repo.go         ← usage_daily CRUD + aggregation
│   │       └── settings_repo.go
│   └── config/
│       ├── settings.go
│       └── defaults.go
│
├── scripts/
│   ├── build-engine.sh
│   ├── dev.sh
│   └── release.sh
├── package.json
├── vite.config.ts
├── tailwind.config.ts
└── README.md
```

---

## 6. Database Schema (SQLite)

```sql
-- Users (multi-user local auth)
CREATE TABLE users (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    username     TEXT NOT NULL UNIQUE,
    display_name TEXT,
    password_hash TEXT NOT NULL,          -- bcrypt
    avatar_color TEXT DEFAULT '#6366f1',
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login   DATETIME
);

-- Projects
CREATE TABLE projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    name       TEXT NOT NULL,
    color      TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Prompts
CREATE TABLE prompts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id),
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    category   TEXT,
    tags       TEXT,                      -- JSON array
    is_slash   INTEGER DEFAULT 0,         -- 1 = slash command
    slash_name TEXT,                      -- e.g. "email"
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Skills
-- Skill có thể link đến prompt (prompt_id) HOẶC có inline prompt_text riêng.
-- Ưu tiên: nếu prompt_id != NULL thì dùng prompts.content; ngược lại dùng prompt_text.
CREATE TABLE skills (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    prompt_id   INTEGER REFERENCES prompts(id) ON DELETE SET NULL,
    prompt_text TEXT,                     -- inline prompt khi không link prompt
    model       TEXT,
    provider    TEXT,
    config_json TEXT,                     -- JSON (xem schema bên dưới)
    tags        TEXT,                     -- JSON array
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- config_json schema:
-- {
--   "temperature": 0.0-2.0 (default 0.7),
--   "max_tokens": integer (default 1000),
--   "top_p": 0.0-1.0 (default 1.0),
--   "system": string (system prompt override),
--   "timeout_ms": integer (default 30000),
--   "stream": boolean (default true)
-- }

-- Settings (per-user key-value)
CREATE TABLE settings (
    user_id INTEGER NOT NULL REFERENCES users(id),
    key     TEXT NOT NULL,
    value   TEXT,
    PRIMARY KEY (user_id, key)
);

-- History
CREATE TABLE history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER REFERENCES users(id),
    query         TEXT NOT NULL,
    response      TEXT,
    provider      TEXT,
    model         TEXT,
    input_tokens  INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    latency_ms    INTEGER DEFAULT 0,
    status        TEXT DEFAULT 'success', -- success | fallback | error
    fallback_from TEXT,                   -- original model nếu có fallback
    skill_id      INTEGER REFERENCES skills(id),
    timestamp     DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Provider tokens (reference to keychain)
CREATE TABLE provider_tokens (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    provider_id    TEXT NOT NULL,          -- e.g. "github_copilot", "gemini"
    auth_type      TEXT NOT NULL,          -- "oauth" | "api_key" | "auto_detected"
    keychain_key   TEXT NOT NULL,          -- key in system keychain
    expires_at     DATETIME,
    refresh_token_key TEXT,               -- keychain key for refresh token
    detected_at    DATETIME,
    last_refreshed DATETIME,
    is_active      INTEGER DEFAULT 1,
    UNIQUE(user_id, provider_id)
);

-- Model priority chain (per-user)
CREATE TABLE model_priority (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    priority   INTEGER NOT NULL,           -- 1 = highest
    provider   TEXT NOT NULL,
    model      TEXT NOT NULL,
    is_enabled INTEGER DEFAULT 1,
    UNIQUE(user_id, priority)
);

-- Usage analytics (daily aggregate)
CREATE TABLE usage_daily (
    date           TEXT NOT NULL,
    user_id        INTEGER NOT NULL REFERENCES users(id),
    provider       TEXT NOT NULL,
    model          TEXT NOT NULL,
    requests       INTEGER DEFAULT 0,
    input_tokens   INTEGER DEFAULT 0,
    output_tokens  INTEGER DEFAULT 0,
    errors         INTEGER DEFAULT 0,
    fallbacks      INTEGER DEFAULT 0,
    avg_latency_ms INTEGER DEFAULT 0,
    PRIMARY KEY (date, user_id, provider, model)
);
```

---

## 7. Provider System

### Consent Flow (First-time scan)

Lần đầu chạy app, **sau khi tạo tài khoản**, hiển thị dialog:
> "Open Prompt muốn tìm kiếm các AI provider đã được cài đặt trên máy (Claude CLI, GitHub Copilot, Gemini...) để tái sử dụng token có sẵn. Chúng tôi sẽ quét các file config trong home directory của bạn. Không có dữ liệu nào được gửi ra ngoài."
> [Cho phép] [Bỏ qua]

Nếu user chọn "Bỏ qua": không scan, chỉ detect từ environment variables.
Setting `provider.auto_detect_enabled` (true/false) có thể thay đổi sau trong Settings.

### Tần suất scan

- **Startup:** scan đầy đủ khi app khởi động
- **File watcher:** real-time theo dõi các file config đã biết (inotify/FSEvents)
- **Periodic:** re-scan mỗi 30 phút (background goroutine)
- **Manual:** nút "Scan now" trong Settings > Providers

### Auto-Detection Sources (theo thứ tự ưu tiên)

1. **Environment Variables** — `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GOOGLE_API_KEY`, `GITHUB_TOKEN`
2. **Config Files** (chỉ khi `auto_detect_enabled = true`)

   | File path | Token field | Provider |
   |-----------|------------|---------|
   | `~/.claude/claude.json` | `$.api_key` | Claude CLI |
   | `~/.claude.json` | `$.api_key` | Claude CLI (alt) |
   | `%APPDATA%/Claude/config.json` (Win) | `$.api_key` | Claude Desktop |
   | `~/.config/Claude/config.json` (Linux) | `$.api_key` | Claude Desktop |
   | `~/.config/gh/hosts.yml` | `github.com.oauth_token` | GitHub Copilot |
   | `~/.config/gcloud/application_default_credentials.json` | `$.access_token` | Gemini |
   | `~/.gemini/credentials.json` | `$.access_token` | Gemini CLI |
   | `~/.vscode/extensions/github.copilot-*/dist/` | parse từ extension storage | VS Code Copilot |

3. **System Keychain** — Windows Credential Manager / libsecret / macOS Keychain
4. **Running Processes** — detect `ollama`, `claude`, `gemini-cli` đang chạy

### Supported Providers

| Provider | Auto-detect | OAuth Browser | API Key | Notes |
|----------|------------|---------------|---------|-------|
| Claude (Anthropic) | ✅ file scan | ❌ (không có public OAuth) | ✅ | Fallback paste key |
| GitHub Copilot | ✅ gh CLI + keychain | ✅ Device Flow | ✅ | |
| Gemini (Google) | ✅ gcloud + env | ✅ PKCE OAuth 2.0 | ✅ | |
| ChatGPT (OpenAI) | ✅ env var | ❌ | ✅ | |
| OpenRouter | ❌ | ❌ | ✅ | |
| LiteLLM | ❌ | ❌ | ✅ URL+key | |
| Azure OpenAI | ❌ | ❌ | ✅ endpoint+key | |
| Ollama | ✅ process detect | ❌ | ❌ | Optional, pluggable |
| Custom | ❌ | ❌ | ✅ URL+key+format | OpenAI-compat |

### OAuth Flow (PKCE)

```
1. Go Engine tạo code_verifier + code_challenge (SHA256)
2. Mở system browser → Provider OAuth URL
3. Go Engine khởi động local HTTP server: 127.0.0.1:RANDOM_PORT/callback
4. User login → Provider redirect về localhost
5. Go Engine nhận auth_code
6. Exchange auth_code + code_verifier → access_token + refresh_token
7. Lưu tokens vào system keychain (không bao giờ SQLite plain text)
8. Server tắt sau callback hoặc timeout 60s
```

Security: chỉ lắng nghe `127.0.0.1`, state parameter random (CSRF protection).

### Token Expiry & Auto-refresh

**Per-provider refresh threshold:**

| Provider | Token TTL | Refresh khi còn | Retry nếu fail |
|----------|-----------|----------------|---------------|
| GitHub Copilot (Device Flow) | ~8 giờ | 1 giờ | 3 lần, backoff 5/15/60 phút |
| Gemini access_token | 1 giờ | 10 phút | 3 lần, backoff 1/5/15 phút |
| Gemini refresh_token | Không hết hạn | N/A | Re-OAuth nếu revoked |
| Claude API key | Không hết hạn | N/A | Alert nếu 401 |

**Token revocation detection:**
- Nếu API call trả về 401/403 → mark token invalid → notification "Re-login required"
- Không tự động re-OAuth (tránh pop browser không mong muốn)

**Background goroutine:** check mỗi 30 phút; file watcher cho real-time changes.
**Notification UI:** khi token còn trong ngưỡng refresh.
**Badge đỏ** trên system tray khi có provider expired hoặc refresh fail.

---

## 8. Model Router & Fallback

### Priority Chain

User config kéo thả thứ tự ưu tiên:

```
Priority 1: Claude 3.5 Sonnet    ← thử trước
Priority 2: Gemini 1.5 Pro       ← fallback nếu Priority 1 lỗi
Priority 3: GPT-4o               ← fallback cấp 2
Priority 4: Ollama llama3        ← offline fallback cuối
```

### Fallback Triggers

- HTTP 429 (rate limit)
- HTTP 5xx (server error)
- Timeout vượt ngưỡng user config (default: 30s)
- Token hết hạn + refresh thất bại
- Provider unreachable

### Fallback Strategies

**User chọn strategy trong Settings > Model Router:**

| Strategy | Mô tả | Data source |
|----------|-------|-------------|
| `sequential` | Thử lần lượt theo priority order | Không cần data |
| `latency-based` | Chọn provider có p95 latency thấp nhất trong 24h qua | `history` table: avg(latency_ms) per model |
| `cost-based` | Chọn provider rẻ nhất hiện có | Hardcoded cost table ($/1K tokens), user có thể override |

**Cost table (hardcoded, updatable qua app update):**
```json
{
  "claude-3.5-sonnet": { "input": 0.003, "output": 0.015 },
  "gemini-1.5-pro":    { "input": 0.00125, "output": 0.005 },
  "gpt-4o":            { "input": 0.005, "output": 0.015 },
  "ollama/*":          { "input": 0.0, "output": 0.0 }
}
```

**Khi toàn bộ chain exhausted (tất cả providers fail):**
1. Trả về lỗi có cấu trúc: `{ "error": "all_providers_failed", "attempts": [...] }`
2. UI hiển thị thông báo với danh sách lý do từng provider
3. Nút "Retry" và "Try offline (Ollama)" nếu Ollama available
4. Không có silent fail — luôn thông báo user

**Reorder priority — xử lý SQLite UNIQUE constraint:**
Swap priority dùng temp value âm trong single transaction:
```sql
BEGIN;
UPDATE model_priority SET priority = -1 WHERE user_id=? AND priority=1;
UPDATE model_priority SET priority = 1  WHERE user_id=? AND priority=2;
UPDATE model_priority SET priority = 2  WHERE user_id=? AND priority=-1;
COMMIT;
```

Mỗi lần fallback ghi lý do vào `history.fallback_from` để hiển thị UI.

---

## 9. Core Features

### 9.1 Global Hotkey

- Default: `Ctrl+Space`
- Configurable trong Settings
- Tauri v2: `tauri-plugin-global-shortcut`
- Platform-specific: Win32 RegisterHotKey / CGEventTap (macOS) / X11 (Linux)

### 9.2 Overlay UI

- **Vị trí:** Center của **active monitor** (monitor chứa cursor khi hotkey được nhấn)
- **Kích thước:** Fixed width 640px, height auto (min 64px, max 480px tự expand khi response dài)
- Always-on-top window
- Auto-focus khi mở
- **Đóng:** Escape / click ngoài overlay
- **Khi đang stream response + click ngoài:** cancel stream → đóng overlay (không chờ xong)
- **Response overflow:** overlay expand tới max-height 480px, sau đó scroll bên trong
- Thu nhỏ về system tray (click X hoặc phím tắt)
- Animation: slide-down + fade (60fps target)
- Keyboard-first navigation

### 9.3 Slash Command System

```
/ → hiển thị danh sách command
/em → fuzzy filter "email"
```

**slash_name validation:**
- Chỉ chứa: `a-z`, `0-9`, `-`, `_` (lowercase, no spaces)
- Độ dài: 1–32 ký tự
- Unique per user
- Case-insensitive khi match: `/Email` = `/email`

**Template Engine:**

Dùng Go `text/template` syntax (đơn giản, built-in):
- `{{.input}}` — nội dung user nhập sau slash command
- `{{.lang}}` — variable tuỳ chỉnh
- `{{.context.app}}` — tên app đang active
- Escape literal `{{`: dùng `{{"{{"}}`

**Multi-variable UI flow:**

Khi prompt template có variables ngoài `{{.input}}`:
1. User gõ `/translate Hello world`
2. System detect template có `{{.lang}}`
3. Hiển thị mini form bên dưới CommandInput:
   ```
   ┌─────────────────────────┐
   │ /translate              │
   │ Input: Hello world      │
   │ lang: [____________]    │
   │              [Run →]    │
   └─────────────────────────┘
   ```
4. User điền `lang = "Vietnamese"` → Enter → run

Variables tự động extract từ template bằng regex `\{\{\.(\w+)\}\}` (bỏ qua `input` và `context.*`).

**Command structure:**
```json
{
  "slash_name": "email",
  "title": "Viết Email",
  "description": "Viết email chuyên nghiệp",
  "category": "business",
  "content": "Bạn là chuyên gia... {{.input}}",
  "model": "claude-3.5-sonnet",
  "provider": "anthropic"
}
```

- Fuzzy search (fuse.js) trên `slash_name` + `title` + `description`
- Keyboard navigation (↑↓ Enter)
- Nhóm theo category
- Context-aware: ưu tiên command phù hợp app đang dùng (mapping trong Settings)

### 9.4 Input Injection

**Flow chi tiết:**

1. Backup nội dung clipboard hiện tại (lưu vào memory)
2. Copy response vào clipboard
3. Simulate `Ctrl+V` paste vào app đang focus
4. Chờ 200ms sau paste
5. Restore clipboard về nội dung gốc (bước 1)

**Fallback sang simulated typing:**
- Trigger khi: Ctrl+V không có phản hồi sau 500ms (text không xuất hiện trong target app)
- Trigger khi: app đích là terminal emulator (detect process name: `wt.exe`, `alacritty`, `kitty`, `gnome-terminal`, `iTerm2`)
- Simulated typing: enigo crate, gõ từng ký tự UTF-8
- Delay giữa các ký tự: 10ms (tránh miss keystrokes)

**Không inject khi:**
- Active window là chính Open Prompt
- Active window là game full-screen (detect bằng `HWND_TOPMOST` + full resolution match)

**Vietnamese/UniKey:** Inject raw Unicode characters, không dùng VNI/TELEX keystrokes. Compatible với UniKey vì UniKey xử lý ở input method layer, inject trực tiếp Unicode bypass IME không conflict.

**Platform specifics:**
- Windows: `SendInput` với `KEYEVENTF_UNICODE`
- macOS: `AXUIElementSetAttributeValue` với `kAXValueAttribute`; cần Accessibility permission từ System Preferences
- Linux: `xdotool type --clearmodifiers` (X11); Wayland: `wtype` hoặc `ydotool` (requires uinput group membership)

### 9.5 Context Awareness

Detect active application (process name + window title):
- Dùng để ưu tiên slash command
- Filter prompt theo app context
- Platform: GetForegroundWindow (Win) / CGWindowListCopy (macOS) / _NET_ACTIVE_WINDOW (Linux)

### 9.6 Multi-user Auth

- Local login với bcrypt (cost factor 12)
- Session token (JWT, expire 7 ngày, **lưu encrypted trong file** `~/.open-prompt/session.dat`, encrypt bằng AES-256-GCM với key từ OS keychain)
- Session tự động load khi app khởi động — user không cần login lại sau restart
- Session expire sau 7 ngày không active, hoặc khi user logout thủ công
- Mỗi user có prompt/skill/settings riêng
- Avatar color để phân biệt user

**First-run onboarding flow:**
1. App khởi động lần đầu → không có user trong DB
2. Hiển thị màn hình "Create your account" (không thể bỏ qua)
3. User nhập username + password (min 8 ký tự)
4. Tạo user đầu tiên → tự động login → vào màn hình chính
5. Sau đó có thể thêm user khác trong Settings > Users

**Multi-user switch:** User switcher ở system tray menu; switch yêu cầu nhập password user đích.

### 9.7 Auto-update

- Tauri `tauri-plugin-updater`
- Check GitHub Releases API khi khởi động
- Download + verify signature
- Install khi restart hoặc ngay lập tức (user chọn)

---

## 10. Usage Analytics

### Statistics Page

- Tổng requests, tokens, fallback rate, avg latency
- Biểu đồ theo thời gian: 7d / 30d / 90d
- Breakdown theo provider và model
- Token expiry dashboard (tất cả providers)
- Fallback history (recent)

### Data Collection

- Ghi sau mỗi request: provider, model, tokens, latency, status
- Daily aggregation job: chạy khi app khởi động (catch-up nếu missed) và mỗi midnight theo **local timezone của máy**
- Không gửi data ra ngoài (local-first)

**Retention policy:**
- Raw `history`: giữ 90 ngày, sau đó xóa tự động
- `usage_daily` aggregate: giữ vĩnh viễn (dữ liệu nhỏ ~1KB/ngày)

---

## 11. i18n

7 ngôn ngữ hỗ trợ:

| Code | Ngôn ngữ |
|------|---------|
| `en` | English |
| `vi` | Tiếng Việt |
| `fr` | Français |
| `zh-CN` | 中文 (Giản thể) |
| `th` | ภาษาไทย |
| `lo` | ພາສາລາວ |
| `ru` | Русский |

Implementation: `react-i18next` trên frontend. Go Engine cung cấp locale data cho notification và system messages.

---

## 12. Security

- API keys và OAuth tokens: **chỉ lưu system keychain**, không bao giờ SQLite plain text
- Local callback OAuth server: chỉ lắng nghe `127.0.0.1`
- State parameter random cho CSRF protection
- Bcrypt cost factor 12 cho password hashing
- SQLite file permission:
  - Linux/macOS: `chmod 600` (owner read/write only)
  - Windows: ACL restrict về `SYSTEM` + current user only (SetFileSecurity)
- Socket authentication: shared secret qua env var `OP_SOCKET_SECRET` (xem Section 2)
- Tauri capabilities (allowlist tối thiểu): `shell.open`, `window.setAlwaysOnTop`, `globalShortcut.register`, `systemTray`, `updater`; **không** có `fs` hay `http` access từ WebView
- Không log tokens, keys, hoặc password hash trong bất kỳ log file nào
- Rate limit local JSON-RPC: max 100 requests/giây (chống abuse từ process khác dù đã có socket secret)

---

## 13. Sample Data

### Sample Slash Commands

```json
[
  {
    "slash_name": "email",
    "title": "Viết Email",
    "category": "business",
    "content": "Bạn là chuyên gia viết email chuyên nghiệp. Viết email dựa trên yêu cầu sau: {{input}}",
    "model": "claude-3.5-sonnet"
  },
  {
    "slash_name": "code",
    "title": "Code Assistant",
    "category": "dev",
    "content": "Bạn là senior developer. Giải quyết vấn đề code sau: {{input}}",
    "model": "claude-3.5-sonnet"
  },
  {
    "slash_name": "translate",
    "title": "Dịch thuật",
    "category": "language",
    "content": "Dịch nội dung sau sang {{.lang}}: {{.input}}",
    "model": "gemini-1.5-pro",
    "variables": ["lang"]
  }
]
```

### Sample Skill

```json
{
  "name": "write_email",
  "provider": "anthropic",
  "model": "claude-3.5-sonnet",
  "prompt": "Bạn là chuyên gia viết email chuyên nghiệp bằng tiếng Việt...",
  "config": {
    "temperature": 0.7,
    "max_tokens": 1000,
    "system": "Luôn viết lịch sự, chuyên nghiệp"
  },
  "tags": ["business", "email", "vietnamese"]
}
```

---

## 14. Open Questions / Future Work

- macOS Accessibility permission UX (cần hướng dẫn user cấp quyền)
- Wayland input injection (một số DE chặt hơn X11)
- Claude OAuth khi Anthropic release public endpoint
- Plugin system cho third-party skills
- Team sharing: sync prompts/skills qua cloud (v2)
- Mobile companion app (v3)
