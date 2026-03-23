# Phase 2A Multi-Provider — Implementation Approach

**Ngày:** 2026-03-23
**Trạng thái:** Approved
**Spec gốc:** `docs/superpowers/specs/2026-03-23-phase2a-multi-provider-design.md`

---

## 1. Gap Analysis Summary

Code hiện có đã triển khai ~35% Phase 2A. Phần còn lại ~65% cần implement.

### Đã có
- `anthropic.go`, `openai.go`, `ollama.go` — StreamComplete nhưng chưa implement Provider interface
- `provider/detector.go` — env + config file scan (monolithic)
- `provider/registry.go` — metadata registry (không có Route/Default/Fallback)
- `handlers_providers.go` — 4 handlers: list, detect, connect, set_priority
- `ProvidersTab.tsx` — hiển thị providers + API key input
- `model_priority` table — đã có trong migration 001

### Chưa có
- Provider interface (`interface.go`)
- 3 providers mới: `gemini.go`, `copilot.go`, `gateway.go`
- `mention.go` — ParseMention() routing
- 7 API handlers mới
- `query.stream` refactor (Registry, @mention, fallback)
- Detector: CLI scanner, localport scanner
- Migration: `model_aliases`, `custom_gateways` tables
- OAuth flow: `oauth.rs`, custom URL scheme
- 3 React components: `ModelPicker.tsx`, `FallbackDialog.tsx`, `GatewayForm.tsx`
- `overlayStore.ts`: activeProvider/activeModel fields

---

## 2. Sub-phase Split

### Sub-phase A1: Provider Interface + Refactor
**Mục tiêu:** Thiết lập nền tảng interface pattern, refactor code hiện có.

**Scope:**
1. `model/providers/interface.go` — Provider interface, AuthType, CompletionRequest
2. Refactor `anthropic.go` → implement Provider interface
3. Refactor `openai.go` → implement Provider interface
4. Refactor `ollama.go` → implement Provider interface
5. `model/providers/registry.go` — ProviderRegistry mới (Register, Route, Default, All, FallbackCandidates)
6. `api/mention.go` — ParseMention()
7. Refactor `handlers_query.go` — dùng Registry + ParseMention + fallback metadata
8. Migration 002: `model_aliases`, `custom_gateways` tables, aliases column

**Kết quả:** @mention routing hoạt động với 3 providers hiện có, fallback chain ready.

### Sub-phase A2: New Providers + Detector
**Mục tiêu:** Thêm providers mới và mở rộng auto-detection.

**Scope:**
1. `gemini.go` — Google Gemini (API key trước, OAuth sau)
2. `copilot.go` — GitHub Copilot (CLIToken trước, Device Flow sau)
3. `gateway.go` — Generic OpenAI-compat (preset templates)
4. Detector mở rộng: CLI scanner, localport scanner, parallel execution
5. 7 API handlers mới: add_gateway, remove, set_default, validate, oauth_start/finish/poll
6. Router cập nhật routes mới

**Kết quả:** 6+ providers sử dụng được, auto-detect tìm providers trên máy.

### Sub-phase A3: OAuth + Frontend
**Mục tiêu:** OAuth flow và frontend UI hoàn chỉnh.

**Scope:**
1. `src-tauri/src/oauth.rs` — WebView OAuth command
2. `tauri.conf.json` — custom URL scheme "open-prompt://"
3. `ModelPicker.tsx` — Ctrl+M quick-switch
4. `FallbackDialog.tsx` — interactive fallback khi provider fail
5. `GatewayForm.tsx` — thêm custom gateway
6. `CommandInput.tsx` — Ctrl+M + @mention dropdown
7. `ResponsePanel.tsx` — detect fallback_providers
8. `overlayStore.ts` — activeProvider, activeModel fields
9. `ProvidersTab.tsx` — drag-drop priority + gateway section

**Kết quả:** Phase 2A hoàn thành theo định nghĩa trong spec gốc.

---

## 3. Thứ tự triển khai

```
A1 (interface + refactor) → A2 (new providers) → A3 (OAuth + frontend)
```

Mỗi sub-phase là một PR riêng, có thể merge độc lập. A1 là blocking — A2 và A3 phụ thuộc vào A1.

---

## 4. Quyết định kỹ thuật

### Provider Interface location
- Giữ ở `go-engine/model/providers/` (cùng package với implementations)
- Registry mới tạo tại `go-engine/model/providers/registry.go` — khác với `provider/registry.go` hiện có (metadata)

### Detector organization
- Giữ detector ở `go-engine/provider/detector.go` (vị trí hiện tại)
- Mở rộng thêm methods thay vì tách file (tránh refactor không cần thiết)

### Migration strategy
- Migration 002 cho schema changes (model_aliases, custom_gateways, aliases column)
- Chạy khi Go engine khởi động, check version trước khi apply

### Naming convention
- JSON-RPC methods: `providers.*` (giữ plural như code hiện có, không đổi sang `provider.*` như spec)
- Lý do: code hiện có đã dùng `providers.*`, đổi tên gây breaking change không cần thiết
