# Providers

Open Prompt hỗ trợ 6 providers và gateway tuỳ chỉnh.

## Providers có sẵn

| Provider | Auth | Models |
|----------|------|--------|
| **Anthropic** | API Key | Claude 3.5 Sonnet, Claude 3 Opus, Haiku |
| **OpenAI** | API Key | GPT-4o, GPT-4, GPT-3.5 Turbo |
| **Gemini** | API Key / OAuth | Gemini 1.5 Pro, Gemini 1.5 Flash |
| **Copilot** | GitHub CLI Token | Copilot Chat |
| **Ollama** | Không cần | Các model local (llama3, mistral, ...) |
| **Gateway** | API Key (tuỳ) | Tuỳ gateway |

## Kết nối provider

### API Key
1. Mở **Settings → Providers**
2. Nhập API key cho provider muốn dùng
3. Nhấn **Lưu**

### Ollama (local)
Ollama được auto-detect nếu đang chạy tại `localhost:11434`.

### Gateway tuỳ chỉnh
Hỗ trợ preset:
- **Ollama** — `http://localhost:11434/v1`
- **LiteLLM** — `http://localhost:4000/v1`
- **OpenRouter** — `https://openrouter.ai/api/v1`
- **vLLM** — `http://localhost:8000/v1`

## @Mention routing

Dùng `@alias` trong prompt để chọn provider:

```
@claude Giải thích đoạn code này
@gpt4 Viết unit test cho hàm handleQuery
@gemini Tóm tắt bài viết
@ollama Dịch sang tiếng Anh
```

## Model Priority

Kéo thả trong **Settings → Providers → Model Priority** để sắp xếp thứ tự ưu tiên. Provider đầu tiên được dùng mặc định, fallback tự động khi fail.

## Auto-detect

Open Prompt tự động phát hiện:
- **Env vars**: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`
- **CLI**: `gh auth token` (GitHub Copilot)
- **Local ports**: 11434 (Ollama), 4000 (LiteLLM), 8000 (vLLM)
