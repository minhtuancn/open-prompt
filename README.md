# Open Prompt

System-wide AI assistant chạy trên Windows, Linux và macOS. Gọi bằng global hotkey, nhập query trong overlay UI, nhận response từ AI và insert vào bất kỳ ứng dụng nào.

## Tính năng

- **Global Hotkey** — `Ctrl+Space` mở overlay tức thì
- **6 AI Providers** — Anthropic, OpenAI, Gemini, Copilot, Ollama, Custom Gateway
- **@Mention Routing** — `@claude viết email`, `@gpt4 hello`
- **Ctrl+M Model Picker** — chuyển provider nhanh
- **Slash Commands** — `/email`, `/review`, `/translate` với template Go
- **Smart Text Injection** — clipboard paste (non-terminal) hoặc typing (terminal)
- **Interactive Fallback** — tự động gợi ý provider thay thế khi lỗi
- **Auto-detect** — tìm API keys từ env vars, CLI tools, config files, local ports
- **Gateway Presets** — Ollama, LiteLLM, OpenRouter, vLLM
- **Multi-user** — bcrypt auth, JWT sessions
- **7 Ngôn ngữ** — vi, en, fr, zh-CN, th, lo, ru
- **Analytics** — usage stats theo provider, model, ngày
- **Conversation History** — multi-turn chat, search
- **Health Checker** — ping providers mỗi 5 phút

## Kiến trúc

```
Tauri v2 (Rust) → React WebView → Go Engine (sidecar qua Unix socket)
                                    ↕
                              AI Providers (HTTPS)
```

## Cài đặt dev

```bash
# Prerequisites: Go 1.22+, Rust, Node.js 18+

# Clone repo
git clone https://github.com/minhtuancn/open-prompt.git
cd open-prompt

# Frontend dependencies
npm install

# Build Go Engine
cd go-engine && go build -o bin/go-engine . && cd ..

# Dev mode
./scripts/dev.sh
npm run tauri dev
```

## Tech Stack

| Layer | Công nghệ |
|-------|-----------|
| Desktop Shell | Tauri v2 (Rust) |
| Frontend | React 18 + TailwindCSS 3 + Zustand 5 |
| Engine | Go 1.22+ (JSON-RPC 2.0 qua Unix socket) |
| Database | SQLite (modernc.org/sqlite — pure Go) |
| AI Providers | Anthropic, OpenAI, Gemini, Copilot, Ollama |
| Auth | bcrypt + JWT |
| i18n | 7 ngôn ngữ |

## License

Private — minhtuancn/open-prompt
