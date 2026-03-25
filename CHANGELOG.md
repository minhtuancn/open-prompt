# Changelog

## [1.0.0] - 2026-03-25

### Public Release
- **Release workflow** — GitHub Actions auto-build cho 4 platforms: Windows (NSIS), macOS Intel/ARM (DMG), Linux (deb + AppImage)
- **Tauri updater** — pubkey signing, auto-update manifest từ release repo
- **CI/CD fixes** — fix rust-toolchain action, permissions, NSIS icon, macOS signing
- **Security hardening** — fix IDOR, SQL injection, XSS, rate limiting
- **Code quality** — deduplicate queries, fix memory leaks, giảm DB round-trips
- **Platform installers** — NSIS (Windows), DMG (macOS), deb + AppImage (Linux)
- **Documentation site** — VitePress với guide + API reference
- **Onboarding wizard** — 5 bước: welcome → account → provider → hotkey → done
- **Community marketplace** — browse, search, publish, install shared prompts
- **Telemetry opt-in** — track events cơ bản khi user đồng ý
- **Drag-drop model priority** — @dnd-kit sortable list trong ProvidersTab
- **Rich text injection** — strip markdown trước khi inject vào app
- **Settings 10 tabs** — thêm Marketplace tab

## [0.4.0] - 2026-03-23

### Advanced Features
- **Plugin system** — install, list, toggle, uninstall (provider/skill/formatter types)
- **Prompt export/import** — JSON format, bulk import
- **Markdown renderer** — code blocks, bold, italic, headings, lists
- **Hotkey customization** — 6 preset options, lưu vào settings
- **Analytics daily aggregation** — rollup history → usage_daily

## [0.3.0] - 2026-03-23

### Production Polish
- **OAuth WebView** — start_oauth (WebView/DeviceFlow/browser), poll_oauth
- **DeviceFlowDialog** — UI cho GitHub Device Flow polling
- **Tauri auto-updater** — check/download/install updates, UpdateTab
- **Multi-turn streaming** — conversation_id trong query.stream
- **Token Expiry Watcher** — kiểm tra tokens mỗi 2 phút

## [0.2.0] - 2026-03-23

### Multi-Provider Engine
- **Provider Interface** — interface chuẩn cho tất cả AI providers
- **6 Providers** — Anthropic, OpenAI, Ollama, Gemini, Copilot, Gateway
- **ProviderRegistry** — Route, Default, FallbackCandidates
- **@Mention Routing** — `@claude`, `@gpt4`, `@gemini`, etc
- **Auto-detect** — env vars, CLI (gh auth token), localport scanning
- **Gateway Presets** — Ollama, LiteLLM, OpenRouter, vLLM
- **ModelPicker** — Ctrl+M quick-switch với keyboard navigation
- **History API** — history.list, history.search với pagination
- **Smart text injection** — clipboard backup, Wayland support
- **i18n 7 ngôn ngữ** — vi, en, fr, zh-CN, th, lo, ru
- **Conversations** — multi-turn chat với messages table
- **Health Checker** — ping providers mỗi 5 phút

## [0.1.0] - 2026-03-22

### Foundation
- Tauri v2 shell — global hotkey, overlay window, system tray
- Go Engine sidecar — JSON-RPC 2.0 qua Unix socket
- Auth — bcrypt + JWT multi-user
- SQLite database — migrations, repos pattern
- Overlay UI — CommandInput, ResponsePanel, SlashMenu
- Streaming — SSE từ Anthropic API
- Settings — 6 tabs
- Skills/Analytics API + UI
- i18n — vi/en
- Text injection — clipboard + enigo
- CI/CD — GitHub Actions build workflow
