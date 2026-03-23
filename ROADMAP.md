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
- [x] Build scripts

## Kế hoạch

### v0.3.0 — Production Polish
- [ ] OAuth WebView hoàn chỉnh (Google, GitHub Device Flow)
- [ ] Tauri auto-updater integration
- [ ] Code signing (Windows Authenticode, macOS notarization)
- [ ] Conversation context trong query (multi-turn streaming)
- [ ] Drag-drop model priority trong ProvidersTab
- [ ] Token expiry watcher + auto-refresh

### v0.4.0 — Advanced Features
- [ ] Plugin system (custom providers, custom skills)
- [ ] Prompt sharing + import/export
- [ ] Rich text injection (Markdown → HTML)
- [ ] Keyboard shortcut customization
- [ ] Usage analytics daily aggregation (usage_daily table)

### v1.0.0 — Public Release
- [ ] Public repo release workflow
- [ ] Documentation site
- [ ] Installer cho Windows/macOS/Linux
- [ ] Community prompts marketplace
