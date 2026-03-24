# Phím tắt

## Hotkey mở overlay

Mặc định: `Ctrl+Space`. Thay đổi trong **Settings → Phím tắt**.

Các lựa chọn:
- `Ctrl+Space`
- `Ctrl+Shift+Space`
- `Alt+Space`
- `Ctrl+/`
- `Ctrl+J`
- `Super+Space`

## Phím tắt trong overlay

| Phím | Chức năng |
|------|-----------|
| `Ctrl+M` | Mở Model Picker — chọn provider/model |
| `@` | Mention provider (gõ `@claude`, `@gpt4`, ...) |
| `/` | Slash commands |
| `Enter` | Gửi query |
| `Shift+Enter` | Xuống dòng |
| `Escape` | Đóng overlay |

## Text injection

Sau khi nhận response, nhấn **Insert ↵** để inject text vào app đang focus:
- **Terminal** → gõ từng ký tự (tránh paste issues)
- **App khác** → clipboard paste (tự động backup/restore clipboard)

Markdown được tự động strip trước khi inject (bỏ `##`, `**`, `` ` ``).
