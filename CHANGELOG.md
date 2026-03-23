# Changelog

## [1.0.0] - 2026-03-23

### Public Release
- **Release workflow** — GitHub Actions auto-build + artifact upload cho 4 platforms
- **Update manifest** — auto-generate manifest khi publish release
- **Code signing** — config cho Windows Authenticode + macOS notarization
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

### Phase 2A: Multi-Provider Engine
- **Provider Interface** — interface chuẩn cho tất cả AI providers
- **6 Providers** — Anthropic, OpenAI, Ollama, Gemini, Copilot, Gateway
- **ProviderRegistry** — Route, Default, FallbackCandidates
- **@Mention Routing** — `@claude`, `@gpt4`, `@gemini`, etc
- **ParseMention** — tách alias từ prompt
- **Auto-detect** — env vars, CLI (gh auth token), localport scanning
- **Auto-register** — providers từ DB tokens + env vars khi khởi động
- **3 API handlers** — add_gateway, validate, remove
- **Gateway Presets** — Ollama, LiteLLM, OpenRouter, vLLM

### Phase 2A3: Frontend Multi-Provider
- **ModelPicker** — Ctrl+M quick-switch với keyboard navigation
- **MentionHint** — @mention dropdown gợi ý providers
- **FallbackDialog** — interactive retry khi provider fail
- **GatewayForm** — preset templates cho gateway
- **overlayStore** — activeProvider, activeModel, fallbackProviders

### Phase 2B: OAuth + Prompt Library
- **History API** — history.list, history.search với pagination
- **HistoryPanel** — browse + search history
- **Prompts Tab** — PromptList/PromptEditor wired vào Settings
- **OAuth handlers** — PKCE, Device Flow placeholders
- **Settings 8 tabs** — Providers, Prompts, Skills, History, Hotkey, Appearance, Language, Analytics

### Phase 2C: Text Injection
- **Clipboard backup/restore** — không mất data user
- **Smart injection** — terminal → typing, non-terminal → clipboard
- **Copy button** — ngoài Insert
- **Wayland support** — swaymsg (Sway), hyprctl (Hyprland)
- **App name feedback** — hiện tên app đã inject

### Phase 3-7: Completion
- **Conversations** — multi-turn chat với messages table
- **i18n 7 ngôn ngữ** — fr, zh-CN, th, lo, ru
- **Health Checker** — ping providers mỗi 5 phút
- **Build scripts** — cross-compile, dev, release
- **Documentation** — README, CHANGELOG, ROADMAP

## [0.1.0] - 2026-03-22

### Phase 1: Foundation
- Tauri v2 shell — hotkey, window, system tray
- Go Engine sidecar — JSON-RPC 2.0 qua Unix socket
- Auth — bcrypt + JWT multi-user
- SQLite database — migrations, repos
- Overlay UI — CommandInput, ResponsePanel, SlashMenu
- Streaming — SSE from Anthropic API
- Settings — 6 tabs (Providers, Skills, Hotkey, Appearance, Language, Analytics)
- Skills/Analytics API + UI
- i18n — vi/en
- Text injection — clipboard + enigo
- CI/CD — GitHub Actions build workflow
