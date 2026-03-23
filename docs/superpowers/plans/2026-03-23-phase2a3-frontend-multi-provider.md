# Phase 2A3: Frontend Multi-Provider — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Thêm ModelPicker (Ctrl+M), FallbackDialog, GatewayForm, @mention hint dropdown vào React overlay, kết nối đầy đủ với multi-provider backend từ Phase 2A1+A2.

**Architecture:** overlayStore thêm `activeProvider`/`activeModel`/`fallbackProviders` fields. CommandInput xử lý Ctrl+M mở ModelPicker và gõ `@` hiện provider hint. ResponsePanel detect `fallback_providers` trong stream error → hiện FallbackDialog. ProvidersTab thêm section GatewayForm. streamQuery cập nhật để truyền `provider`/`model` params và nhận `fallback_providers`.

**Tech Stack:** React 18, TypeScript, Zustand 5, TailwindCSS 3, @tauri-apps/api v2

**Lưu ý:**
- OAuth (Rust `oauth.rs`) tách sang phase riêng — tất cả providers hiện hoạt động qua API key/CLIToken
- Không có unit test React (theo precedent Phase 1) — verify bằng TypeScript compile
- `callEngine<T>()` dùng `invoke('call_engine', { method, params })` qua Tauri IPC
- `streamQuery()` subscribe `stream-chunk` event, payload: `{ delta, done, error?, fallback_providers? }`
- Auth token: `useAuthStore((s) => s.token)` hoặc `localStorage.getItem('auth_token')`

---

## Spec Reference

- `docs/superpowers/specs/2026-03-23-phase2a-multi-provider-design.md` (sections 7, 8)
- `docs/superpowers/specs/2026-03-23-phase2a-implementation-approach.md`

---

## File Map

### New Files

| File | Trách nhiệm |
|------|-------------|
| `src/components/overlay/ModelPicker.tsx` | Ctrl+M quick-switch model/provider |
| `src/components/overlay/FallbackDialog.tsx` | Interactive retry khi provider fail |
| `src/components/overlay/MentionHint.tsx` | @mention provider dropdown |
| `src/components/settings/GatewayForm.tsx` | Form thêm custom gateway |

### Modified Files

| File | Thay đổi |
|------|----------|
| `src/store/overlayStore.ts` | Thêm activeProvider, activeModel, fallbackProviders, lastQuery |
| `src/hooks/useEngine.ts` | streamQuery nhận provider/model, parse fallback_providers |
| `src/components/overlay/CommandInput.tsx` | Ctrl+M, @mention hint |
| `src/components/overlay/ResponsePanel.tsx` | Detect fallback, render FallbackDialog |
| `src/components/settings/ProvidersTab.tsx` | Thêm GatewayForm section |
| `src/App.tsx` | handleQuery truyền provider/model, handleFallbackRetry |

---

## Task 1: Cập nhật overlayStore

**Files:**
- Modify: `src/store/overlayStore.ts`

- [ ] **Step 1.1: Thêm fields và actions mới**

Thay toàn bộ `src/store/overlayStore.ts`:

```tsx
import { create } from 'zustand'

interface OverlayState {
  input: string
  chunks: string[]
  isStreaming: boolean
  error: string | null
  activeProvider: string | null
  activeModel: string | null
  fallbackProviders: string[]
  lastQuery: string
  setInput: (input: string) => void
  appendChunk: (chunk: string) => void
  setStreaming: (v: boolean) => void
  setError: (e: string | null) => void
  setActiveProvider: (p: string | null) => void
  setActiveModel: (m: string | null) => void
  setFallbackProviders: (providers: string[]) => void
  setLastQuery: (q: string) => void
  reset: () => void
}

export const useOverlayStore = create<OverlayState>()((set) => ({
  input: '',
  chunks: [],
  isStreaming: false,
  error: null,
  activeProvider: null,
  activeModel: null,
  fallbackProviders: [],
  lastQuery: '',
  setInput: (input) => set({ input }),
  appendChunk: (chunk) => set((s) => ({ chunks: [...s.chunks, chunk] })),
  setStreaming: (isStreaming) => set({ isStreaming }),
  setError: (error) => set({ error }),
  setActiveProvider: (activeProvider) => set({ activeProvider }),
  setActiveModel: (activeModel) => set({ activeModel }),
  setFallbackProviders: (fallbackProviders) => set({ fallbackProviders }),
  setLastQuery: (lastQuery) => set({ lastQuery }),
  reset: () => set({ input: '', chunks: [], isStreaming: false, error: null, fallbackProviders: [], lastQuery: '' }),
}))
```

- [ ] **Step 1.2: Verify TypeScript compile**

```bash
cd /home/dev/open-prompt-code/open-prompt && npx tsc --noEmit 2>&1 | head -20
```

Expected: No errors (hoặc chỉ lỗi từ files khác — không từ overlayStore).

- [ ] **Step 1.3: Commit**

```bash
git add src/store/overlayStore.ts
git commit -m "feat: overlayStore thêm activeProvider, activeModel, fallbackProviders"
```

---

## Task 2: Cập nhật useEngine streamQuery

**Files:**
- Modify: `src/hooks/useEngine.ts`

- [ ] **Step 2.1: Cập nhật streamQuery nhận provider/model và parse fallback**

Thay toàn bộ `src/hooks/useEngine.ts`:

```tsx
import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'

/** callEngine gọi Go Engine qua Tauri IPC */
export async function callEngine<T>(method: string, params: Record<string, unknown>): Promise<T> {
  return invoke<T>('call_engine', { method, params })
}

interface StreamChunkPayload {
  delta: string
  done: boolean
  error?: string
  error_message?: string
  fallback_providers?: string[]
}

/** streamQuery gọi query.stream và subscribe notifications */
export async function streamQuery(
  params: { token: string; input: string; model?: string; provider?: string },
  onChunk: (chunk: string) => void,
  onDone: () => void,
  onError: (err: string, fallbackProviders?: string[]) => void,
): Promise<void> {
  const unlisten = await listen<StreamChunkPayload>(
    'stream-chunk',
    (event) => {
      const { delta, done, error, error_message, fallback_providers } = event.payload
      if (error) {
        onError(error_message || error, fallback_providers)
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

  callEngine('query.stream', params).catch((e) => onError(String(e)))
}
```

- [ ] **Step 2.2: Commit**

```bash
git add src/hooks/useEngine.ts
git commit -m "feat: streamQuery hỗ trợ provider/model params + fallback_providers"
```

---

## Task 3: ModelPicker component

**Files:**
- Create: `src/components/overlay/ModelPicker.tsx`

- [ ] **Step 3.1: Tạo ModelPicker**

`src/components/overlay/ModelPicker.tsx`:

```tsx
import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface ProviderInfo {
  id: string
  name: string
  connected: boolean
}

interface Props {
  onSelect: (providerName: string) => void
  onClose: () => void
}

export function ModelPicker({ onSelect, onClose }: Props) {
  const token = useAuthStore((s) => s.token)
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [selectedIdx, setSelectedIdx] = useState(0)

  useEffect(() => {
    if (!token) return
    callEngine<ProviderInfo[]>('providers.list', { token })
      .then((list) => {
        const connected = (list ?? []).filter((p) => p.connected)
        setProviders(connected.length > 0 ? connected : list ?? [])
      })
      .catch(console.error)
  }, [token])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { onClose(); return }
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIdx((i) => Math.min(i + 1, providers.length - 1))
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIdx((i) => Math.max(i - 1, 0))
      }
      if (e.key === 'Enter' && providers.length > 0) {
        e.preventDefault()
        onSelect(providers[selectedIdx].id)
        onClose()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [providers, selectedIdx, onSelect, onClose])

  if (providers.length === 0) return null

  return (
    <div className="absolute top-0 left-0 right-0 z-50 bg-surface/98 backdrop-blur-xl border border-white/10 rounded-xl shadow-2xl p-2">
      <div className="flex items-center justify-between px-3 py-1.5 mb-1">
        <span className="text-xs text-white/50 font-medium">Chọn provider</span>
        <span className="text-xs text-white/30">ESC để đóng</span>
      </div>
      {providers.map((p, i) => (
        <button
          key={p.id}
          onClick={() => { onSelect(p.id); onClose() }}
          className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
            i === selectedIdx
              ? 'bg-indigo-500/20 text-white'
              : 'text-white/70 hover:bg-white/5'
          }`}
        >
          <span className="font-medium">{p.name}</span>
          {p.connected && <span className="ml-2 text-xs text-green-400/70">●</span>}
        </button>
      ))}
    </div>
  )
}
```

- [ ] **Step 3.2: Commit**

```bash
git add src/components/overlay/ModelPicker.tsx
git commit -m "feat: thêm ModelPicker component (Ctrl+M quick-switch)"
```

---

## Task 4: MentionHint component

**Files:**
- Create: `src/components/overlay/MentionHint.tsx`

- [ ] **Step 4.1: Tạo MentionHint**

`src/components/overlay/MentionHint.tsx`:

```tsx
import { useEffect, useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

interface ProviderInfo {
  id: string
  name: string
  connected: boolean
}

interface Props {
  query: string
  onSelect: (alias: string) => void
  visible: boolean
}

export function MentionHint({ query, onSelect, visible }: Props) {
  const token = useAuthStore((s) => s.token)
  const [providers, setProviders] = useState<ProviderInfo[]>([])

  useEffect(() => {
    if (!token) return
    callEngine<ProviderInfo[]>('providers.list', { token })
      .then((list) => setProviders(list ?? []))
      .catch(console.error)
  }, [token])

  if (!visible || !query) return null

  const filtered = providers.filter(
    (p) => p.id.includes(query.toLowerCase()) || p.name.toLowerCase().includes(query.toLowerCase())
  )
  if (filtered.length === 0) return null

  return (
    <div className="absolute bottom-full left-5 mb-1 bg-surface border border-white/10 rounded-lg shadow-xl p-1 min-w-48 z-50">
      {filtered.map((p) => (
        <button
          key={p.id}
          onClick={() => onSelect(p.id)}
          className="w-full text-left px-3 py-1.5 rounded-md text-sm text-white/70 hover:bg-white/10 hover:text-white transition-colors"
        >
          <span className="text-indigo-400">@</span>{p.id}
          <span className="ml-2 text-xs text-white/30">{p.name}</span>
        </button>
      ))}
    </div>
  )
}
```

- [ ] **Step 4.2: Commit**

```bash
git add src/components/overlay/MentionHint.tsx
git commit -m "feat: thêm MentionHint dropdown cho @mention"
```

---

## Task 5: FallbackDialog component

**Files:**
- Create: `src/components/overlay/FallbackDialog.tsx`

- [ ] **Step 5.1: Tạo FallbackDialog**

`src/components/overlay/FallbackDialog.tsx`:

```tsx
interface Props {
  errorMessage: string
  providers: string[]
  onRetry: (provider: string) => void
  onCancel: () => void
}

export function FallbackDialog({ errorMessage, providers, onRetry, onCancel }: Props) {
  return (
    <div className="mt-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-3">
      <p className="text-yellow-400 text-xs mb-2">⚠ {errorMessage}</p>
      <p className="text-white/50 text-xs mb-2">Thử lại với:</p>
      <div className="flex flex-wrap gap-2">
        {providers.map((name) => (
          <button
            key={name}
            onClick={() => onRetry(name)}
            className="px-3 py-1 bg-white/10 hover:bg-indigo-500/30 text-white/80 hover:text-white text-xs rounded-md border border-white/10 hover:border-indigo-500/30 transition-colors"
          >
            {name}
          </button>
        ))}
        <button
          onClick={onCancel}
          className="px-3 py-1 text-white/30 hover:text-white/60 text-xs transition-colors"
        >
          Hủy
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 5.2: Commit**

```bash
git add src/components/overlay/FallbackDialog.tsx
git commit -m "feat: thêm FallbackDialog — interactive retry khi provider fail"
```

---

## Task 6: GatewayForm component

**Files:**
- Create: `src/components/settings/GatewayForm.tsx`

- [ ] **Step 6.1: Tạo GatewayForm**

`src/components/settings/GatewayForm.tsx`:

```tsx
import { useState } from 'react'
import { callEngine } from '../../hooks/useEngine'
import { useAuthStore } from '../../store/authStore'

const PRESETS = [
  { name: 'ollama', displayName: 'Ollama (Local)', baseURL: 'http://localhost:11434/v1', defaultModel: 'llama3.2' },
  { name: 'litellm', displayName: 'LiteLLM', baseURL: 'http://localhost:4000/v1', defaultModel: 'gpt-4o' },
  { name: 'openrouter', displayName: 'OpenRouter', baseURL: 'https://openrouter.ai/api/v1', defaultModel: 'openai/gpt-4o' },
  { name: 'vllm', displayName: 'vLLM', baseURL: 'http://localhost:8000/v1', defaultModel: '' },
]

interface Props {
  onAdded?: () => void
}

export function GatewayForm({ onAdded }: Props) {
  const token = useAuthStore((s) => s.token)
  const [name, setName] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [defaultModel, setDefaultModel] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  const applyPreset = (preset: typeof PRESETS[0]) => {
    setName(preset.name)
    setDisplayName(preset.displayName)
    setBaseURL(preset.baseURL)
    setDefaultModel(preset.defaultModel)
  }

  const handleSubmit = async () => {
    if (!token || !name || !baseURL) return
    setSaving(true)
    setError('')
    try {
      await callEngine('providers.add_gateway', {
        token, name, display_name: displayName || name, base_url: baseURL,
        api_key: apiKey, default_model: defaultModel,
      })
      setSuccess(true)
      setTimeout(() => setSuccess(false), 2000)
      setName(''); setDisplayName(''); setBaseURL(''); setApiKey(''); setDefaultModel('')
      onAdded?.()
    } catch (e) {
      setError(String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-4">
      <div className="text-sm font-medium text-white mb-3">Thêm Gateway</div>

      <div className="flex flex-wrap gap-1.5 mb-3">
        {PRESETS.map((p) => (
          <button key={p.name} onClick={() => applyPreset(p)}
            className="text-xs px-2 py-1 bg-white/5 hover:bg-indigo-500/20 text-white/50 hover:text-white rounded-md border border-white/10 transition-colors">
            {p.displayName}
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-2">
        <input placeholder="Tên (vd: my-ollama)" value={name} onChange={(e) => setName(e.target.value)}
          className="bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
        <input placeholder="Base URL (vd: http://localhost:11434/v1)" value={baseURL} onChange={(e) => setBaseURL(e.target.value)}
          className="bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 font-mono" />
        <div className="flex gap-2">
          <input placeholder="API Key (tùy chọn)" type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)}
            className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50 font-mono" />
          <input placeholder="Model mặc định" value={defaultModel} onChange={(e) => setDefaultModel(e.target.value)}
            className="flex-1 bg-black/20 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-white/20 outline-none focus:border-indigo-500/50" />
        </div>
        <div className="flex items-center gap-2">
          <button onClick={handleSubmit} disabled={!name || !baseURL || saving}
            className="text-xs px-4 py-2 bg-indigo-500/80 hover:bg-indigo-500 text-white rounded-lg transition-colors disabled:opacity-40">
            {success ? '✓ Đã thêm' : saving ? '...' : 'Thêm Gateway'}
          </button>
          {error && <span className="text-xs text-red-400">{error}</span>}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 6.2: Commit**

```bash
git add src/components/settings/GatewayForm.tsx
git commit -m "feat: thêm GatewayForm với preset templates"
```

---

## Task 7: Tích hợp Ctrl+M + @mention vào CommandInput

**Files:**
- Modify: `src/components/overlay/CommandInput.tsx`

- [ ] **Step 7.1: Thêm import và state mới**

Ở đầu file, thêm imports:
```tsx
import { ModelPicker } from './ModelPicker'
import { MentionHint } from './MentionHint'
import { useOverlayStore } from '../../store/overlayStore'
```

Thêm state vào component (sau `textareaRef`):
```tsx
const [showModelPicker, setShowModelPicker] = useState(false)
const [mentionQuery, setMentionQuery] = useState('')
const [showMentionHint, setShowMentionHint] = useState(false)
const { activeProvider, setActiveProvider } = useOverlayStore()
```

- [ ] **Step 7.2: Xử lý Ctrl+M trong handleKeyDown**

Thêm vào `handleKeyDown`, trước `if (e.key === 'Enter')`:
```tsx
if (e.key === 'm' && (e.ctrlKey || e.metaKey)) {
  e.preventDefault()
  setShowModelPicker((v) => !v)
  return
}
```

- [ ] **Step 7.3: Xử lý @mention trong handleChange**

Cập nhật `handleChange` để detect `@`:
```tsx
// Detect @mention
const atMatch = value.match(/@(\w*)$/)
if (atMatch) {
  setMentionQuery(atMatch[1])
  setShowMentionHint(true)
} else {
  setShowMentionHint(false)
  setMentionQuery('')
}
```

- [ ] **Step 7.4: Thêm ModelPicker và MentionHint vào JSX**

Trong return JSX, trước `<SlashMenu>`:
```tsx
{showModelPicker && (
  <ModelPicker
    onSelect={(name) => setActiveProvider(name)}
    onClose={() => setShowModelPicker(false)}
  />
)}
```

Trong return JSX, trước `<textarea>`:
```tsx
<MentionHint
  query={mentionQuery}
  onSelect={(alias) => {
    const newInput = input.replace(/@\w*$/, '')
    setInput(newInput)
    setActiveProvider(alias)
    setShowMentionHint(false)
  }}
  visible={showMentionHint}
/>
```

- [ ] **Step 7.5: Hiện active provider badge**

Trong status bar (dòng `isStreaming ? 'Đang xử lý...'`), thêm badge nếu có activeProvider:
```tsx
<span className="text-xs text-white/20">
  {activeProvider && (
    <span className="text-indigo-400 mr-2">
      @{activeProvider}
      <button onClick={() => setActiveProvider(null)} className="ml-1 text-white/30 hover:text-white/60">✕</button>
    </span>
  )}
  {isStreaming ? 'Đang xử lý...' : 'Enter gửi • Ctrl+M chọn model • @ mention provider'}
</span>
```

- [ ] **Step 7.6: Verify compile**

```bash
npx tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 7.7: Commit**

```bash
git add src/components/overlay/CommandInput.tsx
git commit -m "feat: CommandInput thêm Ctrl+M model picker + @mention hint"
```

---

## Task 8: Tích hợp FallbackDialog vào ResponsePanel

**Files:**
- Modify: `src/components/overlay/ResponsePanel.tsx`

- [ ] **Step 8.1: Thêm FallbackDialog import và logic**

Thêm import:
```tsx
import { FallbackDialog } from './FallbackDialog'
import { useOverlayStore } from '../../store/overlayStore'
```

Trong component, lấy fallback state:
```tsx
const { chunks, isStreaming, error, fallbackProviders } = useOverlayStore()
```

- [ ] **Step 8.2: Render FallbackDialog khi có fallback providers**

Sau error display block, thêm:
```tsx
{error && fallbackProviders.length > 0 && (
  <FallbackDialog
    errorMessage={error}
    providers={fallbackProviders}
    onRetry={(provider) => {
      // Trigger retry — App.tsx sẽ xử lý qua callback
      window.dispatchEvent(new CustomEvent('fallback-retry', { detail: { provider } }))
    }}
    onCancel={() => useOverlayStore.getState().setFallbackProviders([])}
  />
)}
```

- [ ] **Step 8.3: Commit**

```bash
git add src/components/overlay/ResponsePanel.tsx
git commit -m "feat: ResponsePanel detect fallback_providers, hiện FallbackDialog"
```

---

## Task 9: Tích hợp trong App.tsx

**Files:**
- Modify: `src/App.tsx`

- [ ] **Step 9.1: Cập nhật handleQuery truyền provider/model**

Thêm import `useOverlayStore` nếu chưa có, lấy thêm fields:
```tsx
const { reset, appendChunk, setStreaming, setError, setFallbackProviders, setLastQuery } = useOverlayStore()
const activeProvider = useOverlayStore((s) => s.activeProvider)
```

Cập nhật `handleQuery`:
```tsx
const handleQuery = async (input: string, slashName?: string, extraVars?: Record<string, string>) => {
  if (!token) return
  reset()
  setStreaming(true)
  setLastQuery(input)
  await streamQuery(
    { token, input, provider: activeProvider || undefined },
    (chunk) => appendChunk(chunk),
    () => setStreaming(false),
    (err, fallback) => {
      setError(err)
      setStreaming(false)
      if (fallback && fallback.length > 0) setFallbackProviders(fallback)
    }
  )
}
```

- [ ] **Step 9.2: Thêm fallback retry listener**

Trong App component, thêm useEffect cho fallback-retry event:
```tsx
useEffect(() => {
  const handler = (e: Event) => {
    const { provider } = (e as CustomEvent).detail
    const lastQuery = useOverlayStore.getState().lastQuery
    if (!token || !lastQuery) return
    // Retry với provider mới
    const { reset: resetStore, setStreaming: setStr, appendChunk: addChunk, setError: setErr, setFallbackProviders: setFb } = useOverlayStore.getState()
    resetStore()
    setStr(true)
    streamQuery(
      { token, input: lastQuery, provider },
      (chunk) => addChunk(chunk),
      () => useOverlayStore.getState().setStreaming(false),
      (err, fallback) => {
        setErr(err)
        useOverlayStore.getState().setStreaming(false)
        if (fallback && fallback.length > 0) setFb(fallback)
      }
    )
  }
  window.addEventListener('fallback-retry', handler)
  return () => window.removeEventListener('fallback-retry', handler)
}, [token])
```

- [ ] **Step 9.3: Verify compile**

```bash
npx tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 9.4: Commit**

```bash
git add src/App.tsx
git commit -m "feat: App.tsx tích hợp provider/model routing + fallback retry"
```

---

## Task 10: Thêm GatewayForm vào ProvidersTab

**Files:**
- Modify: `src/components/settings/ProvidersTab.tsx`

- [ ] **Step 10.1: Import và render GatewayForm**

Thêm import:
```tsx
import { GatewayForm } from './GatewayForm'
```

Trong JSX, sau danh sách providers (sau `{providers.map(...)}`), thêm:
```tsx
<div className="mt-4 pt-4 border-t border-white/10">
  <GatewayForm onAdded={() => {
    if (!token) return
    callEngine<Provider[]>('providers.list', { token })
      .then((list) => setProviders(list ?? []))
      .catch(console.error)
  }} />
</div>
```

- [ ] **Step 10.2: Verify compile**

```bash
npx tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 10.3: Commit**

```bash
git add src/components/settings/ProvidersTab.tsx
git commit -m "feat: ProvidersTab tích hợp GatewayForm"
```

---

## Task 11: Build verification + Merge

- [ ] **Step 11.1: Full TypeScript + Vite build**

```bash
cd /home/dev/open-prompt-code/open-prompt && npx tsc --noEmit && npm run build
```

Expected: Build success.

- [ ] **Step 11.2: Go tests vẫn pass**

```bash
cd go-engine && go test ./... -count=1 2>&1 | tail -12
```

- [ ] **Step 11.3: Merge vào main và push**

```bash
git checkout main && git merge --no-edit && git push origin main
```
