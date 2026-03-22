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

**Startup flow:** Tauri spawn Go sidecar → Go khởi tạo SQLite + migrations → Go listen socket → Tauri ready → nhận hotkey.

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
- Public repo chứa: compiled binaries, CHANGELOG.md, README.md, release notes
- Quy trình: tag version trên private → GitHub Actions build 3 platform → upload binaries lên public repo release

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
CREATE TABLE skills (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    prompt_id   INTEGER REFERENCES prompts(id),
    model       TEXT,
    provider    TEXT,
    config_json TEXT,                     -- JSON: temperature, max_tokens, etc.
    tags        TEXT,                     -- JSON array
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

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

### Auto-Detection Sources (theo thứ tự ưu tiên)

1. **Environment Variables** — `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GOOGLE_API_KEY`, `GITHUB_TOKEN`
2. **Config Files**
   - `~/.claude/claude.json` hoặc `~/.claude.json` → Claude CLI
   - `%APPDATA%/Claude/` (Windows) / `~/.config/Claude/` (Linux) → Claude Desktop
   - `~/.config/gh/hosts.yml` → GitHub CLI → Copilot token
   - `~/.config/gcloud/credentials.db` → Gemini via gcloud
   - `~/.gemini/` → Gemini CLI
   - `~/.vscode/extensions/github.copilot-*/` → VS Code Copilot
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

- Background goroutine scan expiry mỗi 30 phút
- File watcher (inotify/FSEvents/ReadDirectoryChangesW) theo dõi config changes
- Auto-refresh trước khi hết hạn 24h
- Notification UI khi token còn < 24h
- Badge đỏ trên system tray khi có provider expired

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

- `sequential` — thử lần lượt theo priority
- `latency-based` — chọn provider có latency thấp nhất gần đây
- `cost-based` — chọn provider rẻ nhất có sẵn

Mỗi lần fallback ghi lý do vào `history.fallback_from` để hiển thị UI.

---

## 9. Core Features

### 9.1 Global Hotkey

- Default: `Ctrl+Space`
- Configurable trong Settings
- Tauri v2: `tauri-plugin-global-shortcut`
- Platform-specific: Win32 RegisterHotKey / CGEventTap (macOS) / X11 (Linux)

### 9.2 Overlay UI

- Always-on-top window, centered
- Auto-focus khi mở
- Đóng: Escape / click ngoài
- Thu nhỏ về system tray
- Animation: slide-down + fade (60fps)
- Keyboard-first navigation

### 9.3 Slash Command System

```
/ → hiển thị danh sách command
/em → fuzzy filter "email"

Command structure:
{
  name: "email",
  description: "Viết email chuyên nghiệp",
  category: "business",
  prompt: "...",
  model: "claude-3.5-sonnet",
  provider: "anthropic"
}
```

- Fuzzy search (fuse.js)
- Keyboard navigation (↑↓ Enter)
- Nhóm theo category
- Context-aware: ưu tiên command phù hợp app đang dùng

### 9.4 Input Injection

1. Copy response vào clipboard
2. Simulate `Ctrl+V` paste vào app đang focus
3. Fallback: simulated typing (enigo) nếu paste không được

Vietnamese UTF-8 compatible, tương thích UniKey.

### 9.5 Context Awareness

Detect active application (process name + window title):
- Dùng để ưu tiên slash command
- Filter prompt theo app context
- Platform: GetForegroundWindow (Win) / CGWindowListCopy (macOS) / _NET_ACTIVE_WINDOW (Linux)

### 9.6 Multi-user Auth

- Local login với bcrypt (cost factor 12)
- Session token (JWT, expire 24h, lưu memory)
- Mỗi user có prompt/skill/settings riêng
- Avatar color để phân biệt user

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
- Daily aggregation job (chạy midnight) → `usage_daily` table
- Không gửi data ra ngoài (local-first)

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
- SQLite file: permission 600 (chỉ owner đọc được)
- Tauri capabilities: giới hạn quyền WebView tối thiểu
- Không log tokens, keys trong bất kỳ log file nào

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
    "content": "Dịch nội dung sau sang {{lang}}: {{input}}",
    "model": "gemini-1.5-pro"
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
