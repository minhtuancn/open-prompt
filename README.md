# Open Prompt v1.0.0

System-wide AI assistant chạy trên Windows, Linux và macOS. Gọi bằng global hotkey, nhập query trong overlay UI, nhận response từ AI và insert vào bất kỳ ứng dụng nào.

## Download

| Platform | File |
|----------|------|
| Windows | `open-prompt_1.0.0_x64-setup.exe` |
| macOS Intel | `open-prompt_1.0.0_x64.dmg` |
| macOS Apple Silicon | `open-prompt_1.0.0_aarch64.dmg` |
| Linux (deb) | `open-prompt_1.0.0_amd64.deb` |
| Linux (AppImage) | `open-prompt_1.0.0_amd64.AppImage` |

→ [Releases](https://github.com/minhtuancn/open-prompt/releases/latest)

## Tính năng

- **Global Hotkey** — `Ctrl+Space` mở overlay tức thì
- **6 AI Providers** — Anthropic, OpenAI, Gemini, Copilot, Ollama, Custom Gateway
- **@Mention Routing** — `@claude viết email`, `@gpt4 hello`
- **Ctrl+M Model Picker** — chuyển provider nhanh
- **Slash Commands** — `/email`, `/review`, `/translate` với template
- **Smart Text Injection** — clipboard paste (non-terminal) hoặc typing (terminal)
- **Interactive Fallback** — tự động gợi ý provider thay thế khi lỗi
- **Auto-detect** — tìm API keys từ env vars, CLI tools, config files, local ports
- **Gateway Presets** — Ollama, LiteLLM, OpenRouter, vLLM
- **Multi-user** — bcrypt auth, JWT sessions
- **7 Ngôn ngữ** — vi, en, fr, zh-CN, th, lo, ru
- **Analytics** — usage stats theo provider, model, ngày
- **Conversation History** — multi-turn chat, search
- **Community Marketplace** — chia sẻ và tải prompts
- **Auto-updater** — tự cập nhật khi có phiên bản mới

## Kiến trúc

```
Tauri v2 (Rust) ←→ React 18 + Zustand  (overlay UI)
      ↓ spawns
Go Engine sidecar ←→ SQLite (~/.open-prompt/open-prompt.db)
      ↓ HTTPS
Anthropic / OpenAI / Gemini / Copilot / Ollama
```

IPC: JSON-RPC 2.0 qua Unix socket (Linux/macOS) hoặc TCP (Windows).

## Cài đặt dev

```bash
# Prerequisites: Go 1.22+, Rust stable, Node.js 18+

git clone https://github.com/minhtuancn/open-prompt.git
cd open-prompt
npm install
npm run tauri dev
```

## Build

```bash
# Linux
npm run tauri build

# Windows (cross-compile từ Linux)
cargo install cargo-xwin
rustup target add x86_64-pc-windows-msvc
npm run tauri -- build --target x86_64-pc-windows-msvc --runner "cargo-xwin"
```

## Tech Stack

| Layer | Công nghệ |
|-------|-----------|
| Desktop Shell | Tauri v2 (Rust) |
| Frontend | React 18 + TailwindCSS 3 + Zustand 5 |
| Engine | Go 1.22+ (JSON-RPC 2.0) |
| Database | SQLite (pure Go, no cgo) |
| AI Providers | Anthropic, OpenAI, Gemini, Copilot, Ollama |
| Auth | bcrypt + JWT |
| i18n | 7 ngôn ngữ |

## License

MIT — xem [LICENSE](LICENSE)
