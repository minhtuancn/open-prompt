# Open Prompt v1.1 — Stabilization & Polish Design

**Ngày:** 2026-03-24
**Mục tiêu:** Review toàn diện backend (Go) + frontend (React), fix bugs, tối ưu performance, tư vấn roadmap
**Đối tượng:** Developer / kỹ sư phần mềm
**Ưu tiên:** Ổn định & polish trước tính năng mới
**Hướng tiếp cận:** Fix theo mức độ nghiêm trọng (CRITICAL → HIGH → MEDIUM → LOW)

---

## Tổng quan kết quả review

| Mức độ | Backend | Frontend | Tổng |
|--------|---------|----------|------|
| CRITICAL | 2 | 2 | **4** |
| HIGH | 7 | 8 | **15** |
| MEDIUM | 18+ | 18 | **36+** |
| LOW | 5+ | 7 | **12+** |

---

## Section 1: CRITICAL Issues

### 1.1 IDOR trong Conversation (Backend)

**File:** `go-engine/db/repos/conversation_repo.go`, `go-engine/api/handlers_conversations.go`, `go-engine/api/handlers_query.go`

**Vấn đề:** `GetMessages()`, `AddMessage()` không kiểm tra `user_id`. Handler gọi `conversations.GetMessages(conversationID)` mà không verify ownership. Kẻ tấn công có thể đọc/ghi conversation của user khác bằng cách đoán ID.

**Lưu ý quan trọng:** `handleConversationsMessages` (line 62) discard `claims` từ `requireAuth` — phải fix cả handler-level để truyền `claims.UserID` xuống repo. Nếu chỉ fix repo mà không fix handler thì IDOR vẫn tồn tại.

**Fix:**
- Thêm tham số `userID` vào tất cả method của `ConversationRepo`
- Thêm `AND user_id = ?` vào mọi query liên quan
- `handleConversationsMessages`: sử dụng `claims.UserID` từ `requireAuth` (hiện đang bỏ qua)
- `handleQuery` (handlers_query.go): validate ownership trước khi `AddMessage()` khi `ConversationID > 0`
- Cả read path (GetMessages) và write path (AddMessage) đều cần fix đồng thời

### 1.2 XSS trong MarkdownRenderer (Frontend)

**File:** `src/components/overlay/MarkdownRenderer.tsx`

**Vấn đề:** Có 2 vector XSS chính:
1. **Heading branch (lines 39-43)** bypass hoàn toàn `inlineFormat()`/`escapeHtml()` — inject `processed.slice(N)` trực tiếp vào HTML string mà không escape. Input như `# <script>alert(1)</script>` sẽ render raw HTML.
2. **`escapeHtml()` thiếu escape quote** — không escape `"` và `'`, cho phép attribute injection trong một số trường hợp.

**Fix:**
- Thay thế bằng thư viện `react-markdown` + `rehype-sanitize` (khuyến nghị)
- Nếu giữ custom renderer: heading branch phải gọi `inlineFormat()` trước khi inject, mở rộng `escapeHtml()` để escape cả `"` và `'`, dùng DOMPurify để sanitize output cuối cùng

### 1.3 Goroutine Leak trong RateLimiter (Backend)

**File:** `go-engine/api/ratelimit.go` (lines 37-43)

**Vấn đề:** Cleanup goroutine trong `NewRateLimiter` không có `stopCh` hay method `Stop()`. Goroutine chạy vĩnh viễn, leak khi server shutdown hoặc trong tests.

**Lưu ý:** `health_checker.go` và `token_expiry_watcher.go` đã có `Stop()` với `stopCh` pattern — chỉ `ratelimit.go` còn thiếu.

**Fix:**
- Thêm `stopCh chan struct{}` vào `RateLimiter`
- Thêm method `Stop()` close channel
- Cleanup goroutine select trên `stopCh`
- Gọi `Stop()` trong `Server.Close()`

### 1.4 localStorage Inconsistency (Frontend)

**File:** `src/components/prompts/PromptList.tsx`, `PromptEditor.tsx`, `SkillEditor.tsx`, `CommandInput.tsx`

**Vấn đề:** Nhiều component đọc `localStorage.getItem('auth_token')` trực tiếp thay vì dùng `useAuthStore`. Gây race condition và stale token.

**Fix:**
- Thống nhất dùng `useAuthStore((s) => s.token)` ở mọi nơi
- Xoá tất cả localStorage access trực tiếp trong components

### 1.5 (Chuyển xuống HIGH) OAuth Placeholder (Backend)

**File:** `go-engine/api/handlers_oauth.go`

**Vấn đề:** Hardcoded "PLACEHOLDER" cho OAuth credentials. Các endpoint trả message rõ ràng là placeholder (ví dụ: "cần GitHub OAuth App ID thật"), `handleOAuthPoll` trả `"done": false` vĩnh viễn nên không thực sự hoàn thành flow. Tuy nhiên `handleOAuthFinish` trả `"ok": true` kèm placeholder message — gây misleading.

**Mức độ thực tế:** HIGH (misleading UX, không phải security vulnerability vì không grant access)

**Fix:**
- `handleOAuthFinish`: return `"ok": false` với error message thay vì `"ok": true`
- Hoặc disable endpoint hoàn toàn cho đến khi implement xong
- Thêm: fix `rand.Read(buf)` (line 103) không check error return

---

## Section 2: HIGH Issues

### Backend (6 issues)

**2.1 Thiếu validation ConversationID**
- **File:** `go-engine/api/handlers_query.go`
- Chỉ check `> 0` mà không verify conversation tồn tại
- Fix: validate existence + ownership trước khi `AddMessage()`

**2.2 Weak API key validation**
- **File:** `go-engine/provider/token_manager.go`
- Gemini chỉ check length, OpenAI chỉ check prefix `sk-`
- Fix: validation cụ thể hơn theo từng provider, cân nhắc test call

**2.3 Silenced errors trong streaming**
- **File:** `go-engine/api/handlers_query.go`
- Ignore lỗi `SendNotification` với `_`
- Fix: log tất cả lỗi, return early nếu connection đã mất

**2.4 Silenced errors trong lưu history/conversation**
- **File:** `go-engine/api/handlers_query.go`
- Bỏ qua lỗi khi gọi `AddMessage()` và `Insert()`
- Fix: log errors, trả partial success response

**2.5 (Đã xác minh: không phải N+1)** ~~N+1 query trong handleProvidersList~~
- **File:** `go-engine/api/handlers_providers.go`
- **Cập nhật:** Sau khi verify code, handler thực tế chỉ có 1 DB query + in-memory map build + O(1) lookups. Không có N+1 SQL query. Health checks được xử lý bởi `HealthChecker` service riêng.
- **Thay thế bằng:** OAuth Placeholder (chuyển từ CRITICAL xuống HIGH — xem 1.5)

**2.6 Thiếu string length validation**
- **File:** Tất cả `handlers_*.go`
- Prompt title, content, marketplace text không giới hạn length
- Fix: define `MaxTitleLen`, `MaxContentLen`, validate ở handler layer

### Frontend (8 issues)

**2.7 DeviceFlowDialog unmount safety**
- **File:** `src/components/overlay/DeviceFlowDialog.tsx`
- Interval cleanup function đã đúng, nhưng `setStatus()` và `onComplete()` vẫn được gọi sau khi component unmount trong async `invoke` callback. Thiếu `isMounted` guard.
- Fix: thêm `isMountedRef = useRef(true)`, set `false` trong cleanup, check trước mọi state update và callback

**2.8 useEngine race condition**
- **File:** `src/hooks/useEngine.ts`
- Listener setup trước khi `callEngine()` gọi, error có thể bị miss
- Fix: wrap trong try-catch đúng cách, đảm bảo thứ tự setup

**2.9 ResponsePanel memory leaks**
- **File:** `src/components/overlay/ResponsePanel.tsx`
- Nhiều `setTimeout` không cleanup khi unmount
- Fix: track tất cả timer refs, clear trong useEffect cleanup

**2.10 ProvidersTab orphaned timers**
- **File:** `src/components/settings/ProvidersTab.tsx`
- `timeoutRef` không init đúng, không clear timer cũ trước khi set mới
- Fix: init `null`, always clear trước khi set

**2.11 Thiếu ARIA labels**
- **File:** Tất cả interactive components
- Buttons, menus, lists thiếu `aria-label`, `role`, `aria-expanded`
- Fix: thêm accessibility attributes

**2.12 Keyboard navigation không đầy đủ**
- **File:** `src/components/overlay/SlashMenu.tsx`
- Chỉ handle keyboard khi menu visible, không có focus trap
- Fix: implement proper focus management

**2.13 Silent failures**
- **File:** Nhiều components dùng `.catch(console.error)`
- User không thấy lỗi, UI stuck ở loading state
- Fix: set error state và show error message

**2.14 Responsive design thiếu**
- **File:** `OnboardingWizard.tsx`, `LoginScreen.tsx`
- Hardcoded width không adapt màn hình nhỏ
- Fix: thêm `max-w-full` và responsive padding

---

## Section 3: MEDIUM Issues

### 3A. Security & Validation

- **Rate limiter keying** — dùng remote address thay vì user ID từ JWT. Fix: ưu tiên user ID nếu có JWT
- **Template injection trong PromptBuilder** — `text/template.Parse()` với user input không giới hạn size. Fix: cap max template size
- **API key exposure** — frontend truyền key plaintext qua IPC. Fix: thêm notice, xem xét encrypt in transit
- **Unvalidated prompt input** — frontend validate nhưng backend cũng cần validate lại
- **Error messages leak info** — hiển thị lỗi raw cho user. Fix: wrap generic message

### 3B. Database & Performance

- **Thiếu request limit** — `Limit`/`Offset` không cap. Fix: `const MaxLimit = 1000`
- **History aggregation chậm** — `SummaryByPeriod` group by mỗi lần gọi. Fix: dùng bảng pre-aggregated
- **Thiếu indexes** — `prompts.user_id`, `skills.user_id`, `settings.user_id`
- **DELETE CASCADE sai** — `history.user_id` dùng `SET NULL` thay vì `CASCADE` (xem `go-engine/db/migrations/001_init.sql` line 61)
- **Migration version tracking** — `IF NOT EXISTS` không scale. Fix: thêm bảng version tracking
- **Provider list fetch trùng** — Frontend fetch 3 lần từ 3 component. Fix: cache trong Zustand store

### 3C. Architecture & Code Quality

- **Inconsistent repo patterns** — Một số repo truyền `userID`, một số không. Fix: thống nhất convention
- **Thiếu request validation layer** — Mỗi handler tự decode/validate. Fix: centralized middleware
- **2 provider registry systems** — `provider.Registry` (quản lý tokens/credentials) và `model/providers.Registry` (quản lý provider instances). Phục vụ mục đích khác nhau nhưng tên gây nhầm lẫn. Fix: rename cho rõ ràng (ví dụ: `TokenStore` vs `ProviderRegistry`) thay vì merge
- **Inconsistent state management** — Mix local state và Zustand. Fix: document rule
- **Global RPC access** — CommandInput dùng `window.__rpc?.call()` (line 27) trong cùng `useEffect` với localStorage anti-pattern. Fix cả hai cùng lúc: thống nhất qua `callEngine()` + `useAuthStore`
- **Lost error context** — `if err != nil || existing == nil` gộp 2 loại lỗi. Fix: tách check
- **Weak typing RPC calls** — Unsafe cast. Fix: tạo typed RPC interface

### 3D. UX & i18n

- **i18n không đầy đủ** — Nhiều chuỗi Vietnamese hardcode. Fix: wrap qua `useI18n().t()`
- **Focus management thiếu** — Switch tab không quản lý focus
- **Dangerous cast trong SettingsLayout** — `useState<any>`. Fix: proper union type
- **App.tsx fallback-retry event** — `CustomEvent` không validate. Fix: dùng Zustand action
- **Socket path hardcoded** — `/tmp/open-prompt.sock`. Fix: dùng `$XDG_RUNTIME_DIR`
- **Thiếu DB path env override** — Fix: thêm `OP_DB_PATH` env variable
- **Inconsistent error response codes** — Fix: document mapping

---

## Section 4: LOW Issues

- Regexp compile trong render (CommandInput) — move ra module-level constant
- Thiếu memoization list items — `React.memo` cho list > 20 items
- Zustand re-renders (App.tsx) — consolidate selectors
- Magic numbers (`5000ms`, `3000ms`) — named constants
- Dark mode media query thiếu
- Tailwind opacity inconsistency
- Import ordering — ESLint import-sort
- Bundle analysis thiếu — `vite-plugin-visualizer`
- Zustand devtools middleware cho dev mode
- Scanner buffer check (`server.go`)
- `rand.Read` unchecked error trong `handlers_oauth.go` line 103

---

## Section 5: Tư vấn thêm cho Roadmap v1.1

### 5.1 Structured Logging
Hiện log rời rạc. Nên dùng `slog` (Go 1.21+) với levels rõ ràng.

### 5.2 Graceful Shutdown Flow
Implement: stop accepting connections -> drain active requests -> stop background goroutines -> close DB -> exit.

### 5.3 Typed RPC Layer
Tạo shared type definitions giữa frontend và backend. Generate từ JSON schema hoặc code generation.

### 5.4 React Error Boundary
Wrap top-level components để prevent app crash khi component lỗi.

### 5.5 Health Check cho Frontend
Heartbeat mechanism để frontend biết backend còn sống.

### 5.6 Test Coverage
Bắt đầu với:
- Unit test cho repos (database layer)
- Unit test cho critical handlers (query, auth, conversation)
- Component test cho overlay components

---

## Thứ tự thực hiện

| Phase | Nội dung | Ưu tiên |
|-------|----------|---------|
| Phase 1 | CRITICAL fixes (Section 1) | Cao nhất |
| Phase 2 | HIGH fixes (Section 2) | Ngay sau Phase 1 |
| Phase 3 | MEDIUM fixes (Section 3) | Theo nhóm chủ đề |
| Phase 4 | LOW fixes + Roadmap items (Section 4-5) | Khi có thời gian |
