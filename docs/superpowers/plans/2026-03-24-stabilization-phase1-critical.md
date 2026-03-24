# Stabilization Phase 1: CRITICAL Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 4 CRITICAL issues: IDOR in conversations, XSS in MarkdownRenderer, goroutine leak in RateLimiter, localStorage inconsistency.

**Architecture:** Fixes are independent — each task can be implemented and tested in isolation. Backend fixes (Go) and frontend fixes (React/TS) can run in parallel.

**Tech Stack:** Go 1.22+, React 18, TypeScript, SQLite, Zustand

**Spec:** `docs/superpowers/specs/2026-03-24-stabilization-polish-design.md`

---

## File Structure

| Task | Files | Action |
|------|-------|--------|
| 1. IDOR | `go-engine/db/repos/conversation_repo.go` | Modify |
| | `go-engine/api/handlers_conversations.go` | Modify |
| | `go-engine/api/handlers_query.go` | Modify |
| | `go-engine/db/repos/conversation_repo_test.go` | Create |
| 2. XSS | `src/components/overlay/MarkdownRenderer.tsx` | Modify |
| | `package.json` | Modify (add react-markdown, rehype-sanitize) |
| 3. Goroutine | `go-engine/api/ratelimit.go` | Modify |
| | `go-engine/api/ratelimit_test.go` | Create |
| | `go-engine/api/router.go` | Modify |
| 4. localStorage | `src/components/overlay/CommandInput.tsx` | Modify |
| | `src/components/prompts/PromptList.tsx` | Modify |
| | `src/components/prompts/PromptEditor.tsx` | Modify |
| | `src/components/skills/SkillEditor.tsx` | Modify |

---

## Task 1: Fix IDOR trong Conversation

**Files:**
- Modify: `go-engine/db/repos/conversation_repo.go:83-120`
- Modify: `go-engine/api/handlers_conversations.go:62-75`
- Modify: `go-engine/api/handlers_query.go:147-149`
- Create: `go-engine/db/repos/conversation_repo_test.go`

- [ ] **Step 1: Write failing test cho AddMessage với ownership check**

```go
// go-engine/db/repos/conversation_repo_test.go
package repos_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/db"
	"github.com/minhtuancn/open-prompt/go-engine/db/repos"
)

// Dùng chung newTestDB từ prompt_repo_test.go (đã có sẵn trong package).
// Nếu chạy riêng file này, cần copy helper hoặc extract ra testutil_test.go:
//
// func newTestDB(t *testing.T) *db.DB { ... }  — xem prompt_repo_test.go
//
// Helper tạo temp file DB, chạy Migrate, seed user_id=1.

func TestAddMessage_RejectsWrongUser(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewConversationRepo(database)

	// Cần thêm user_id=2 để test cross-user access
	userRepo := repos.NewUserRepo(database)
	_, err := userRepo.Create("user2", "hashedpw")
	if err != nil {
		t.Fatal(err)
	}

	// User 1 (seeded by newTestDB) tạo conversation
	convID, err := repo.Create(1, "Test conv")
	if err != nil {
		t.Fatal(err)
	}

	// User 2 thử add message vào conversation của user 1 → phải bị từ chối
	err = repo.AddMessage(convID, 2, "user", "hacked", "", "", 0)
	if err == nil {
		t.Fatal("expected error when adding message to another user's conversation")
	}
}

func TestGetMessages_RejectsWrongUser(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewConversationRepo(database)

	userRepo := repos.NewUserRepo(database)
	_, err := userRepo.Create("user2", "hashedpw")
	if err != nil {
		t.Fatal(err)
	}

	convID, err := repo.Create(1, "Test conv")
	if err != nil {
		t.Fatal(err)
	}

	// Add message as correct user
	err = repo.AddMessage(convID, 1, "user", "hello", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	// User 2 thử đọc → phải bị từ chối
	_, err = repo.GetMessages(convID, 2)
	if err == nil {
		t.Fatal("expected error when reading another user's conversation")
	}
}

func TestGetMessages_AllowsCorrectUser(t *testing.T) {
	database := newTestDB(t)
	repo := repos.NewConversationRepo(database)

	convID, err := repo.Create(1, "Test conv")
	if err != nil {
		t.Fatal(err)
	}

	err = repo.AddMessage(convID, 1, "user", "hello", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := repo.GetMessages(convID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("expected content 'hello', got '%s'", msgs[0].Content)
	}
}
```

**Lưu ý:** Test file dùng `package repos_test` (external test package) và `newTestDB` helper đã có sẵn trong `prompt_repo_test.go`. Helper này dùng `db.OpenPath` + `db.Migrate` + seed user_id=1, đúng pattern project.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/dev/open-prompt/go-engine && go test ./db/repos/ -run TestAddMessage_RejectsWrongUser -v`
Expected: FAIL — `AddMessage` chưa nhận tham số `userID`

- [ ] **Step 3: Modify `AddMessage` thêm userID parameter + ownership check**

Trong `go-engine/db/repos/conversation_repo.go`, sửa method `AddMessage` (line 83):

```go
// TRƯỚC:
func (r *ConversationRepo) AddMessage(convID int64, role, content, provider, model string, latencyMs int64) error {

// SAU:
func (r *ConversationRepo) AddMessage(convID, userID int64, role, content, provider, model string, latencyMs int64) error {
	// Verify conversation belongs to user
	var ownerID int64
	err := r.db.QueryRow(`SELECT user_id FROM conversations WHERE id = ?`, convID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("forbidden: conversation does not belong to user")
	}
```

Thêm `"fmt"` vào import nếu chưa có.

- [ ] **Step 4: Modify `GetMessages` thêm userID parameter + ownership check**

Trong `go-engine/db/repos/conversation_repo.go`, sửa method `GetMessages` (line 99):

```go
// TRƯỚC:
func (r *ConversationRepo) GetMessages(convID int64) ([]Message, error) {
	rows, err := r.db.Query(
		`SELECT id, conversation_id, role, content, COALESCE(provider,''), COALESCE(model,''),
				COALESCE(latency_ms,0), created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`,
		convID,
	)

// SAU:
func (r *ConversationRepo) GetMessages(convID, userID int64) ([]Message, error) {
	// Verify conversation belongs to user
	var ownerID int64
	err := r.db.QueryRow(`SELECT user_id FROM conversations WHERE id = ?`, convID).Scan(&ownerID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	if ownerID != userID {
		return nil, fmt.Errorf("forbidden: conversation does not belong to user")
	}
	rows, err := r.db.Query(
		`SELECT id, conversation_id, role, content, COALESCE(provider,''), COALESCE(model,''),
				COALESCE(latency_ms,0), created_at
		 FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`,
		convID,
	)
```

- [ ] **Step 5: Fix handler `handleConversationsMessages` — use claims**

Trong `go-engine/api/handlers_conversations.go` (line 62):

```go
// TRƯỚC:
_, rpcErr := r.requireAuth(req)
// ...
msgs, err := r.conversations.GetMessages(p.ConversationID)

// SAU:
claims, rpcErr := r.requireAuth(req)
// ...
msgs, err := r.conversations.GetMessages(p.ConversationID, claims.UserID)
```

- [ ] **Step 6: Fix handler `handleQuery` — pass userID to AddMessage**

Trong `go-engine/api/handlers_query.go` (lines 147-149):

```go
// TRƯỚC:
if p.ConversationID > 0 {
	_ = r.conversations.AddMessage(p.ConversationID, "user", finalInput, "", "", 0)
	_ = r.conversations.AddMessage(p.ConversationID, "assistant", sb.String(), providerName, modelName, latency)
}

// SAU:
if p.ConversationID > 0 {
	if err := r.conversations.AddMessage(p.ConversationID, claims.UserID, "user", finalInput, "", "", 0); err != nil {
		log.Printf("addMessage user error: %v", err)
	}
	if err := r.conversations.AddMessage(p.ConversationID, claims.UserID, "assistant", sb.String(), providerName, modelName, latency); err != nil {
		log.Printf("addMessage assistant error: %v", err)
	}
}
```

Đảm bảo `claims` đã được lấy từ `requireAuth` ở đầu handler (kiểm tra context hiện tại).

- [ ] **Step 7: Fix tất cả caller khác của AddMessage/GetMessages**

Search toàn bộ codebase cho `AddMessage(` và `GetMessages(` — update tất cả caller truyền thêm `userID`.

Run: `cd /home/dev/open-prompt/go-engine && grep -rn "\.AddMessage\|\.GetMessages" --include="*.go"`

- [ ] **Step 8: Run tests**

Run: `cd /home/dev/open-prompt/go-engine && go test ./db/repos/ -run TestAddMessage -v && go test ./db/repos/ -run TestGetMessages -v`
Expected: ALL PASS

- [ ] **Step 9: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`
Expected: No compilation errors

- [ ] **Step 10: Commit**

```bash
cd /home/dev/open-prompt
git add go-engine/db/repos/conversation_repo.go go-engine/db/repos/conversation_repo_test.go go-engine/api/handlers_conversations.go go-engine/api/handlers_query.go
git commit -m "security: fix CRITICAL IDOR — add user ownership check to conversation operations"
```

---

## Task 2: Fix XSS trong MarkdownRenderer

**Files:**
- Modify: `src/components/overlay/MarkdownRenderer.tsx`
- Modify: `package.json` (add dependencies)

- [ ] **Step 1: Install react-markdown và rehype-sanitize**

Run: `cd /home/dev/open-prompt && npm install react-markdown rehype-sanitize`

- [ ] **Step 2: Rewrite MarkdownRenderer dùng react-markdown**

Thay thế toàn bộ nội dung `src/components/overlay/MarkdownRenderer.tsx`.

**Quan trọng:** Giữ nguyên prop name `text` và named export `export function` để không break caller (`ResponsePanel.tsx` line 87: `<MarkdownRenderer text={text} />`).

```tsx
import React from 'react'
import ReactMarkdown from 'react-markdown'
import rehypeSanitize from 'rehype-sanitize'

interface Props {
  text: string
}

/** MarkdownRenderer render Markdown an toàn với react-markdown + rehype-sanitize */
export function MarkdownRenderer({ text }: Props) {
  return (
    <div className="markdown-body text-sm text-white/90 leading-relaxed">
      <ReactMarkdown
        rehypePlugins={[rehypeSanitize]}
        components={{
          h1: ({ children }) => (
            <h1 className="text-white font-bold mt-3 mb-1">{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className="text-white font-semibold mt-3 mb-1">{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="text-white font-semibold text-sm mt-3 mb-1">{children}</h3>
          ),
          p: ({ children }) => (
            <p className="mb-2">{children}</p>
          ),
          code: ({ className, children, ...props }) => {
            const isBlock = className?.startsWith('language-')
            if (isBlock) {
              return (
                <pre className="bg-black/30 rounded p-3 my-2 overflow-x-auto">
                  <code className="text-xs font-mono text-indigo-300" {...props}>
                    {children}
                  </code>
                </pre>
              )
            }
            return (
              <code className="bg-white/10 px-1 py-0.5 rounded text-xs font-mono text-indigo-300" {...props}>
                {children}
              </code>
            )
          },
          strong: ({ children }) => (
            <strong className="text-white font-semibold">{children}</strong>
          ),
          em: ({ children }) => (
            <em className="text-white/80 italic">{children}</em>
          ),
          ul: ({ children }) => (
            <ul className="list-disc list-inside mb-2 space-y-1">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="list-decimal list-inside mb-2 space-y-1">{children}</ol>
          ),
          li: ({ children }) => (
            <li className="text-white/80">{children}</li>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-white/20 pl-3 my-2 text-white/60">{children}</blockquote>
          ),
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  )
}
```

- [ ] **Step 3: Verify build**

Run: `cd /home/dev/open-prompt && npm run build`
Expected: No TypeScript or build errors

- [ ] **Step 4: Test manually — verify XSS payload is sanitized**

Kiểm tra render output với các input:
- `# <script>alert(1)</script>` → phải render text thuần, không execute script
- `**bold** _italic_ \`code\`` → phải render đúng formatting
- `` ```js\nconsole.log('hi')\n``` `` → phải render code block

- [ ] **Step 5: Commit**

```bash
cd /home/dev/open-prompt
git add src/components/overlay/MarkdownRenderer.tsx package.json package-lock.json
git commit -m "security: fix XSS — replace custom markdown renderer with react-markdown + rehype-sanitize"
```

---

## Task 3: Fix Goroutine Leak trong RateLimiter

**Files:**
- Modify: `go-engine/api/ratelimit.go:26-43`
- Create: `go-engine/api/ratelimit_test.go`
- Modify: `go-engine/api/router.go` (cleanup on shutdown)

- [ ] **Step 1: Write failing test cho RateLimiter Stop**

```go
// go-engine/api/ratelimit_test.go
package api

import (
	"testing"
	"time"
)

func TestRateLimiter_StopTerminatesGoroutine(t *testing.T) {
	rl := NewRateLimiter()

	// Goroutine should be running
	time.Sleep(10 * time.Millisecond)

	// Stop should not panic and should terminate goroutine
	rl.Stop()

	// Calling Stop again should not panic (idempotent)
	rl.Stop()
}

func TestRateLimiter_AllowAfterStop(t *testing.T) {
	rl := NewRateLimiter()
	rl.Stop()

	// Allow should still work after stop (just no cleanup)
	allowed := rl.Allow("test.method", "caller1")
	if !allowed {
		t.Fatal("expected Allow to return true for first call")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/dev/open-prompt/go-engine && go test ./api/ -run TestRateLimiter_Stop -v`
Expected: FAIL — `Stop` method does not exist

- [ ] **Step 3: Add stopCh and Stop() to RateLimiter**

Trong `go-engine/api/ratelimit.go`, sửa struct và constructor:

```go
// Thêm field vào struct RateLimiter (khoảng line 26):
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	stopCh  chan struct{}
	stopped sync.Once
}

// Sửa NewRateLimiter (khoảng line 33):
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		stopCh:  make(chan struct{}),
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanup()
			case <-rl.stopCh:
				return
			}
		}
	}()
	return rl
}

// Thêm method Stop():
func (rl *RateLimiter) Stop() {
	rl.stopped.Do(func() {
		close(rl.stopCh)
	})
}
```

Thêm `"sync"` vào import nếu chưa có.

- [ ] **Step 4: Run tests**

Run: `cd /home/dev/open-prompt/go-engine && go test ./api/ -run TestRateLimiter -v`
Expected: ALL PASS

- [ ] **Step 5: Wire Stop() vào Server.Close()**

`Server.Close()` nằm ở `go-engine/api/server.go` line 93. Server có field `router *Router`, và `Router` có field `rateLimiter *RateLimiter`. Sửa `Server.Close()`:

```go
// go-engine/api/server.go line 93
// TRƯỚC:
func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// SAU:
func (s *Server) Close() {
	if s.router != nil && s.router.rateLimiter != nil {
		s.router.rateLimiter.Stop()
	}
	if s.listener != nil {
		s.listener.Close()
	}
}
```

- [ ] **Step 6: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`
Expected: No errors

- [ ] **Step 7: Commit**

```bash
cd /home/dev/open-prompt
git add go-engine/api/ratelimit.go go-engine/api/ratelimit_test.go go-engine/api/router.go
git commit -m "fix: goroutine leak — add Stop() to RateLimiter with graceful shutdown"
```

---

## Task 4: Fix localStorage Inconsistency

**Files:**
- Modify: `src/components/overlay/CommandInput.tsx:25`
- Modify: `src/components/prompts/PromptList.tsx:23,40`
- Modify: `src/components/prompts/PromptEditor.tsx:43`
- Modify: `src/components/skills/SkillEditor.tsx:37`

- [ ] **Step 1: Verify authStore token selector**

Đọc `src/store/authStore.ts` để confirm interface. Store dùng Zustand persist với key `op-auth`, expose `token` field.

- [ ] **Step 2: Fix CommandInput.tsx**

Trong `src/components/overlay/CommandInput.tsx`:

```tsx
// TRƯỚC (line 25):
const token = localStorage.getItem('auth_token')

// SAU:
// Thêm import ở đầu file:
import { useAuthStore } from '../../store/authStore'

// Trong component, thay bằng:
const token = useAuthStore((s) => s.token)
```

Lưu ý: `token` giờ có thể `null` thay vì `string | null`, cần verify logic downstream xử lý đúng.

- [ ] **Step 3: Fix PromptList.tsx**

Trong `src/components/prompts/PromptList.tsx`:

```tsx
// Thêm import:
import { useAuthStore } from '../../store/authStore'

// Trong component function, khai báo 1 lần:
const token = useAuthStore((s) => s.token)

// Xoá cả 2 dòng localStorage (line 23 và 40):
// const token = localStorage.getItem('auth_token')  ← xoá
```

- [ ] **Step 4: Fix PromptEditor.tsx**

Trong `src/components/prompts/PromptEditor.tsx`:

```tsx
// Thêm import ở đầu file:
import { useAuthStore } from '../../store/authStore'

// QUAN TRỌNG: Đặt hook ở TOP LEVEL của component function (trước handleSubmit),
// KHÔNG đặt bên trong handleSubmit (vi phạm Rules of Hooks).
// Ví dụ:
export default function PromptEditor(...) {
  const token = useAuthStore((s) => s.token)  // ← ĐẶT Ở ĐÂY
  // ...
  const handleSubmit = async () => {
    // Xoá dòng: const token = localStorage.getItem('auth_token')  ← xoá line 43
    if (!token) return  // guard đã có sẵn, giữ nguyên
    // ... phần còn lại giữ nguyên
  }
}
```

- [ ] **Step 5: Fix SkillEditor.tsx**

Trong `src/components/skills/SkillEditor.tsx`:

```tsx
// Thêm import ở đầu file:
import { useAuthStore } from '../../store/authStore'

// Đặt hook ở TOP LEVEL của component function:
export default function SkillEditor(...) {
  const token = useAuthStore((s) => s.token)  // ← TOP LEVEL
  // ...
  const handleSave = async () => {
    // Xoá dòng: const token = localStorage.getItem('auth_token')  ← xoá line 37
    if (!token) { setError('Chưa đăng nhập'); return }  // ← THÊM null guard
    // ... phần còn lại giữ nguyên, token được dùng trong payload
  }
}
```

**Quan trọng:** Thêm null guard `if (!token)` vì `useAuthStore` trả về `string | null`, và `token` được truyền thẳng vào payload (line 42). Nếu `null`, backend sẽ reject auth.

- [ ] **Step 6: Search for remaining localStorage.getItem('auth_token')**

Run: `cd /home/dev/open-prompt && grep -rn "localStorage.getItem.*auth_token" --include="*.tsx" --include="*.ts" src/`
Expected: No results (tất cả đã được thay thế)

- [ ] **Step 7: Build check**

Run: `cd /home/dev/open-prompt && npm run build`
Expected: No errors

- [ ] **Step 8: Commit**

```bash
cd /home/dev/open-prompt
git add src/components/overlay/CommandInput.tsx src/components/prompts/PromptList.tsx src/components/prompts/PromptEditor.tsx src/components/skills/SkillEditor.tsx
git commit -m "fix: replace direct localStorage access with useAuthStore for consistent auth state"
```

---

## Final Verification

- [ ] **Step 1: Full backend build**

Run: `cd /home/dev/open-prompt/go-engine && go build ./... && go test ./...`
Expected: Build OK, all tests pass

- [ ] **Step 2: Full frontend build**

Run: `cd /home/dev/open-prompt && npm run build`
Expected: Build OK, no warnings

- [ ] **Step 3: Final commit (nếu cần fix gì thêm)**

```bash
git log --oneline -5
```

Verify 4 commits đã được tạo cho 4 CRITICAL fixes.
