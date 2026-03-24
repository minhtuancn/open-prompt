# Stabilization Phase 2: HIGH Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 11 HIGH issues across backend (Go) and frontend (React) — error handling, validation, memory safety, accessibility, UX.

**Architecture:** Tasks are independent. Backend and frontend fixes can run in parallel. Grouped by theme for efficiency.

**Tech Stack:** Go 1.22+, React 18, TypeScript, Zustand, TailwindCSS

**Spec:** `docs/superpowers/specs/2026-03-24-stabilization-polish-design.md`

**Note:** 3 issues from spec verified as already resolved:
- 2.1 ConversationID validation — implicit via Phase 1 IDOR fix (AddMessage now checks ownership+existence)
- 2.10 ProvidersTab timers — already properly managed with timeoutRef
- 2.12 Keyboard navigation — already properly implemented in SlashMenu

---

## File Structure

| Task | Files | Action |
|------|-------|--------|
| 1. Streaming errors | `go-engine/api/handlers_query.go` | Modify |
| 2. History errors | `go-engine/api/handlers_query.go` | Modify (same file as T1) |
| 3. API key validation | `go-engine/provider/token_manager.go` | Modify |
| 4. String length validation | `go-engine/api/handlers_prompts.go` | Modify |
| | `go-engine/api/handlers_skills.go` | Modify |
| | `go-engine/api/validation.go` | Create |
| 5. OAuth placeholder | `go-engine/api/handlers_oauth.go` | Modify |
| 6. DeviceFlowDialog | `src/components/overlay/DeviceFlowDialog.tsx` | Modify |
| 7. useEngine race condition | `src/hooks/useEngine.ts` | Modify |
| 8. ResponsePanel leaks | `src/components/overlay/ResponsePanel.tsx` | Modify |
| 9. Silent failures | 7 component files | Modify |
| 10. ARIA labels | 5 component files | Modify |
| 11. Responsive design | `src/components/onboarding/OnboardingWizard.tsx` | Modify |
| | `src/components/auth/LoginScreen.tsx` | Modify |

---

## Task 1: Fix Silenced Streaming Errors

**Files:**
- Modify: `go-engine/api/handlers_query.go:98,118,132`

**Problem:** 3 `_ = SendNotification(...)` calls silently drop connection errors.

- [ ] **Step 1: Add error logging to all SendNotification calls**

Replace all 3 instances of `_ = SendNotification(conn, ...)` with:

```go
if err := SendNotification(conn, "stream.chunk", ...); err != nil {
    log.Printf("ERROR send stream chunk: %v", err)
    return // stop streaming if connection lost
}
```

Specifically:
- **Line 98** (normal chunk): log + continue (don't return, chunks can be retried)
- **Line 118** (error/fallback notification): log error
- **Line 132** (done notification): log error

For line 98 (inside onChunk callback), check if SendNotification can indicate a broken connection. If so, set a flag to stop further streaming.

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add go-engine/api/handlers_query.go
git commit -m "fix: log streaming notification errors instead of silencing"
```

---

## Task 2: Fix Silenced History Insert Errors

**Files:**
- Modify: `go-engine/api/handlers_query.go:120-127,137-145`

**Problem:** 2 `_ = r.history.Insert(...)` calls silently drop database errors.

- [ ] **Step 1: Add error logging to history.Insert calls**

Replace both instances:

```go
// Line 120 (error case):
if err := r.history.Insert(repos.InsertHistoryInput{...}); err != nil {
    log.Printf("ERROR insert history (error case): %v", err)
}

// Line 137 (success case):
if err := r.history.Insert(repos.InsertHistoryInput{...}); err != nil {
    log.Printf("ERROR insert history (success case): %v", err)
}
```

Don't return error to client — history insert failure shouldn't break the streaming response.

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add go-engine/api/handlers_query.go
git commit -m "fix: log history insert errors instead of silencing"
```

---

## Task 3: Improve API Key Validation

**Files:**
- Modify: `go-engine/provider/token_manager.go:27-49`

**Problem:** Gemini only checks length >= 10, OpenAI only checks `sk-` prefix. Insufficient validation.

- [ ] **Step 1: Improve ValidateKeyFormat**

Update the validation per provider:

```go
func ValidateKeyFormat(provider, key string) error {
    if key == "" {
        return fmt.Errorf("API key không được rỗng")
    }
    switch provider {
    case "ollama":
        return nil // Ollama không cần key
    case "anthropic":
        if !strings.HasPrefix(key, "sk-ant-") {
            return fmt.Errorf("Anthropic key phải bắt đầu bằng 'sk-ant-'")
        }
        if len(key) < 40 {
            return fmt.Errorf("Anthropic key quá ngắn")
        }
    case "openai":
        if !strings.HasPrefix(key, "sk-") {
            return fmt.Errorf("OpenAI key phải bắt đầu bằng 'sk-'")
        }
        if len(key) < 20 {
            return fmt.Errorf("OpenAI key quá ngắn")
        }
    case "gemini":
        if len(key) < 30 {
            return fmt.Errorf("Gemini key quá ngắn (cần ít nhất 30 ký tự)")
        }
        // Gemini keys are alphanumeric + hyphens + underscores
        for _, c := range key {
            if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
                return fmt.Errorf("Gemini key chứa ký tự không hợp lệ")
            }
        }
    case "copilot":
        if len(key) < 10 {
            return fmt.Errorf("Copilot token quá ngắn")
        }
    default:
        // Custom gateway — basic length check
        if len(key) < 5 {
            return fmt.Errorf("API key quá ngắn")
        }
    }
    return nil
}
```

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add go-engine/provider/token_manager.go
git commit -m "fix: strengthen API key format validation per provider"
```

---

## Task 4: Add String Length Validation

**Files:**
- Create: `go-engine/api/validation.go`
- Modify: `go-engine/api/handlers_prompts.go:52-54,93`
- Modify: `go-engine/api/handlers_skills.go:68`

**Problem:** Prompt title/content and skill fields have no length limits. Can accept unbounded input.

- [ ] **Step 1: Create validation helper**

Create `go-engine/api/validation.go`:

```go
package api

// Giới hạn kích thước input
const (
    MaxTitleLen   = 200
    MaxContentLen = 50000
    MaxTagsLen    = 500
    MaxNameLen    = 32
)

// truncateString cắt chuỗi nếu vượt quá maxLen
func truncateString(s string, maxLen int) string {
    if len(s) > maxLen {
        return s[:maxLen]
    }
    return s
}
```

- [ ] **Step 2: Add length validation to handlers_prompts.go**

After the empty check (line 54), add truncation:

```go
// Create handler — after empty check:
p.Title = truncateString(p.Title, MaxTitleLen)
p.Content = truncateString(p.Content, MaxContentLen)

// Update handler — after empty check (line 93):
p.Title = truncateString(p.Title, MaxTitleLen)
p.Content = truncateString(p.Content, MaxContentLen)
```

- [ ] **Step 3: Add length validation to handlers_skills.go**

After the name empty check (line 68):

```go
p.Name = truncateString(p.Name, MaxNameLen)
p.PromptText = truncateString(p.PromptText, MaxContentLen)
p.Tags = truncateString(p.Tags, MaxTagsLen)
```

- [ ] **Step 4: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add go-engine/api/validation.go go-engine/api/handlers_prompts.go go-engine/api/handlers_skills.go
git commit -m "fix: add string length validation for prompts and skills input"
```

---

## Task 5: Fix OAuth Placeholder

**Files:**
- Modify: `go-engine/api/handlers_oauth.go:71-74,103`

**Problem:** `handleOAuthFinish` returns `"ok": true` for placeholder. `rand.Read` error unchecked.

- [ ] **Step 1: Fix handleOAuthFinish to return ok:false**

```go
// Line 71-74, change to:
return map[string]interface{}{
    "ok":      false,
    "message": "OAuth chưa được cấu hình — cần Client ID thật",
}, nil
```

- [ ] **Step 2: Fix rand.Read error handling**

```go
// Line 103, change to:
if _, err := rand.Read(buf); err != nil {
    return "", "", // caller should handle empty verifier
}
```

Actually, since `generatePKCE` returns `(verifier, challenge string)`, we need to propagate the error:

```go
func generatePKCE() (verifier, challenge string, err error) {
    buf := make([]byte, 32)
    if _, err = rand.Read(buf); err != nil {
        return "", "", fmt.Errorf("PKCE generate failed: %w", err)
    }
    verifier = base64.RawURLEncoding.EncodeToString(buf)
    h := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(h[:])
    return verifier, challenge, nil
}
```

Update all callers of `generatePKCE()` to handle the error.

- [ ] **Step 3: Build check**

Run: `cd /home/dev/open-prompt/go-engine && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add go-engine/api/handlers_oauth.go
git commit -m "fix: OAuth placeholder returns ok:false, handle rand.Read error in PKCE"
```

---

## Task 6: Fix DeviceFlowDialog Unmount Safety

**Files:**
- Modify: `src/components/overlay/DeviceFlowDialog.tsx`

**Problem:** State updates (`setStatus`, `setError`, `onComplete`) can fire after unmount.

- [ ] **Step 1: Add isMounted guard**

Add `useRef` for mounted state:

```tsx
const isMountedRef = useRef(true)

useEffect(() => {
  return () => {
    isMountedRef.current = false
    if (intervalRef.current) clearInterval(intervalRef.current)
  }
}, [])
```

Wrap all state updates inside the interval callback:

```tsx
if (result.done) {
  if (intervalRef.current) clearInterval(intervalRef.current)
  if (!isMountedRef.current) return
  if (result.error) {
    setStatus('error')
    setError(result.error)
  } else {
    setStatus('success')
    onComplete()
  }
}
```

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 3: Commit**

```bash
git add src/components/overlay/DeviceFlowDialog.tsx
git commit -m "fix: add isMounted guard to DeviceFlowDialog polling"
```

---

## Task 7: Fix useEngine Race Condition

**Files:**
- Modify: `src/hooks/useEngine.ts`

**Problem:** Listener is set up before `callEngine` is invoked. If callEngine emits events before listener is ready, events are lost.

- [ ] **Step 1: Restructure to ensure listener is ready before call**

The current order is actually correct (listener THEN call), but the issue is that `callEngine` errors aren't properly caught. Wrap in try-catch:

```tsx
export async function streamQuery(
  params: { token: string; input: string; model?: string; provider?: string },
  onChunk: (chunk: string) => void,
  onDone: () => void,
  onError: (err: string, fallbackProviders?: string[]) => void,
): Promise<void> {
  let unlisten: (() => void) | null = null

  try {
    unlisten = await listen<StreamChunkPayload>(
      'stream-chunk',
      (event) => {
        const { delta, done, error, error_message, fallback_providers } = event.payload
        if (error) {
          onError(error_message || error, fallback_providers)
          unlisten?.()
          return
        }
        if (done) {
          onDone()
          unlisten?.()
          return
        }
        onChunk(delta)
      }
    )

    await callEngine('query.stream', params)
  } catch (e) {
    unlisten?.()
    onError(String(e))
  }
}
```

Key changes:
- `callEngine` is now `await`ed (was fire-and-forget)
- If callEngine throws, unlisten is called to clean up
- `unlisten` is checked for null before calling

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 3: Commit**

```bash
git add src/hooks/useEngine.ts
git commit -m "fix: properly await callEngine and handle errors in streamQuery"
```

---

## Task 8: Fix ResponsePanel Timer Leaks

**Files:**
- Modify: `src/components/overlay/ResponsePanel.tsx:47,57`

**Problem:** Two `setTimeout(() => setCopied(false), 2000)` calls are not tracked in refs.

- [ ] **Step 1: Add copyTimerRef and track both timeouts**

```tsx
// Add ref alongside existing injectTimerRef:
const copyTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

// Update cleanup useEffect:
useEffect(() => {
  return () => {
    if (injectTimerRef.current) clearTimeout(injectTimerRef.current)
    if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
  }
}, [])

// Replace line 47:
if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
copyTimerRef.current = setTimeout(() => setCopied(false), 2000)

// Replace line 57 (same pattern):
if (copyTimerRef.current) clearTimeout(copyTimerRef.current)
copyTimerRef.current = setTimeout(() => setCopied(false), 2000)
```

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 3: Commit**

```bash
git add src/components/overlay/ResponsePanel.tsx
git commit -m "fix: track copy timer refs to prevent memory leaks on unmount"
```

---

## Task 9: Fix Silent Failures in 7 Components

**Files:**
- Modify: `src/components/analytics/UsageStats.tsx:43`
- Modify: `src/components/skills/SkillList.tsx:30`
- Modify: `src/components/overlay/MentionHint.tsx:25`
- Modify: `src/components/overlay/ModelPicker.tsx:28`
- Modify: `src/components/overlay/CommandInput.tsx:33`
- Modify: `src/components/settings/ProvidersTab.tsx:95`
- Modify: `src/components/settings/ModelPriorityList.tsx:123`

**Problem:** All use `.catch(console.error)` — user sees nothing when errors occur.

- [ ] **Step 1: Add error state to each component**

For each component, add error state and replace `.catch(console.error)` with error handling:

```tsx
// Pattern for list-loading components:
const [error, setError] = useState<string | null>(null)

// Replace .catch(console.error) with:
.catch((e) => {
  console.error(e)
  setError('Không thể tải dữ liệu')
})

// Add error display in render:
{error && <p className="text-red-400 text-sm px-3 py-2">{error}</p>}
```

Apply this pattern to all 7 files. Each file needs:
1. Add `const [error, setError] = useState<string | null>(null)`
2. Replace `.catch(console.error)` with `.catch((e) => { console.error(e); setError('...') })`
3. Add error display in JSX (before or instead of loading state)
4. Clear error on retry: `setError(null)` at start of fetch

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 3: Commit**

```bash
git add src/components/analytics/UsageStats.tsx src/components/skills/SkillList.tsx src/components/overlay/MentionHint.tsx src/components/overlay/ModelPicker.tsx src/components/overlay/CommandInput.tsx src/components/settings/ProvidersTab.tsx src/components/settings/ModelPriorityList.tsx
git commit -m "fix: show error messages to user instead of silently catching"
```

---

## Task 10: Add ARIA Labels

**Files:**
- Modify: `src/components/overlay/CommandInput.tsx`
- Modify: `src/components/overlay/SlashMenu.tsx`
- Modify: `src/components/overlay/ModelPicker.tsx`
- Modify: `src/components/overlay/FallbackDialog.tsx`
- Modify: `src/App.tsx`

**Problem:** Interactive elements lack `aria-label`, `role`, `aria-expanded`.

- [ ] **Step 1: Add ARIA attributes to each file**

**CommandInput.tsx:**
- Clear provider button: `aria-label="Xoá provider đã chọn"`
- Model picker toggle: `aria-label="Chọn model" aria-expanded={showModelPicker}`

**SlashMenu.tsx:**
- Menu container: `role="listbox" aria-label="Danh sách lệnh"`
- Each command button: `role="option" aria-selected={index === activeIndex}`

**ModelPicker.tsx:**
- Each provider button: `aria-label={p.name}`
- Container: `role="listbox" aria-label="Chọn AI provider"`

**FallbackDialog.tsx:**
- Retry buttons: `aria-label={"Thử lại với " + name}`
- Cancel button: `aria-label="Huỷ"`

**App.tsx:**
- Settings button: add `aria-label="Cài đặt"` (already has title)

- [ ] **Step 2: Build check**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 3: Commit**

```bash
git add src/components/overlay/CommandInput.tsx src/components/overlay/SlashMenu.tsx src/components/overlay/ModelPicker.tsx src/components/overlay/FallbackDialog.tsx src/App.tsx
git commit -m "fix: add ARIA labels and roles to interactive elements"
```

---

## Task 11: Fix Responsive Design

**Files:**
- Modify: `src/components/onboarding/OnboardingWizard.tsx:31`
- Modify: `src/components/auth/LoginScreen.tsx:31`

**Problem:** Hardcoded widths (`w-[480px]`, `w-80`) don't adapt to small screens.

- [ ] **Step 1: Fix OnboardingWizard**

```tsx
// Line 31, change:
// FROM: w-[480px]
// TO:   w-full max-w-[480px] mx-auto px-4 md:px-8
```

- [ ] **Step 2: Fix LoginScreen**

```tsx
// Line 31, change:
// FROM: w-80
// TO:   w-full max-w-sm mx-auto px-4
```

- [ ] **Step 3: Build check**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 4: Commit**

```bash
git add src/components/onboarding/OnboardingWizard.tsx src/components/auth/LoginScreen.tsx
git commit -m "fix: responsive design for onboarding and login screens"
```

---

## Final Verification

- [ ] **Step 1: Full backend build + test**

Run: `cd /home/dev/open-prompt/go-engine && go build ./... && go test ./...`

- [ ] **Step 2: Full frontend build**

Run: `cd /home/dev/open-prompt && npm run build`

- [ ] **Step 3: Verify commit log**

```bash
git log --oneline -15
```
