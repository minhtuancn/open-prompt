# Open Prompt — Phase 1: Foundation & Core Loop

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Working prototype — user tạo account, nhập API key Claude, nhấn `Ctrl+Space`, overlay mở, gõ query, nhận streaming response.

**Architecture:** Tauri v2 (Rust) làm shell — quản lý hotkey, window, system tray, spawn Go sidecar. Go Engine là JSON-RPC 2.0 server qua Unix socket (Linux/macOS) hoặc Named Pipe (Windows). React + TailwindCSS chạy trong WebView, giao tiếp với Rust qua Tauri commands, Rust forward đến Go qua socket.

**Tech Stack:** Go 1.22+, Tauri v2, Rust stable, React 18, Vite, TailwindCSS v3, SQLite (modernc.org/sqlite — pure Go, no CGO), bcrypt, JWT (golang-jwt), zustand, @tauri-apps/api v2

---

## Spec Reference

`docs/superpowers/specs/2026-03-22-open-prompt-design.md`

---

## File Map

### Go Engine (`go-engine/`)
```
go-engine/
├── main.go                    ← entry point, startup, signal handling
├── go.mod
├── go.sum
├── Makefile                   ← build targets per platform
├── api/
│   ├── server.go              ← Unix socket / Named Pipe listener, JSON-RPC dispatch
│   ├── router.go              ← method → handler mapping
│   ├── middleware.go          ← secret validation, token auth
│   └── types.go               ← Request, Response, Error types
├── auth/
│   ├── service.go             ← Register, Login, ValidateToken
│   ├── session.go             ← JWT issue/validate, session file encrypt/decrypt
│   └── service_test.go
├── db/
│   ├── sqlite.go              ← Open connection, run migrations
│   ├── migrations/
│   │   └── 001_init.sql       ← tất cả tables từ spec
│   └── repos/
│       ├── user_repo.go
│       └── settings_repo.go
├── model/
│   ├── router.go              ← route request → provider
│   ├── stream.go              ← streaming chunks qua socket notifications
│   └── providers/
│       └── anthropic.go       ← Anthropic API client (streaming)
└── config/
    ├── settings.go            ← load/save key-value settings
    └── defaults.go
```

### Tauri / Rust (`src-tauri/`)
```
src-tauri/
├── Cargo.toml
├── tauri.conf.json
├── capabilities/
│   └── main.json              ← allowlist: globalShortcut, window, shell, tray
└── src/
    ├── main.rs                ← entry, setup plugins
    ├── lib.rs                 ← app builder, state
    ├── hotkey.rs              ← register/unregister global shortcut
    ├── tray.rs                ← system tray icon + menu
    ├── window.rs              ← overlay window create/show/hide, monitor detection
    ├── sidecar.rs             ← spawn Go engine, manage process, stdout "ready" signal
    └── ipc.rs                 ← socket client, forward JSON-RPC calls, stream events
```

### React Frontend (`src/`)
```
src/
├── main.tsx
├── App.tsx                    ← route: onboarding | login | overlay
├── components/
│   ├── onboarding/
│   │   └── CreateAccount.tsx  ← first-run: username + password form
│   ├── auth/
│   │   └── LoginScreen.tsx    ← login form
│   ├── overlay/
│   │   ├── CommandInput.tsx   ← main input textarea
│   │   └── ResponsePanel.tsx  ← streaming response display
│   └── settings/
│       └── ApiKeySetup.tsx    ← paste API key for Claude
├── store/
│   ├── authStore.ts           ← user session (Zustand)
│   └── overlayStore.ts        ← query state, streaming chunks
├── hooks/
│   └── useEngine.ts           ← wrapper for Tauri invoke() → Go RPC
└── styles/
    └── globals.css            ← Tailwind directives
```

---

## Task 1: Project Scaffold

**Files:**
- Create: `package.json`, `vite.config.ts`, `tailwind.config.ts`, `tsconfig.json`
- Create: `src/main.tsx`, `src/App.tsx`, `src/styles/globals.css`
- Create: `src-tauri/Cargo.toml`, `src-tauri/tauri.conf.json`
- Create: `src-tauri/capabilities/main.json`
- Create: `src-tauri/src/main.rs`, `src-tauri/src/lib.rs`

- [ ] **Step 1.1: Init Tauri v2 project**

```bash
cd /home/dev/open-prompt-code/open-prompt
npm create tauri-app@latest . -- --template react-ts --manager npm --force
```

Expected: tạo `package.json`, `src/`, `src-tauri/` với template react-ts.

- [ ] **Step 1.2: Cài Tauri v2 plugins cần thiết**

```bash
npm install
npm install @tauri-apps/api@^2
npm install @tauri-apps/plugin-global-shortcut @tauri-apps/plugin-shell
```

```bash
cd src-tauri
cargo add tauri-plugin-global-shortcut tauri-plugin-shell
cd ..
```

- [ ] **Step 1.3: Cài React dependencies**

```bash
npm install zustand fuse.js
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

- [ ] **Step 1.4: Cấu hình TailwindCSS**

Sửa `tailwind.config.ts`:
```ts
import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        surface: '#1a1a2e',
        accent: '#6366f1',
      },
    },
  },
  plugins: [],
} satisfies Config
```

Sửa `src/styles/globals.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

body {
  @apply bg-transparent m-0 p-0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}
```

- [ ] **Step 1.5: Cấu hình Tauri window cho overlay**

Sửa `src-tauri/tauri.conf.json`:
```json
{
  "productName": "open-prompt",
  "version": "0.1.0",
  "identifier": "com.minhtuancn.open-prompt",
  "build": {
    "frontendDist": "../dist",
    "devUrl": "http://localhost:1420",
    "beforeDevCommand": "npm run dev",
    "beforeBuildCommand": "npm run build"
  },
  "app": {
    "windows": [
      {
        "label": "overlay",
        "title": "Open Prompt",
        "width": 640,
        "height": 80,
        "minWidth": 640,
        "maxWidth": 640,
        "minHeight": 64,
        "maxHeight": 480,
        "resizable": false,
        "decorations": false,
        "transparent": true,
        "alwaysOnTop": true,
        "skipTaskbar": true,
        "visible": false,
        "center": true,
        "focus": true
      }
    ],
    "trayIcon": {
      "iconPath": "icons/icon.png",
      "iconAsTemplate": true
    }
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "icon": ["icons/icon.png"]
  }
}
```

- [ ] **Step 1.6: Tạo placeholder icon hợp lệ**

Tạo icon PNG tối thiểu hợp lệ bằng Python (available trên tất cả platforms):

```bash
mkdir -p src-tauri/icons

python3 - <<'EOF'
import struct, zlib

def make_minimal_png(width, height, color_rgb):
    """Tạo PNG tối thiểu với màu solid"""
    def png_chunk(name, data):
        c = struct.pack('>I', len(data)) + name + data
        return c + struct.pack('>I', zlib.crc32(c[4:]) & 0xffffffff)

    signature = b'\x89PNG\r\n\x1a\n'
    ihdr = png_chunk(b'IHDR', struct.pack('>IIBBBBB', width, height, 8, 2, 0, 0, 0))
    raw = b''.join(b'\x00' + bytes(color_rgb) * width for _ in range(height))
    idat = png_chunk(b'IDAT', zlib.compress(raw))
    iend = png_chunk(b'IEND', b'')
    return signature + ihdr + idat + iend

png = make_minimal_png(32, 32, [99, 102, 241])  # indigo #6366f1
with open('src-tauri/icons/icon.png', 'wb') as f:
    f.write(png)
print("Icon created: src-tauri/icons/icon.png")
EOF
```

Verify PNG hợp lệ:
```bash
python3 -c "
import struct
with open('src-tauri/icons/icon.png', 'rb') as f:
    sig = f.read(8)
assert sig == b'\x89PNG\r\n\x1a\n', 'Invalid PNG'
print('PNG valid')
"
```

Expected: `PNG valid`

- [ ] **Step 1.7: Verify scaffold build**

```bash
npm run dev &
sleep 5
kill %1
echo "Scaffold OK"
```

Expected: Vite khởi động không có lỗi.

- [ ] **Step 1.8: Commit**

```bash
git add -A
git commit -m "chore: khởi tạo project scaffold Tauri v2 + React + Tailwind"
```

---

## Task 2: Go Engine — Setup

**Files:**
- Create: `go-engine/go.mod`
- Create: `go-engine/main.go`
- Create: `go-engine/Makefile`
- Create: `go-engine/config/defaults.go`
- Create: `go-engine/api/types.go`

- [ ] **Step 2.1: Init Go module**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go mod init github.com/minhtuancn/open-prompt/go-engine
```

- [ ] **Step 2.2: Cài Go dependencies**

```bash
go get modernc.org/sqlite
go get golang.org/x/crypto/bcrypt
go get github.com/golang-jwt/jwt/v5
go get github.com/google/uuid
```

- [ ] **Step 2.3: Tạo JSON-RPC types**

Tạo `go-engine/api/types.go`:
```go
package api

// Request là cấu trúc JSON-RPC 2.0 request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

// Response là cấu trúc JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Notification là JSON-RPC 2.0 notification (không có id, dùng cho streaming)
type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// RPCError là error object trong JSON-RPC 2.0
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Predefined error codes
var (
	ErrUnauthorized      = &RPCError{Code: -32001, Message: "unauthorized"}
	ErrProviderNotFound  = &RPCError{Code: -32002, Message: "provider_not_found"}
	ErrAllProvidersFailed = &RPCError{Code: -32003, Message: "all_providers_failed"}
	ErrMethodNotFound    = &RPCError{Code: -32601, Message: "method_not_found"}
	ErrInvalidParams     = &RPCError{Code: -32602, Message: "invalid_params"}
	ErrInternal          = &RPCError{Code: -32603, Message: "internal_error"}
)

// NewResponse tạo success response
func NewResponse(id interface{}, result interface{}) Response {
	return Response{JSONRPC: "2.0", Result: result, ID: id}
}

// NewErrorResponse tạo error response
func NewErrorResponse(id interface{}, err *RPCError) Response {
	return Response{JSONRPC: "2.0", Error: err, ID: id}
}
```

- [ ] **Step 2.4: Tạo config defaults**

Tạo `go-engine/config/defaults.go`:
```go
package config

const (
	// DefaultTimeout là timeout mặc định cho AI requests (ms)
	DefaultTimeout = 30000

	// DefaultBcryptCost là cost factor cho bcrypt hashing
	DefaultBcryptCost = 12

	// DefaultJWTExpiry là thời gian expire của JWT session (ngày)
	DefaultJWTExpiry = 7

	// SocketEnvKey là env variable chứa shared secret
	SocketEnvKey = "OP_SOCKET_SECRET"

	// SocketPath là path của Unix socket (Linux/macOS)
	SocketPath = "/tmp/open-prompt.sock"

	// NamedPipeName là tên Named Pipe (Windows)
	NamedPipeName = `\\.\pipe\open-prompt`

	// DBFileName là tên file SQLite
	DBFileName = "open-prompt.db"

	// HistoryRetentionDays là số ngày giữ raw history
	HistoryRetentionDays = 90
)
```

- [ ] **Step 2.5: Tạo main.go**

Tạo `go-engine/main.go`:
```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/minhtuancn/open-prompt/go-engine/api"
	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db"
)

func main() {
	// Đọc shared secret từ env (bắt buộc)
	secret := os.Getenv(config.SocketEnvKey)
	if secret == "" {
		log.Fatal("OP_SOCKET_SECRET is required")
	}

	// Khởi tạo database
	database, err := db.Open()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Khởi động JSON-RPC server
	server, err := api.NewServer(secret, database)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	go func() {
		if err := server.Listen(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Thông báo Tauri rằng engine đã ready (qua stdout)
	fmt.Println("ready")

	// Chờ signal để graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	server.Close()
}
```

- [ ] **Step 2.6: Tạo Makefile**

Tạo `go-engine/Makefile`:
```makefile
.PHONY: build build-linux build-windows build-darwin test clean

# Build cho host platform
build:
	go build -o bin/go-engine .

# Cross-compile cho từng platform
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/go-engine-linux-amd64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/go-engine-windows-amd64.exe .

build-darwin:
	GOOS=darwin GOARCH=arm64 go build -o bin/go-engine-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/go-engine-darwin-amd64 .

test:
	go test ./... -v -race

clean:
	rm -rf bin/
```

- [ ] **Step 2.7: Verify Go setup**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go build ./... 2>&1
```

Expected: lỗi "package not found" cho api và db (chưa tạo) — OK, chỉ verify go.mod hợp lệ.

- [ ] **Step 2.8: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/
git commit -m "chore: khởi tạo Go engine module và cấu trúc cơ bản"
```

---

## Task 3: Database Layer

**Files:**
- Create: `go-engine/db/sqlite.go`
- Create: `go-engine/db/migrations/001_init.sql`
- Create: `go-engine/db/repos/user_repo.go`
- Create: `go-engine/db/repos/settings_repo.go`
- Test: `go-engine/db/sqlite_test.go`

- [ ] **Step 3.1: Viết failing test cho DB**

Tạo `go-engine/db/sqlite_test.go`:
```go
package db_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

func TestOpenAndMigrate(t *testing.T) {
	// Dùng in-memory SQLite cho test
	database, err := db.OpenInMemory()
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Verify tables tồn tại
	tables := []string{"users", "projects", "prompts", "skills", "settings", "history", "provider_tokens", "model_priority", "usage_daily"}
	for _, table := range tables {
		var count int
		err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("table %q not found after migration", table)
		}
	}
}
```

- [ ] **Step 3.2: Chạy test — verify fail**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./db/... -v -run TestOpenAndMigrate 2>&1
```

Expected: FAIL với "package db not found"

- [ ] **Step 3.3: Tạo migration SQL**

Tạo `go-engine/db/migrations/001_init.sql`:
```sql
-- Users
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    display_name  TEXT,
    password_hash TEXT    NOT NULL,
    avatar_color  TEXT    NOT NULL DEFAULT '#6366f1',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login    DATETIME
);

-- Projects
CREATE TABLE IF NOT EXISTS projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    color      TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Prompts
CREATE TABLE IF NOT EXISTS prompts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL,
    title      TEXT    NOT NULL,
    content    TEXT    NOT NULL,
    category   TEXT,
    tags       TEXT,           -- JSON array: ["tag1","tag2"]
    is_slash   INTEGER NOT NULL DEFAULT 0,
    slash_name TEXT,           -- lowercase, no spaces, 1-32 chars
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Skills
CREATE TABLE IF NOT EXISTS skills (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    prompt_id   INTEGER REFERENCES prompts(id) ON DELETE SET NULL,
    prompt_text TEXT,          -- inline prompt khi không link prompt
    model       TEXT,
    provider    TEXT,
    config_json TEXT,          -- JSON: {temperature, max_tokens, top_p, system, timeout_ms, stream}
    tags        TEXT,          -- JSON array
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Settings per-user
CREATE TABLE IF NOT EXISTS settings (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key     TEXT    NOT NULL,
    value   TEXT,
    PRIMARY KEY (user_id, key)
);

-- History
CREATE TABLE IF NOT EXISTS history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER REFERENCES users(id) ON DELETE SET NULL,
    query         TEXT    NOT NULL,
    response      TEXT,
    provider      TEXT,
    model         TEXT,
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    latency_ms    INTEGER NOT NULL DEFAULT 0,
    status        TEXT    NOT NULL DEFAULT 'success',
    fallback_from TEXT,
    skill_id      INTEGER REFERENCES skills(id) ON DELETE SET NULL,
    timestamp     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Provider tokens (references keychain)
CREATE TABLE IF NOT EXISTS provider_tokens (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id       TEXT    NOT NULL,
    auth_type         TEXT    NOT NULL,
    keychain_key      TEXT    NOT NULL,
    expires_at        DATETIME,
    refresh_token_key TEXT,
    detected_at       DATETIME,
    last_refreshed    DATETIME,
    is_active         INTEGER NOT NULL DEFAULT 1,
    UNIQUE(user_id, provider_id)
);

-- Model priority chain
CREATE TABLE IF NOT EXISTS model_priority (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    priority   INTEGER NOT NULL,
    provider   TEXT    NOT NULL,
    model      TEXT    NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    UNIQUE(user_id, priority)
);

-- Usage analytics daily aggregate
CREATE TABLE IF NOT EXISTS usage_daily (
    date           TEXT    NOT NULL,
    user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider       TEXT    NOT NULL,
    model          TEXT    NOT NULL,
    requests       INTEGER NOT NULL DEFAULT 0,
    input_tokens   INTEGER NOT NULL DEFAULT 0,
    output_tokens  INTEGER NOT NULL DEFAULT 0,
    errors         INTEGER NOT NULL DEFAULT 0,
    fallbacks      INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (date, user_id, provider, model)
);

-- Indexes cho queries thường dùng
CREATE INDEX IF NOT EXISTS idx_prompts_user ON prompts(user_id);
CREATE INDEX IF NOT EXISTS idx_prompts_slash ON prompts(slash_name) WHERE is_slash = 1;
CREATE INDEX IF NOT EXISTS idx_history_user_time ON history(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_history_cleanup ON history(timestamp);
```

- [ ] **Step 3.4: Tạo sqlite.go**

Tạo `go-engine/db/sqlite.go`:
```go
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minhtuancn/open-prompt/go-engine/config"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initSQL string

// DB wraps sql.DB với helpers
type DB struct {
	*sql.DB
}

// Open mở SQLite database tại path chuẩn
func Open() (*DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, config.DBFileName)
	return openPath(path)
}

// OpenInMemory mở SQLite in-memory (dùng cho test)
func OpenInMemory() (*DB, error) {
	return openPath(":memory:")
}

func openPath(path string) (*DB, error) {
	raw, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	raw.SetMaxOpenConns(1) // SQLite không hỗ trợ concurrent writes
	return &DB{raw}, nil
}

// Migrate chạy migration SQL
func Migrate(db *DB) error {
	_, err := db.Exec(initSQL)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

// dataDir trả về thư mục data của app (~/.open-prompt)
func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".open-prompt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}
```

- [ ] **Step 3.5: Chạy test — verify pass**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./db/... -v -run TestOpenAndMigrate
```

Expected: PASS — tất cả 9 tables tìm thấy.

- [ ] **Step 3.6: Tạo user_repo.go**

Tạo `go-engine/db/repos/user_repo.go`:
```go
package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// User là model cho bảng users
type User struct {
	ID           int64
	Username     string
	DisplayName  sql.NullString
	PasswordHash string
	AvatarColor  string
	CreatedAt    time.Time
	LastLogin    sql.NullTime
}

// UserRepo xử lý CRUD cho bảng users
type UserRepo struct {
	db *db.DB
}

// NewUserRepo tạo UserRepo mới
func NewUserRepo(database *db.DB) *UserRepo {
	return &UserRepo{db: database}
}

// Create tạo user mới
func (r *UserRepo) Create(username, passwordHash string) (*User, error) {
	res, err := r.db.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		username, passwordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, _ := res.LastInsertId()
	return r.FindByID(id)
}

// FindByUsername tìm user theo username
func (r *UserRepo) FindByUsername(username string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(
		`SELECT id, username, display_name, password_hash, avatar_color, created_at, last_login
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarColor, &u.CreatedAt, &u.LastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// FindByID tìm user theo ID
func (r *UserRepo) FindByID(id int64) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(
		`SELECT id, username, display_name, password_hash, avatar_color, created_at, last_login
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarColor, &u.CreatedAt, &u.LastLogin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// Count trả về tổng số users (dùng để detect first-run)
func (r *UserRepo) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// UpdateLastLogin cập nhật thời gian login cuối
func (r *UserRepo) UpdateLastLogin(id int64) error {
	_, err := r.db.Exec(`UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}
```

- [ ] **Step 3.7: Tạo settings_repo.go**

Tạo `go-engine/db/repos/settings_repo.go`:
```go
package repos

import (
	"database/sql"
	"fmt"

	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// SettingsRepo xử lý key-value settings per user
type SettingsRepo struct {
	db *db.DB
}

// NewSettingsRepo tạo SettingsRepo mới
func NewSettingsRepo(database *db.DB) *SettingsRepo {
	return &SettingsRepo{db: database}
}

// Get lấy giá trị setting, trả về empty string nếu không tồn tại
func (r *SettingsRepo) Get(userID int64, key string) (string, error) {
	var value sql.NullString
	err := r.db.QueryRow(
		`SELECT value FROM settings WHERE user_id = ? AND key = ?`, userID, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %q: %w", key, err)
	}
	return value.String, nil
}

// Set lưu setting (upsert)
func (r *SettingsRepo) Set(userID int64, key, value string) error {
	_, err := r.db.Exec(
		`INSERT INTO settings (user_id, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value`,
		userID, key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
```

- [ ] **Step 3.8: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/db/
git commit -m "feat: thêm database layer SQLite với migrations và repos"
```

---

## Task 4: Auth Service

**Files:**
- Create: `go-engine/auth/service.go`
- Create: `go-engine/auth/session.go`
- Test: `go-engine/auth/service_test.go`

- [ ] **Step 4.1: Viết failing tests cho auth**

Tạo `go-engine/auth/service_test.go`:
```go
package auth_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/auth"
	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestRegisterAndLogin(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc := auth.NewService(userRepo, "test-jwt-secret")

	// Register
	user, err := svc.Register("alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", user.Username)
	}

	// Login
	token, err := svc.Login("alice", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	// Validate token
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, claims.UserID)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc := auth.NewService(userRepo, "test-jwt-secret")

	_, _ = svc.Register("bob", "correct")
	_, err := svc.Login("bob", "wrong")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestFirstRun(t *testing.T) {
	database := setupTestDB(t)
	userRepo := repos.NewUserRepo(database)
	svc := auth.NewService(userRepo, "test-jwt-secret")

	isFirst, err := svc.IsFirstRun()
	if err != nil {
		t.Fatal(err)
	}
	if !isFirst {
		t.Error("expected first run before any users created")
	}

	_, _ = svc.Register("alice", "password")

	isFirst, _ = svc.IsFirstRun()
	if isFirst {
		t.Error("expected not first run after user created")
	}
}
```

- [ ] **Step 4.2: Chạy test — verify fail**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./auth/... -v 2>&1
```

Expected: FAIL "package auth not found"

- [ ] **Step 4.3: Implement service.go**

Tạo `go-engine/auth/service.go`:
```go
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

var (
	ErrUserExists       = errors.New("username already exists")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

// Service xử lý authentication logic
type Service struct {
	users     *repos.UserRepo
	jwtSecret string
}

// NewService tạo auth service mới
func NewService(userRepo *repos.UserRepo, jwtSecret string) *Service {
	return &Service{users: userRepo, jwtSecret: jwtSecret}
}

// IsFirstRun kiểm tra xem có user nào trong DB chưa
func (s *Service) IsFirstRun() (bool, error) {
	count, err := s.users.Count()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// Register tạo user mới với bcrypt password
func (s *Service) Register(username, password string) (*repos.User, error) {
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Kiểm tra username đã tồn tại chưa
	existing, err := s.users.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), config.DefaultBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	return s.users.Create(username, string(hash))
}

// Login xác thực user và trả về JWT token
func (s *Service) Login(username, password string) (string, error) {
	user, err := s.users.FindByUsername(username)
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return "", ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidPassword
	}

	_ = s.users.UpdateLastLogin(user.ID)

	return issueToken(user.ID, user.Username, s.jwtSecret)
}

// ValidateToken kiểm tra JWT và trả về claims
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, s.jwtSecret)
}
```

- [ ] **Step 4.4: Implement session.go (JWT)**

Tạo `go-engine/auth/session.go`:
```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/minhtuancn/open-prompt/go-engine/config"
)

// Claims là JWT claims cho session
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"sub"`
	jwt.RegisteredClaims
}

// issueToken tạo JWT cho user
func issueToken(userID int64, username, secret string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 0, config.DefaultJWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// parseToken validates và decode JWT
func parseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}
```

- [ ] **Step 4.5: Chạy tests — verify pass**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./auth/... -v
```

Expected: PASS — 3 tests pass.

- [ ] **Step 4.6: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/auth/
git commit -m "feat: thêm auth service với bcrypt và JWT"
```

---

## Task 5: JSON-RPC Server

**Files:**
- Create: `go-engine/api/server.go`
- Create: `go-engine/api/router.go`
- Create: `go-engine/api/middleware.go`
- Create: `go-engine/api/handlers_auth.go`
- Test: `go-engine/api/server_test.go`

- [ ] **Step 5.1: Viết failing test cho server**

Tạo `go-engine/api/server_test.go`:
```go
package api_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/api"
	"github.com/minhtuancn/open-prompt/go-engine/db"
)

func setupServer(t *testing.T) (*api.Server, string) {
	t.Helper()
	database, _ := db.OpenInMemory()
	db.Migrate(database)
	t.Cleanup(func() { database.Close() })

	secret := "test-secret"
	srv, err := api.NewServer(secret, database)
	if err != nil {
		t.Fatal(err)
	}

	// Dùng random port TCP thay vì Unix socket cho test
	addr := srv.TestAddr()
	go srv.Listen()
	time.Sleep(50 * time.Millisecond) // đợi server ready
	t.Cleanup(srv.Close)
	return srv, addr
}

func callRPC(t *testing.T, addr, secret, method string, params interface{}) api.Response {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	req := api.Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
	// Thêm secret header vào đầu message
	msg, _ := json.Marshal(map[string]interface{}{
		"secret":  secret,
		"request": req,
	})
	conn.Write(append(msg, '\n'))

	var resp api.Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestAuthRegisterAndLogin(t *testing.T) {
	_, addr := setupServer(t)

	// Register
	resp := callRPC(t, addr, "test-secret", "auth.register", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	if resp.Error != nil {
		t.Fatalf("register error: %v", resp.Error)
	}

	// Login
	resp = callRPC(t, addr, "test-secret", "auth.login", map[string]string{
		"username": "alice",
		"password": "password123",
	})
	if resp.Error != nil {
		t.Fatalf("login error: %v", resp.Error)
	}
}
```

- [ ] **Step 5.2: Chạy test — verify fail**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./api/... -v -run TestAuthRegisterAndLogin 2>&1 | head -20
```

Expected: FAIL "package api not found"

- [ ] **Step 5.3: Implement server.go**

Tạo `go-engine/api/server.go`:
```go
package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/minhtuancn/open-prompt/go-engine/config"
	"github.com/minhtuancn/open-prompt/go-engine/db"
)

// Server là JSON-RPC server qua Unix socket / Named Pipe / TCP (test)
type Server struct {
	secret   string
	db       *db.DB
	router   *Router
	listener net.Listener
	testAddr string // chỉ dùng trong test mode
}

// NewServer tạo server mới
func NewServer(secret string, database *db.DB) (*Server, error) {
	s := &Server{
		secret: secret,
		db:     database,
	}
	s.router = newRouter(s)
	return s, nil
}

// TestAddr trả về địa chỉ TCP cho test (tạo listener ngẫu nhiên)
func (s *Server) TestAddr() string {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	s.listener = ln
	return ln.Addr().String()
}

// Listen bắt đầu lắng nghe connections
func (s *Server) Listen() error {
	if s.listener == nil {
		var err error
		s.listener, err = createListener()
		if err != nil {
			return fmt.Errorf("create listener: %w", err)
		}
	}
	log.Printf("listening on %s", s.listener.Addr())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return nil // closed
		}
		go s.handleConn(conn)
	}
}

// Close đóng server
func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// handleConn xử lý một connection
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB buffer

	for scanner.Scan() {
		line := scanner.Bytes()
		resp := s.processMessage(conn, line)
		if resp != nil {
			data, _ := json.Marshal(resp)
			conn.Write(append(data, '\n'))
		}
	}
}

// processMessage decode và dispatch một message
func (s *Server) processMessage(conn net.Conn, data []byte) *Response {
	// Decode envelope với secret
	var envelope struct {
		Secret  string  `json:"secret"`
		Request Request `json:"request"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return &Response{JSONRPC: "2.0", Error: ErrInvalidParams}
	}

	// Validate secret
	if envelope.Secret != s.secret {
		return &Response{JSONRPC: "2.0", Error: ErrUnauthorized, ID: envelope.Request.ID}
	}

	// Dispatch
	result, rpcErr := s.router.dispatch(conn, &envelope.Request)
	if rpcErr != nil {
		return &Response{JSONRPC: "2.0", Error: rpcErr, ID: envelope.Request.ID}
	}
	resp := NewResponse(envelope.Request.ID, result)
	return &resp
}

// createListener tạo Unix socket (Linux/macOS) hoặc Named Pipe (Windows)
func createListener() (net.Listener, error) {
	if runtime.GOOS == "windows" {
		// Windows: dùng TCP localhost thay vì Named Pipe (đơn giản hơn cho v1)
		return net.Listen("tcp", "127.0.0.1:0")
	}
	// Linux/macOS: Unix socket
	os.Remove(config.SocketPath) // remove stale socket
	return net.Listen("unix", config.SocketPath)
}

// SendNotification gửi JSON-RPC notification qua connection (dùng cho streaming)
func SendNotification(conn net.Conn, method string, params interface{}) error {
	n := Notification{JSONRPC: "2.0", Method: method, Params: params}
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}
```

- [ ] **Step 5.4: Implement router.go và handlers_auth.go**

Tạo `go-engine/api/router.go`:
```go
package api

import (
	"fmt"
	"net"

	"github.com/minhtuancn/open-prompt/go-engine/auth"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// Router map method → handler
type Router struct {
	server  *Server
	auth    *auth.Service
	users   *repos.UserRepo
	settings *repos.SettingsRepo
}

func newRouter(s *Server) *Router {
	users := repos.NewUserRepo(s.db)
	settings := repos.NewSettingsRepo(s.db)
	// JWT secret từ settings hoặc generate mới
	jwtSecret := mustGetJWTSecret(settings)
	return &Router{
		server:   s,
		auth:     auth.NewService(users, jwtSecret),
		users:    users,
		settings: settings,
	}
}

// dispatch gọi handler tương ứng với method
func (r *Router) dispatch(conn net.Conn, req *Request) (interface{}, *RPCError) {
	switch req.Method {
	case "auth.register":
		return r.handleRegister(req)
	case "auth.login":
		return r.handleLogin(req)
	case "auth.me":
		return r.handleMe(req)
	case "auth.is_first_run":
		return r.handleIsFirstRun(req)
	case "settings.get":
		return r.handleSettingsGet(req)
	case "settings.set":
		return r.handleSettingsSet(req)
	case "query.stream":
		return r.handleQueryStream(conn, req)
	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}

// mustGetJWTSecret lấy JWT secret từ settings hoặc tạo mới
func mustGetJWTSecret(settings *repos.SettingsRepo) string {
	// Dùng system-level secret (user_id = 0 không tồn tại, dùng global key)
	// Tạm thời dùng hardcoded secret cho v1, sẽ replace bằng keychain trong Phase 2
	return "open-prompt-jwt-secret-v1"
}
```

Tạo `go-engine/api/handlers_auth.go`:
```go
package api

import (
	"encoding/json"
	"fmt"
)

type registerParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenParams struct {
	Token string `json:"token"`
}

func (r *Router) handleRegister(req *Request) (interface{}, *RPCError) {
	var p registerParams
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, ErrInvalidParams
	}
	user, err := r.auth.Register(p.Username, p.Password)
	if err != nil {
		return nil, &RPCError{Code: -32001, Message: err.Error()}
	}
	return map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	}, nil
}

func (r *Router) handleLogin(req *Request) (interface{}, *RPCError) {
	var p loginParams
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, ErrInvalidParams
	}
	token, err := r.auth.Login(p.Username, p.Password)
	if err != nil {
		return nil, &RPCError{Code: -32001, Message: err.Error()}
	}
	return map[string]string{"token": token}, nil
}

func (r *Router) handleMe(req *Request) (interface{}, *RPCError) {
	var p tokenParams
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, ErrInvalidParams
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return map[string]interface{}{
		"user_id":  claims.UserID,
		"username": claims.Username,
	}, nil
}

func (r *Router) handleIsFirstRun(req *Request) (interface{}, *RPCError) {
	isFirst, err := r.auth.IsFirstRun()
	if err != nil {
		return nil, ErrInternal
	}
	return map[string]bool{"is_first_run": isFirst}, nil
}

func (r *Router) handleSettingsGet(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token string `json:"token"`
		Key   string `json:"key"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, ErrInvalidParams
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, ErrUnauthorized
	}
	value, err := r.settings.Get(claims.UserID, p.Key)
	if err != nil {
		return nil, ErrInternal
	}
	return map[string]string{"value": value}, nil
}

func (r *Router) handleSettingsSet(req *Request) (interface{}, *RPCError) {
	var p struct {
		Token string `json:"token"`
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, ErrInvalidParams
	}
	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if err := r.settings.Set(claims.UserID, p.Key, p.Value); err != nil {
		return nil, ErrInternal
	}
	return map[string]bool{"ok": true}, nil
}

// handleQueryStream — sẽ implement đầy đủ trong Task 6 (handlers_query.go)
// KHÔNG định nghĩa ở đây — để tránh duplicate method error
// Dispatch được thêm vào switch ở Task 6

// decodeParams decode params từ interface{} sang struct
func decodeParams(params interface{}, dst interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	return json.Unmarshal(data, dst)
}
```

- [ ] **Step 5.5: Chạy test — verify pass**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./api/... -v -run TestAuthRegisterAndLogin
```

Expected: PASS

- [ ] **Step 5.6: Build Go engine**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go build -o bin/go-engine .
echo "Build OK: $(ls -la bin/go-engine)"
```

Expected: binary được tạo.

- [ ] **Step 5.7: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/api/
git commit -m "feat: thêm JSON-RPC server với auth handlers"
```

---

## Task 6: Anthropic Provider (Single Provider MVP)

**Files:**
- Create: `go-engine/model/providers/anthropic.go`
- Create: `go-engine/model/router.go`
- Create: `go-engine/model/stream.go`
- Create: `go-engine/api/handlers_query.go`
- Test: `go-engine/model/providers/anthropic_test.go`

- [ ] **Step 6.1: Viết failing test cho Anthropic provider**

Tạo `go-engine/model/providers/anthropic_test.go`:
```go
package providers_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// TestAnthropicProvider chỉ chạy khi có API key thật
// Dùng: ANTHROPIC_API_KEY=sk-... go test ./model/providers/... -v -run TestAnthropicProvider
func TestAnthropicProvider(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	p := providers.NewAnthropicProvider(apiKey)
	var sb strings.Builder

	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
		Prompt: "Say hello in one word.",
	}, func(chunk string) {
		sb.WriteString(chunk)
	})

	if err != nil {
		t.Fatalf("stream complete: %v", err)
	}
	if sb.Len() == 0 {
		t.Error("expected non-empty response")
	}
	t.Logf("response: %q", sb.String())
}

func TestAnthropicProviderBadKey(t *testing.T) {
	p := providers.NewAnthropicProvider("sk-bad-key")
	err := p.StreamComplete(context.Background(), providers.CompletionRequest{
		Model:  "claude-3-5-haiku-20241022",
		Prompt: "hello",
	}, func(chunk string) {})

	if err == nil {
		t.Error("expected error for bad API key")
	}
}
```

- [ ] **Step 6.2: Tạo CompletionRequest type**

Tạo `go-engine/model/types.go`:
```go
package model

// CompletionRequest là request gửi đến AI provider
type CompletionRequest struct {
	Model       string
	Prompt      string
	System      string
	Temperature float64
	MaxTokens   int
	Stream      bool
}

// CompletionResult là kết quả sau khi complete
type CompletionResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
	LatencyMs    int64
}
```

- [ ] **Step 6.3: Implement Anthropic provider**

Tạo `go-engine/model/providers/anthropic.go`:
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

const anthropicBaseURL = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

// CompletionRequest là request gửi đến provider
type CompletionRequest struct {
	Model       string
	Prompt      string
	System      string
	Temperature float64
	MaxTokens   int
}

// AnthropicProvider gọi Anthropic Messages API
type AnthropicProvider struct {
	apiKey string
	client *http.Client
}

// NewAnthropicProvider tạo provider mới
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// StreamComplete gọi Anthropic API với streaming
func (p *AnthropicProvider) StreamComplete(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
	}
	if req.System != "" {
		body["system"] = req.System
	}

	bodyBytes, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicBaseURL+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse Server-Sent Events
	scanner := bufio.NewScanner(resp.Body)
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
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			onChunk(event.Delta.Text)
		}
	}
	return scanner.Err()
}
```

- [ ] **Step 6.4: Implement model router**

Tạo `go-engine/model/router.go`:
```go
package model

import (
	"context"
	"fmt"
	"time"

	"github.com/minhtuancn/open-prompt/go-engine/model/providers"
)

// Router route request đến đúng provider
type Router struct {
	providers map[string]*providers.AnthropicProvider
}

// NewRouter tạo router mới
func NewRouter() *Router {
	return &Router{
		providers: make(map[string]*providers.AnthropicProvider),
	}
}

// RegisterAnthropic đăng ký Anthropic provider
func (r *Router) RegisterAnthropic(apiKey string) {
	r.providers["anthropic"] = providers.NewAnthropicProvider(apiKey)
}

// Stream gửi request và stream response qua callback
func (r *Router) Stream(ctx context.Context, req CompletionRequest, onChunk func(string)) error {
	p, ok := r.providers["anthropic"]
	if !ok {
		return fmt.Errorf("no provider configured for anthropic")
	}

	start := time.Now()
	err := p.StreamComplete(ctx, providers.CompletionRequest{
		Model:       req.Model,
		Prompt:      req.Prompt,
		System:      req.System,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}, onChunk)

	_ = time.Since(start) // sẽ dùng để log latency sau
	return err
}
```

- [ ] **Step 6.5: Implement handleQueryStream**

> **Quan trọng:** Chỉ định nghĩa `handleQueryStream` trong file này. `router.go` đã có comment placeholder (không phải method) — không cần xóa gì thêm.

Tạo mới `go-engine/api/handlers_query.go`:
```go
package api

import (
	"context"
	"fmt"
	"net"

	"github.com/minhtuancn/open-prompt/go-engine/model"
)

func (r *Router) handleQueryStream(conn interface{}, req *Request) (interface{}, *RPCError) {
	var p struct {
		Token  string `json:"token"`
		Input  string `json:"input"`
		Model  string `json:"model"`
		System string `json:"system"`
	}
	if err := decodeParams(req.Params, &p); err != nil {
		return nil, ErrInvalidParams
	}

	claims, err := r.auth.ValidateToken(p.Token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Lấy API key từ settings
	apiKey, _ := r.settings.Get(claims.UserID, "anthropic_api_key")
	if apiKey == "" {
		return nil, &RPCError{Code: -32002, Message: "anthropic API key not configured"}
	}

	// Build model router
	modelRouter := model.NewRouter()
	modelRouter.RegisterAnthropic(apiKey)

	modelName := p.Model
	if modelName == "" {
		modelName = "claude-3-5-sonnet-20241022"
	}

	netConn, ok := conn.(net.Conn)
	if !ok {
		return nil, ErrInternal
	}

	// Stream response qua JSON-RPC notifications
	ctx := context.Background()
	err = modelRouter.Stream(ctx, model.CompletionRequest{
		Model:  modelName,
		Prompt: p.Input,
		System: p.System,
	}, func(chunk string) {
		SendNotification(netConn, "stream.chunk", map[string]interface{}{
			"delta": chunk,
			"done":  false,
		})
	})

	if err != nil {
		SendNotification(netConn, "stream.chunk", map[string]interface{}{
			"delta": "",
			"done":  true,
			"error": fmt.Sprintf("%v", err),
		})
		return nil, nil // notification đã gửi, không cần return error
	}

	// Gửi done notification
	SendNotification(netConn, "stream.chunk", map[string]interface{}{
		"delta": "",
		"done":  true,
	})

	return nil, nil // response đã qua notifications
}
```

Xóa placeholder trong `router.go`:
```go
// Xóa dòng này trong router.go dispatch:
// case "query.stream":
//     return r.handleQueryStream(conn, req)
// Và thêm lại đúng signature:
```

Sửa `go-engine/api/router.go` — cập nhật dispatch signature để truyền `net.Conn`:
```go
// dispatch nhận conn interface{} để handlers_query có thể dùng net.Conn
func (r *Router) dispatch(conn interface{}, req *Request) (interface{}, *RPCError) {
    // ... (giữ nguyên, đã đúng)
    case "query.stream":
        return r.handleQueryStream(conn, req)
    // ...
}
```

- [ ] **Step 6.6: Build và verify không có compile error**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go build ./...
```

Expected: build thành công, không có error.

- [ ] **Step 6.7: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add go-engine/model/ go-engine/api/handlers_query.go
git commit -m "feat: thêm Anthropic provider và streaming query handler"
```

---

## Task 7: Tauri — Sidecar + IPC

**Files:**
- Create: `src-tauri/src/sidecar.rs`
- Create: `src-tauri/src/ipc.rs`
- Modify: `src-tauri/src/lib.rs`
- Modify: `src-tauri/src/main.rs`

- [ ] **Step 7.1: Copy Go binary vào Tauri sidecar path**

Tauri sidecar cần binary ở đúng path. Tạo script:

Tạo `scripts/build-engine.sh`:
```bash
#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPT_DIR/.."

cd "$ROOT/go-engine"

# Build cho host platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

echo "Building Go engine for $OS-$ARCH..."
go build -o "bin/go-engine-$OS-$ARCH" .

# Copy vào Tauri sidecar path
SIDECAR_DIR="$ROOT/src-tauri/binaries"
mkdir -p "$SIDECAR_DIR"

TARGET_TRIPLE=""
case "$OS-$ARCH" in
    linux-amd64)   TARGET_TRIPLE="x86_64-unknown-linux-gnu" ;;
    darwin-amd64)  TARGET_TRIPLE="x86_64-apple-darwin" ;;
    darwin-arm64)  TARGET_TRIPLE="aarch64-apple-darwin" ;;
    windows-amd64) TARGET_TRIPLE="x86_64-pc-windows-msvc" ;;
esac

cp "bin/go-engine-$OS-$ARCH" "$SIDECAR_DIR/go-engine-$TARGET_TRIPLE"
echo "Copied to $SIDECAR_DIR/go-engine-$TARGET_TRIPLE"
```

```bash
chmod +x scripts/build-engine.sh
bash scripts/build-engine.sh
```

- [ ] **Step 7.2: Cấu hình Tauri sidecar**

Thêm vào `src-tauri/tauri.conf.json` trong section `bundle`:
```json
"externalBin": ["binaries/go-engine"]
```

Thêm vào `src-tauri/capabilities/main.json`:
```json
{
  "$schema": "../gen/schemas/desktop-schema.json",
  "identifier": "main-capability",
  "description": "Capability for the main window",
  "windows": ["overlay"],
  "permissions": [
    "core:default",
    "shell:allow-execute",
    "shell:allow-open",
    "global-shortcut:allow-register",
    "global-shortcut:allow-unregister"
  ]
}
```

- [ ] **Step 7.3: Implement sidecar.rs**

Tạo `src-tauri/src/sidecar.rs`:
```rust
use std::process::{Child, Command};
use std::sync::Mutex;
use tauri::{AppHandle, Manager};

/// SidecarState giữ process handle của Go engine
pub struct SidecarState(pub Mutex<Option<Child>>);

/// Spawn Go engine sidecar và đợi "ready" signal từ stdout
pub fn spawn_engine(app: &AppHandle) -> Result<(), String> {
    // Tạo shared secret ngẫu nhiên
    let secret: String = (0..32)
        .map(|_| format!("{:02x}", rand::random::<u8>()))
        .collect();

    // Lưu secret vào app state để IPC dùng
    app.manage(EngineSecret(secret.clone()));

    let sidecar_path = app
        .path()
        .resolve("binaries/go-engine", tauri::path::BaseDirectory::Resource)
        .map_err(|e| format!("resolve sidecar path: {e}"))?;

    let mut child = Command::new(&sidecar_path)
        .env("OP_SOCKET_SECRET", &secret)
        .stdout(std::process::Stdio::piped())
        .spawn()
        .map_err(|e| format!("spawn go engine: {e}"))?;

    // Đợi "ready" từ stdout
    use std::io::{BufRead, BufReader};
    let stdout = child.stdout.take().unwrap();
    let mut reader = BufReader::new(stdout);
    let mut line = String::new();
    reader.read_line(&mut line).map_err(|e| format!("read stdout: {e}"))?;

    if !line.trim().eq("ready") {
        return Err(format!("unexpected startup output: {line}"));
    }

    // Lưu child process
    let state = app.state::<SidecarState>();
    *state.0.lock().unwrap() = Some(child);

    Ok(())
}

// Trên Windows: Go engine in port ra stdout sau "ready"
// Đọc port và lưu vào EnginePort state
// (Linux/macOS dùng Unix socket, không cần port)
#[cfg(windows)]
fn read_engine_port(reader: &mut impl BufRead) -> Result<u16, String> {
    let mut line = String::new();
    reader.read_line(&mut line).map_err(|e| format!("read port: {e}"))?;
    line.trim().parse::<u16>().map_err(|e| format!("parse port: {e}"))
}

/// EngineSecret lưu shared secret để IPC dùng
pub struct EngineSecret(pub String);

/// EnginePort lưu TCP port (chỉ dùng trên Windows)
pub struct EnginePort(pub u16);
```

- [ ] **Step 7.4: Implement ipc.rs**

> **Kiến trúc streaming:** `call_engine` command xử lý hai loại request:
> 1. **Regular requests** (auth.login, settings.get, v.v.): đọc 1 response line, return.
> 2. **Streaming requests** (query.stream): đọc nhiều notification lines (`stream.chunk`), emit Tauri event `stream-chunk` cho mỗi chunk, cho đến khi `done: true`.

Tạo `src-tauri/src/ipc.rs`:
```rust
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::io::{BufRead, BufReader, Write};
use tauri::{command, AppHandle, Emitter, Manager};

use crate::sidecar::EngineSecret;

#[derive(Serialize, Deserialize, Debug)]
struct RpcEnvelope {
    secret: String,
    request: RpcRequestInner,
}

#[derive(Serialize, Deserialize, Debug)]
struct RpcRequestInner {
    jsonrpc: String,
    method: String,
    params: Value,
    id: u64,
}

#[derive(Serialize, Deserialize, Debug)]
struct RpcResponse {
    jsonrpc: Option<String>,
    result: Option<Value>,
    error: Option<Value>,
    id: Option<Value>,
    // Notification fields
    method: Option<String>,
    params: Option<Value>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct StreamChunk {
    pub delta: String,
    pub done: bool,
    pub error: Option<String>,
}

/// call_engine gọi Go Engine qua socket
/// - Với query.stream: đọc notifications và emit "stream-chunk" events
/// - Với các method khác: đọc 1 response và return
#[command]
pub async fn call_engine(
    app: AppHandle,
    method: String,
    params: Value,
) -> Result<Value, String> {
    let secret = app.state::<EngineSecret>().0.clone();
    let is_streaming = method == "query.stream";

    let envelope = RpcEnvelope {
        secret,
        request: RpcRequestInner {
            jsonrpc: "2.0".into(),
            method: method.clone(),
            params,
            id: 1,
        },
    };

    let mut msg = serde_json::to_vec(&envelope).map_err(|e| e.to_string())?;
    msg.push(b'\n');

    // Dùng spawn_blocking vì socket I/O là blocking
    let app_clone = app.clone();
    tauri::async_runtime::spawn_blocking(move || {
        #[cfg(unix)]
        {
            use std::os::unix::net::UnixStream;
            let mut conn = UnixStream::connect("/tmp/open-prompt.sock")
                .map_err(|e| format!("connect: {e}"))?;
            conn.write_all(&msg).map_err(|e| e.to_string())?;
            handle_response(BufReader::new(conn), is_streaming, &app_clone)
        }
        #[cfg(windows)]
        {
            // Windows v1: Go engine print port qua stdout, Tauri lưu trong EnginePort state
            // Tạm thời: port được lưu khi spawn (xem sidecar.rs)
            let port = app_clone.state::<crate::sidecar::EnginePort>().0;
            let addr = format!("127.0.0.1:{port}");
            let mut conn = std::net::TcpStream::connect(&addr)
                .map_err(|e| format!("connect {addr}: {e}"))?;
            conn.write_all(&msg).map_err(|e| e.to_string())?;
            handle_response(BufReader::new(conn), is_streaming, &app_clone)
        }
    })
    .await
    .map_err(|e| e.to_string())?
}

/// handle_response đọc response(s) từ socket
fn handle_response<R: BufRead>(
    mut reader: R,
    is_streaming: bool,
    app: &AppHandle,
) -> Result<Value, String> {
    loop {
        let mut line = String::new();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let msg: RpcResponse = serde_json::from_str(line).map_err(|e| {
            format!("parse response: {e} — raw: {line}")
        })?;

        // Nếu là notification (stream.chunk)
        if msg.method.as_deref() == Some("stream.chunk") {
            if let Some(params) = msg.params {
                let chunk: StreamChunk = serde_json::from_value(params)
                    .map_err(|e| format!("parse chunk: {e}"))?;
                let done = chunk.done;
                // Emit event lên React frontend
                app.emit("stream-chunk", chunk).map_err(|e| e.to_string())?;
                if done {
                    return Ok(Value::Null);
                }
            }
            continue; // đọc notification tiếp theo
        }

        // Regular JSON-RPC response
        if let Some(err) = msg.error {
            return Err(err.to_string());
        }
        return Ok(msg.result.unwrap_or(Value::Null));
    }
}
```

- [ ] **Step 7.5: Cập nhật lib.rs**

Sửa `src-tauri/src/lib.rs`:
```rust
mod hotkey;
mod ipc;
mod sidecar;
mod tray;
mod window;

pub use sidecar::{EnginePort, EngineSecret, SidecarState};

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .plugin(tauri_plugin_shell::init())
        .manage(SidecarState(std::sync::Mutex::new(None)))
        .manage(EnginePort(0)) // sẽ được update sau khi spawn trên Windows
        .invoke_handler(tauri::generate_handler![ipc::call_engine])
        .setup(|app| {
            // Spawn Go engine
            sidecar::spawn_engine(app.handle())
                .expect("failed to spawn go engine");

            // Setup system tray
            tray::setup_tray(app.handle())?;

            // Register hotkey
            hotkey::register_hotkey(app.handle())?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error running tauri application");
}
```

- [ ] **Step 7.6: Tạo hotkey.rs và tray.rs (stubs)**

Tạo `src-tauri/src/hotkey.rs`:
```rust
use tauri::{AppHandle, Error};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, ShortcutState};

pub fn register_hotkey(app: &AppHandle) -> Result<(), Error> {
    app.global_shortcut().on_shortcut("Ctrl+Space", move |app_handle, shortcut, event| {
        if event.state == ShortcutState::Pressed {
            crate::window::toggle_overlay(app_handle);
        }
    })?;
    Ok(())
}
```

Tạo `src-tauri/src/tray.rs`:
```rust
use tauri::{AppHandle, Error, Manager};
use tauri::tray::TrayIconBuilder;
use tauri::menu::{MenuBuilder, MenuItemBuilder};

pub fn setup_tray(app: &AppHandle) -> Result<(), Error> {
    let quit = MenuItemBuilder::with_id("quit", "Quit Open Prompt").build(app)?;
    let menu = MenuBuilder::new(app).items(&[&quit]).build()?;

    TrayIconBuilder::new()
        .menu(&menu)
        .on_menu_event(|app, event| {
            if event.id == "quit" {
                app.exit(0);
            }
        })
        .build(app)?;

    Ok(())
}
```

Tạo `src-tauri/src/window.rs`:
```rust
use tauri::{AppHandle, Manager};

pub fn toggle_overlay(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("overlay") {
        if window.is_visible().unwrap_or(false) {
            let _ = window.hide();
        } else {
            let _ = window.show();
            let _ = window.set_focus();
        }
    }
}
```

- [ ] **Step 7.7: Thêm rand dependency vào Cargo.toml**

Chạy lệnh sau để thêm (không overwrite toàn bộ file):
```bash
cd src-tauri
cargo add rand@0.8 serde --features serde/derive
cargo add serde_json
cd ..
```

> **Quan trọng:** Dùng `cargo add` thay vì sửa tay `Cargo.toml` để tránh xóa nhầm các Tauri dependencies đã có.

- [ ] **Step 7.8: Verify Rust compile**

```bash
cd /home/dev/open-prompt-code/open-prompt/src-tauri
cargo check 2>&1
```

Expected: compile thành công (có thể có warnings, không có errors).

- [ ] **Step 7.9: Commit**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add src-tauri/ scripts/
git commit -m "feat: thêm Tauri sidecar spawn và IPC bridge đến Go engine"
```

---

## Task 8: React UI — Onboarding + Overlay

**Files:**
- Create: `src/store/authStore.ts`
- Create: `src/store/overlayStore.ts`
- Create: `src/hooks/useEngine.ts`
- Create: `src/components/onboarding/CreateAccount.tsx`
- Create: `src/components/auth/LoginScreen.tsx`
- Create: `src/components/overlay/CommandInput.tsx`
- Create: `src/components/overlay/ResponsePanel.tsx`
- Modify: `src/App.tsx`

- [ ] **Step 8.1: Tạo auth store**

Tạo `src/store/authStore.ts`:
```ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  token: string | null
  username: string | null
  userId: number | null
  setAuth: (token: string, username: string, userId: number) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      username: null,
      userId: null,
      setAuth: (token, username, userId) => set({ token, username, userId }),
      clearAuth: () => set({ token: null, username: null, userId: null }),
    }),
    { name: 'op-auth' }
  )
)
```

- [ ] **Step 8.2: Tạo overlay store**

Tạo `src/store/overlayStore.ts`:
```ts
import { create } from 'zustand'

interface OverlayState {
  input: string
  chunks: string[]
  isStreaming: boolean
  error: string | null
  setInput: (input: string) => void
  appendChunk: (chunk: string) => void
  setStreaming: (v: boolean) => void
  setError: (e: string | null) => void
  reset: () => void
}

export const useOverlayStore = create<OverlayState>()((set) => ({
  input: '',
  chunks: [],
  isStreaming: false,
  error: null,
  setInput: (input) => set({ input }),
  appendChunk: (chunk) => set((s) => ({ chunks: [...s.chunks, chunk] })),
  setStreaming: (isStreaming) => set({ isStreaming }),
  setError: (error) => set({ error }),
  reset: () => set({ input: '', chunks: [], isStreaming: false, error: null }),
}))
```

- [ ] **Step 8.3: Tạo useEngine hook**

Tạo `src/hooks/useEngine.ts`:
```ts
import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'

/** callEngine gọi Go Engine qua Tauri IPC */
export async function callEngine<T>(method: string, params: Record<string, unknown>): Promise<T> {
  return invoke<T>('call_engine', { method, params })
}

/** streamQuery gọi query.stream và subscribe notifications */
export async function streamQuery(
  params: { token: string; input: string; model?: string },
  onChunk: (chunk: string) => void,
  onDone: () => void,
  onError: (err: string) => void,
): Promise<void> {
  // Tauri sẽ forward stream.chunk events từ Go Engine
  const unlisten = await listen<{ delta: string; done: boolean; error?: string }>(
    'stream-chunk',
    (event) => {
      const { delta, done, error } = event.payload
      if (error) {
        onError(error)
        unlisten()
        return
      }
      if (done) {
        onDone()
        unlisten()
        return
      }
      onChunk(delta)
    }
  )

  // Trigger stream (fire and forget — response đến qua events)
  callEngine('query.stream', params).catch(onError)
}
```

- [ ] **Step 8.4: Tạo CreateAccount component**

Tạo `src/components/onboarding/CreateAccount.tsx`:
```tsx
import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

export function CreateAccount({ onDone }: { onDone: () => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const setAuth = useAuthStore((s) => s.setAuth)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password.length < 8) {
      setError('Mật khẩu cần ít nhất 8 ký tự')
      return
    }
    setLoading(true)
    setError('')
    try {
      await callEngine('auth.register', { username, password })
      const result = await callEngine<{ token: string }>('auth.login', { username, password })
      const me = await callEngine<{ user_id: number; username: string }>('auth.me', { token: result.token })
      setAuth(result.token, me.username, me.user_id)
      onDone()
    } catch (err: unknown) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-surface text-white">
      <h1 className="text-2xl font-bold mb-6">Chào mừng đến Open Prompt</h1>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3 w-80">
        <input
          autoFocus
          className="bg-white/10 rounded-lg px-4 py-2 outline-none focus:ring-2 ring-accent"
          placeholder="Tên đăng nhập"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <input
          type="password"
          className="bg-white/10 rounded-lg px-4 py-2 outline-none focus:ring-2 ring-accent"
          placeholder="Mật khẩu (ít nhất 8 ký tự)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="bg-accent rounded-lg py-2 font-semibold disabled:opacity-50 hover:bg-indigo-500 transition"
        >
          {loading ? 'Đang tạo...' : 'Tạo tài khoản'}
        </button>
      </form>
    </div>
  )
}
```

- [ ] **Step 8.5: Tạo CommandInput + ResponsePanel**

Tạo `src/components/overlay/CommandInput.tsx`:
```tsx
import { useRef } from 'react'
import { useOverlayStore } from '../../store/overlayStore'

interface Props {
  onSubmit: (input: string) => void
}

export function CommandInput({ onSubmit }: Props) {
  const { input, setInput, isStreaming } = useOverlayStore()
  const ref = useRef<HTMLTextAreaElement>(null)

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (input.trim() && !isStreaming) {
        onSubmit(input.trim())
      }
    }
    if (e.key === 'Escape') {
      window.close()
    }
  }

  return (
    <div className="relative">
      <textarea
        ref={ref}
        autoFocus
        rows={1}
        className="w-full bg-transparent text-white text-lg placeholder-white/40 outline-none resize-none px-5 py-4 leading-relaxed"
        placeholder="Hỏi AI... (Enter để gửi, Shift+Enter xuống dòng)"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        disabled={isStreaming}
      />
    </div>
  )
}
```

Tạo `src/components/overlay/ResponsePanel.tsx`:
```tsx
import { useOverlayStore } from '../../store/overlayStore'

export function ResponsePanel() {
  const { chunks, isStreaming, error } = useOverlayStore()
  const text = chunks.join('')

  if (!text && !isStreaming && !error) return null

  return (
    <div className="px-5 pb-4 max-h-80 overflow-y-auto">
      <div className="border-t border-white/10 pt-3">
        {error ? (
          <p className="text-red-400 text-sm">{error}</p>
        ) : (
          <p className="text-white/90 text-sm leading-relaxed whitespace-pre-wrap">
            {text}
            {isStreaming && <span className="animate-pulse">▌</span>}
          </p>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 8.6: Cập nhật App.tsx**

Tạo `src/App.tsx`:
```tsx
import { useEffect, useState } from 'react'
import { callEngine } from './hooks/useEngine'
import { useAuthStore } from './store/authStore'
import { useOverlayStore } from './store/overlayStore'
import { CreateAccount } from './components/onboarding/CreateAccount'
import { CommandInput } from './components/overlay/CommandInput'
import { ResponsePanel } from './components/overlay/ResponsePanel'
import { streamQuery } from './hooks/useEngine'
import './styles/globals.css'

type AppState = 'loading' | 'first-run' | 'overlay'

export default function App() {
  const [state, setState] = useState<AppState>('loading')
  const { token } = useAuthStore()
  const { reset, appendChunk, setStreaming, setError } = useOverlayStore()

  useEffect(() => {
    async function init() {
      try {
        const result = await callEngine<{ is_first_run: boolean }>('auth.is_first_run', {})
        if (result.is_first_run) {
          setState('first-run')
        } else {
          setState('overlay')
        }
      } catch {
        setState('overlay')
      }
    }
    init()
  }, [])

  const handleQuery = async (input: string) => {
    if (!token) return
    reset()
    setStreaming(true)
    await streamQuery(
      { token, input },
      (chunk) => appendChunk(chunk),
      () => setStreaming(false),
      (err) => { setError(err); setStreaming(false) }
    )
  }

  if (state === 'loading') {
    return <div className="flex items-center justify-center h-screen bg-surface">
      <div className="text-white/40">Đang khởi động...</div>
    </div>
  }

  if (state === 'first-run') {
    return <CreateAccount onDone={() => setState('overlay')} />
  }

  return (
    <div className="bg-surface/95 backdrop-blur-xl rounded-2xl border border-white/10 shadow-2xl overflow-hidden min-h-16">
      <CommandInput onSubmit={handleQuery} />
      <ResponsePanel />
    </div>
  )
}
```

- [ ] **Step 8.7: Verify React build**

```bash
cd /home/dev/open-prompt-code/open-prompt
npm run build 2>&1 | tail -20
```

Expected: build thành công, không có TypeScript errors.

- [ ] **Step 8.8: Commit**

```bash
git add src/
git commit -m "feat: thêm overlay UI với onboarding, input và response streaming"
```

---

## Task 9: Settings — API Key Setup

**Files:**
- Create: `src/components/settings/ApiKeySetup.tsx`
- Modify: `src/App.tsx` (thêm settings flow sau first-run)

- [ ] **Step 9.1: Tạo ApiKeySetup component**

Tạo `src/components/settings/ApiKeySetup.tsx`:
```tsx
import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface Props {
  onDone: () => void
}

export function ApiKeySetup({ onDone }: Props) {
  const [apiKey, setApiKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { token } = useAuthStore()

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!apiKey.startsWith('sk-ant-')) {
      setError('Claude API key phải bắt đầu bằng sk-ant-')
      return
    }
    setLoading(true)
    try {
      await callEngine('settings.set', {
        token,
        key: 'anthropic_api_key',
        value: apiKey,
      })
      onDone()
    } catch (err: unknown) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-surface text-white">
      <h2 className="text-xl font-bold mb-2">Cấu hình AI Provider</h2>
      <p className="text-white/60 text-sm mb-6 text-center px-8">
        Nhập Anthropic API key để bắt đầu dùng Claude
      </p>
      <form onSubmit={handleSave} className="flex flex-col gap-3 w-80">
        <input
          autoFocus
          type="password"
          className="bg-white/10 rounded-lg px-4 py-2 outline-none focus:ring-2 ring-accent font-mono text-sm"
          placeholder="sk-ant-api03-..."
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          required
        />
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="bg-accent rounded-lg py-2 font-semibold disabled:opacity-50 hover:bg-indigo-500 transition"
        >
          {loading ? 'Đang lưu...' : 'Lưu và tiếp tục'}
        </button>
        <button
          type="button"
          onClick={onDone}
          className="text-white/40 text-sm hover:text-white/60"
        >
          Bỏ qua (cấu hình sau)
        </button>
      </form>
    </div>
  )
}
```

- [ ] **Step 9.2: Thêm ApiKeySetup vào App flow**

Sửa `src/App.tsx` — thêm state `'api-setup'` giữa `first-run` và `overlay`:
```tsx
// Thêm vào AppState type:
type AppState = 'loading' | 'first-run' | 'api-setup' | 'overlay'

// Trong CreateAccount onDone:
<CreateAccount onDone={() => setState('api-setup')} />

// Thêm case:
if (state === 'api-setup') {
  return <ApiKeySetup onDone={() => setState('overlay')} />
}
```

- [ ] **Step 9.3: Commit**

```bash
git add src/components/settings/
git commit -m "feat: thêm API key setup flow"
```

---

## Task 10: Integration Test & Dev Run

**Files:** Không tạo file mới — test toàn bộ flow.

- [ ] **Step 10.1: Build Go engine**

```bash
cd /home/dev/open-prompt-code/open-prompt
bash scripts/build-engine.sh
```

Expected: binary `src-tauri/binaries/go-engine-*` được tạo.

- [ ] **Step 10.2: Test Go engine standalone**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
OP_SOCKET_SECRET=test123 ./bin/go-engine &
ENGINE_PID=$!
sleep 1

# Gọi is_first_run
echo '{"secret":"test123","request":{"jsonrpc":"2.0","method":"auth.is_first_run","params":{},"id":1}}' | \
  nc -U /tmp/open-prompt.sock
echo ""

kill $ENGINE_PID
```

Expected: `{"jsonrpc":"2.0","result":{"is_first_run":true},"id":1}`

- [ ] **Step 10.3: Chạy toàn bộ Go tests**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./... -v -race 2>&1
```

Expected: tất cả tests PASS.

- [ ] **Step 10.4: Chạy Tauri dev mode**

```bash
cd /home/dev/open-prompt-code/open-prompt
npm run tauri dev 2>&1 &
```

Verify:
- App khởi động, Go engine được spawn
- Overlay window ở chế độ invisible
- System tray icon xuất hiện
- Ctrl+Space mở overlay

> **Lưu ý:** Cần DISPLAY environment variable trên Linux để test GUI.

- [ ] **Step 10.5: Test flow E2E**

1. App hiện màn hình "Tạo tài khoản" → nhập username + password → tiếp tục
2. Hiện màn hình nhập API key Claude → paste key thật → lưu
3. Ctrl+Space → overlay mở
4. Gõ "Say hello in Vietnamese" → Enter
5. Verify streaming response xuất hiện từng token

- [ ] **Step 10.6: Final commit Phase 1**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add -A
git commit -m "feat: hoàn thành Phase 1 - core loop overlay AI assistant"
```

---

## Task 11: CI Setup (GitHub Actions)

**Files:**
- Create: `.github/workflows/test.yml`
- Create: `.github/workflows/build.yml`

- [ ] **Step 11.1: Tạo test workflow**

Tạo `.github/workflows/test.yml`:
```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  go-tests:
    name: Go Engine Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache-dependency-path: go-engine/go.sum
      - name: Run Go tests
        run: |
          cd go-engine
          go test ./... -race -v

  typescript-check:
    name: TypeScript Check
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
      - run: npm run build
```

- [ ] **Step 11.2: Tạo build workflow**

Tạo `.github/workflows/build.yml`:
```yaml
name: Build

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    strategy:
      matrix:
        include:
          - platform: ubuntu-latest
            args: '--target x86_64-unknown-linux-gnu'
          - platform: windows-latest
            args: '--target x86_64-pc-windows-msvc'
          - platform: macos-latest
            args: '--target aarch64-apple-darwin --target x86_64-apple-darwin'

    runs-on: ${{ matrix.platform }}

    steps:
      - uses: actions/checkout@v4
        with:
          submodules: true

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install Rust
        uses: dtolnay/rust-toolchain@stable

      - name: Install Linux dependencies
        if: matrix.platform == 'ubuntu-latest'
        run: |
          sudo apt-get update
          sudo apt-get install -y libwebkit2gtk-4.1-dev libappindicator3-dev librsvg2-dev patchelf

      - name: Build Go engine
        run: bash scripts/build-engine.sh

      - name: Install Node deps
        run: npm ci

      - name: Build Tauri app
        uses: tauri-apps/tauri-action@v0
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          args: ${{ matrix.args }}
```

- [ ] **Step 11.3: Commit CI**

```bash
git add .github/
git commit -m "ci: thêm GitHub Actions workflows cho test và build"
```

---

## Checklist Hoàn thành Phase 1

- [ ] Go engine khởi động với `OP_SOCKET_SECRET`
- [ ] Go tests pass: auth, db, api
- [ ] Tauri spawn Go engine, nhận "ready" signal
- [ ] Ctrl+Space mở overlay window
- [ ] First-run flow: tạo tài khoản → nhập API key
- [ ] Gõ query → nhận streaming response từ Claude
- [ ] System tray icon + Quit menu
- [ ] GitHub Actions CI chạy được

---

## Phase tiếp theo

Sau khi Phase 1 hoàn thành và ổn định:

- **Plan 2:** Provider System (auto-detect, OAuth Copilot/Gemini, token manager)
- **Plan 3:** Prompt & Skill System (slash commands, CRUD, template engine)
- **Plan 4:** Input Injection (clipboard backup/restore, simulated typing)
- **Plan 5:** Analytics & Full Settings UI
- **Plan 6:** i18n + CI/CD Release pipeline
