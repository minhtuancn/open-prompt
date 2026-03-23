# Bắt đầu

## Cài đặt

### Linux (deb)
```bash
sudo dpkg -i open-prompt_1.0.0_amd64.deb
```

### Linux (AppImage)
```bash
chmod +x open-prompt_1.0.0_amd64.AppImage
./open-prompt_1.0.0_amd64.AppImage
```

### macOS
Mở file `.dmg`, kéo Open Prompt vào Applications.

### Windows
Chạy file `.msi` hoặc `.exe` installer.

## Lần đầu chạy

1. **Tạo tài khoản** — nhập username và password
2. **Chọn provider** — nhập API key cho Anthropic, OpenAI, hoặc Gemini
3. **Chọn phím tắt** — mặc định `Ctrl+Space`
4. **Bắt đầu dùng** — nhấn hotkey để mở overlay

## Phát triển

```bash
# Clone repo
git clone https://github.com/minhtuancn/open-prompt.git
cd open-prompt

# Cài dependencies
npm install

# Build Go Engine
./scripts/build-engine.sh

# Chạy dev mode
npm run tauri dev
```

## Kiến trúc

```
┌─────────────────────────┐
│     Tauri v2 (Rust)     │  ← Global hotkey, system tray, window
├─────────────────────────┤
│  React + TypeScript     │  ← Overlay UI, Settings
├─────────────────────────┤
│  Go Engine (sidecar)    │  ← JSON-RPC, providers, SQLite
└─────────────────────────┘
```

- **Tauri** quản lý cửa sổ overlay, hotkey, text injection
- **Go Engine** chạy như sidecar, giao tiếp qua Unix socket (JSON-RPC 2.0)
- **React** render overlay UI và settings
