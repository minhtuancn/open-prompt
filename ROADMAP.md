# Roadmap

## Đã hoàn thành

### v0.1.0 — Foundation
- [x] Tauri v2 + Go Engine sidecar
- [x] Auth, SQLite, JSON-RPC
- [x] Overlay UI + Streaming
- [x] Slash commands + Prompts
- [x] Text injection

### v0.2.0 — Multi-Provider + UX
- [x] Provider interface + 6 providers
- [x] @mention routing + Ctrl+M picker
- [x] Auto-detect (env, CLI, localport)
- [x] Gateway presets (Ollama, LiteLLM, OpenRouter, vLLM)
- [x] Fallback dialog
- [x] History browsing + search
- [x] i18n 7 ngôn ngữ
- [x] Smart text injection (clipboard backup, Wayland)
- [x] Health checker
- [x] Conversations (multi-turn)

### v0.3.0 — Production Polish
- [x] OAuth WebView hoàn chỉnh (Google, GitHub Device Flow)
- [x] Tauri auto-updater integration
- [x] Conversation context trong query (multi-turn streaming)
- [x] Drag-drop model priority trong ProvidersTab
- [x] Token expiry watcher + callback

### v0.4.0 — Advanced Features
- [x] Plugin system (custom providers, custom skills)
- [x] Prompt sharing + import/export
- [x] Rich text injection (strip Markdown trước inject)
- [x] Keyboard shortcut customization
- [x] Usage analytics daily aggregation (usage_daily table)

### v1.0.0 — Public Release ✅
- [x] Release workflow (GitHub Actions, 4 platforms)
- [x] Tauri update signing + pubkey
- [x] Security hardening (IDOR, SQL injection, XSS, rate limiting)
- [x] Platform installers (NSIS, DMG, deb, AppImage)
- [x] Documentation site (VitePress)
- [x] Onboarding wizard (5 bước)
- [x] Community prompts marketplace
- [x] Telemetry opt-in
- [x] MIT License

## Kế hoạch

### v1.1.0 — Community & Polish
- [ ] Marketplace server (remote API thay vì local-only)
- [ ] Prompt rating + reviews
- [ ] Plugin marketplace
- [ ] Auto-update Go Engine sidecar
- [ ] Crash reporting (Sentry)
- [ ] Keyboard shortcut recorder (tuỳ chỉnh tự do)
- [ ] Apple Developer signing + notarization
