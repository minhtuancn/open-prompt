# Open Prompt — Phase 1 Completion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hoàn thành Phase 1 — verify Tauri build trên Linux, chạy integration test end-to-end, đảm bảo toàn bộ Phase 1 checklist pass.

**Architecture:** Tauri v2 (Rust) + Go Engine (sidecar qua Unix socket) + React WebView. Code đã hoàn chỉnh; task còn lại là cài system dependencies, verify build, và integration test.

**Tech Stack:** Go 1.22+, Tauri v2, Rust stable, React 18, Vite, TailwindCSS v3, SQLite (modernc.org/sqlite), bcrypt, JWT, zustand

---

## Spec Reference

`docs/superpowers/specs/2026-03-22-open-prompt-design.md`

## Plan Gốc

`docs/superpowers/plans/2026-03-22-phase1-foundation.md`

---

## Trạng thái hiện tại (2026-03-23)

| Thành phần | Trạng thái |
|------------|------------|
| Go engine code | ✅ Hoàn chỉnh |
| Go engine binary | ✅ `go-engine/bin/go-engine-linux-amd64` |
| Tauri binary sidecar | ✅ `src-tauri/binaries/go-engine-x86_64-unknown-linux-gnu` |
| Go tests (db, auth, api) | ✅ PASS |
| Go test (anthropic) | ⚠️ FAIL vì không có API credit (expected) |
| React TypeScript | ✅ Build sạch |
| Tauri cargo build | ❌ Thiếu Linux system dependencies |
| Integration test | ❌ Chưa chạy |

---

## File Map

Không có file mới cần tạo. Tất cả code đã có. Plan này chỉ verify và fix.

```
go-engine/
├── bin/go-engine-linux-amd64         ← compiled ✅
├── api/, auth/, db/, model/, config/ ← tất cả code ✅
src-tauri/
├── binaries/go-engine-x86_64-unknown-linux-gnu ← binary copied ✅
└── src/ (main.rs, lib.rs, hotkey.rs, tray.rs, window.rs, sidecar.rs, ipc.rs) ✅
src/ (React components, stores, hooks) ✅
```

---

## Task 1: Cài Linux System Dependencies cho Tauri

**Files:** Không có — chỉ cài apt packages

- [ ] **Step 1.1: Cài Tauri dependencies cho Ubuntu/Debian**

```bash
sudo apt-get update
sudo apt-get install -y \
  libwebkit2gtk-4.1-dev \
  libappindicator3-dev \
  librsvg2-dev \
  patchelf \
  libssl-dev \
  pkg-config
```

Expected: tất cả packages cài thành công, không có lỗi.

- [ ] **Step 1.2: Verify pkg-config tìm thấy webkit2gtk**

```bash
pkg-config --modversion webkit2gtk-4.1
```

Expected: in ra version string, ví dụ `2.44.0`.

---

## Task 2: Verify Tauri Cargo Build

**Files:** `src-tauri/` — không thay đổi, chỉ verify

- [ ] **Step 2.1: Chạy cargo check**

```bash
cd /home/dev/open-prompt-code/open-prompt/src-tauri
source ~/.cargo/env
cargo check 2>&1 | tail -5
```

Expected: `Finished dev [unoptimized + debuginfo] target(s) in ...`

Nếu có lỗi khác (không phải missing libs), đọc thông báo lỗi và fix trước khi tiếp tục.

- [ ] **Step 2.2: Kiểm tra warnings nghiêm trọng**

```bash
source ~/.cargo/env
cargo check 2>&1 | grep "^error" | head -20
```

Expected: không có dòng `error` nào (chỉ warnings là OK).

- [ ] **Step 2.3: Commit nếu có fix**

Nếu không có thay đổi gì, skip. Nếu có fix:

```bash
cd /home/dev/open-prompt-code/open-prompt
git add src-tauri/
git commit -m "fix: sửa lỗi Rust compilation"
```

---

## Task 3: Smoke Test Go Engine

**Files:** Không có — chỉ test binary hiện tại

- [ ] **Step 3.1: Test Go engine binary khởi động**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine

# Chạy engine, bắt stdout và kiểm tra có "ready"
OP_SOCKET_SECRET=test-secret-32bytes-000000000000 ./bin/go-engine-linux-amd64 &
ENGINE_PID=$!
sleep 2

# Đọc socket signal — engine viết "ready" ra stdout khi sẵn sàng
# Kiểm tra process còn sống
if kill -0 $ENGINE_PID 2>/dev/null; then
  echo "PASS: Engine đang chạy (PID=$ENGINE_PID)"
else
  echo "FAIL: Engine đã crash"
fi
kill $ENGINE_PID 2>/dev/null
```

Expected: in ra `PASS: Engine đang chạy (PID=...)` và không thấy lỗi fatal.

- [ ] **Step 3.2: Cài netcat và test socket connection**

```bash
# Cài netcat nếu chưa có
which nc || sudo apt-get install -y netcat-openbsd
```

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine

# Cleanup socket cũ
rm -f /tmp/open-prompt.sock

# Start engine
OP_SOCKET_SECRET=test-secret-32bytes-000000000000 ./bin/go-engine-linux-amd64 &
ENGINE_PID=$!
sleep 1

# Gửi JSON-RPC request (format envelope: secret + request, đúng theo ipc.rs)
echo '{"secret":"test-secret-32bytes-000000000000","request":{"jsonrpc":"2.0","method":"auth.is_first_run","params":{},"id":1}}' \
  | nc -U /tmp/open-prompt.sock

kill $ENGINE_PID 2>/dev/null
```

Expected: response JSON với `{"jsonrpc":"2.0","result":{"is_first_run":true},"id":1}` (hoặc false nếu DB đã có data).

- [ ] **Step 3.3: Go tests đầy đủ (skip anthropic)**

```bash
cd /home/dev/open-prompt-code/open-prompt/go-engine
go test ./... -v -run 'Test' -skip 'TestAnthropicProvider' 2>&1 | tail -20
```

Expected: tất cả tests PASS, không có FAIL nào.

---

## Task 4: Build Tauri App (Dev Mode)

**Files:** Không có — chỉ chạy build

- [ ] **Step 4.1: Verify Go binary đúng vị trí**

```bash
ls -la /home/dev/open-prompt-code/open-prompt/src-tauri/binaries/
```

Expected: file `go-engine-x86_64-unknown-linux-gnu` tồn tại và có thể execute.

```bash
file /home/dev/open-prompt-code/open-prompt/src-tauri/binaries/go-engine-x86_64-unknown-linux-gnu
```

Expected: `ELF 64-bit LSB executable, x86-64`

- [ ] **Step 4.2: Build Tauri (release)**

```bash
cd /home/dev/open-prompt-code/open-prompt
source ~/.cargo/env
npm run tauri build -- --no-bundle 2>&1 | tail -30
```

> Dùng `--no-bundle` để skip packaging (deb/AppImage), chỉ build binary. Nhanh hơn cho verification.

Expected: `Finished release [optimized] target(s)` và binary tại `src-tauri/target/release/open-prompt`.

Nếu lỗi liên quan đến icon (32x32 PNG), check `src-tauri/icons/icon.png` tồn tại:

```bash
python3 -c "
import struct
with open('src-tauri/icons/icon.png', 'rb') as f:
    sig = f.read(8)
assert sig == b'\x89PNG\r\n\x1a\n', 'Invalid PNG'
print('PNG valid:', sig.hex())
"
```

Nếu icon không tồn tại hoặc invalid, tạo lại:

```bash
mkdir -p src-tauri/icons
python3 - <<'EOF'
import struct, zlib

def make_minimal_png(width, height, color_rgb):
    def png_chunk(name, data):
        c = struct.pack('>I', len(data)) + name + data
        return c + struct.pack('>I', zlib.crc32(c[4:]) & 0xffffffff)
    signature = b'\x89PNG\r\n\x1a\n'
    ihdr = png_chunk(b'IHDR', struct.pack('>IIBBBBB', width, height, 8, 2, 0, 0, 0))
    raw = b''.join(b'\x00' + bytes(color_rgb) * width for _ in range(height))
    idat = png_chunk(b'IDAT', zlib.compress(raw))
    iend = png_chunk(b'IEND', b'')
    return signature + ihdr + idat + iend

png = make_minimal_png(32, 32, [99, 102, 241])
with open('src-tauri/icons/icon.png', 'wb') as f:
    f.write(png)
print("Icon created")
EOF
```

- [ ] **Step 4.3: Commit build success**

```bash
cd /home/dev/open-prompt-code/open-prompt
# KHÔNG dùng git add -A (tránh commit build artifacts trong target/)
# Chỉ add docs và config nếu có thay đổi
git add docs/ scripts/ .github/ src-tauri/icons/ 2>/dev/null || true
git diff --staged --stat
git commit -m "chore: verify Tauri build thành công trên Linux" --allow-empty
```

---

## Task 5: Integration Test — First Run Flow

**Files:** Không có — chỉ test thủ công

> Yêu cầu: môi trường desktop với display (X11 hoặc Wayland). Nếu chạy headless, dùng Xvfb.

- [ ] **Step 5.1: Chuẩn bị môi trường test**

Nếu có display (kiểm tra):

```bash
echo $DISPLAY
```

Nếu không có display (headless), setup Xvfb:

```bash
# Cài Xvfb nếu cần
sudo apt-get install -y xvfb

# Chạy với virtual display
export DISPLAY=:99
Xvfb :99 -screen 0 1024x768x24 &
sleep 1
```

- [ ] **Step 5.2: Chạy app và kiểm tra first-run**

```bash
cd /home/dev/open-prompt-code/open-prompt
source ~/.cargo/env
./src-tauri/target/release/open-prompt &
APP_PID=$!
sleep 5
echo "App PID: $APP_PID"
```

Quan sát:
1. App khởi động không crash
2. System tray icon xuất hiện
3. Nhấn `Ctrl+Space` → overlay window mở

- [ ] **Step 5.3: Test first-run create account**

Trong overlay window:
1. Màn hình "Tạo tài khoản" xuất hiện
2. Nhập username và password (≥8 ký tự)
3. Submit → chuyển sang màn hình API key

- [ ] **Step 5.4: Test API key setup**

1. Nhập Claude API key (`sk-ant-...`)
2. Submit → chuyển sang overlay chính

- [ ] **Step 5.5: Test query (nếu có API key hợp lệ)**

1. Nhập query trong CommandInput
2. Nhấn Enter
3. Response xuất hiện từng token (streaming)

> Nếu không có API key hợp lệ: skip step này, test flow bỏ qua ("Bỏ qua" button).

- [ ] **Step 5.6: Kill app sau test**

```bash
kill $APP_PID 2>/dev/null
# Cleanup DB nếu muốn reset first-run state
rm -f ~/.open-prompt/open-prompt.db
```

---

## Task 6: Final Phase 1 Checklist

- [ ] **Step 6.1: Verify toàn bộ checklist Phase 1**

Chạy từng item và mark kết quả:

```bash
# 1. Go engine khởi động với OP_SOCKET_SECRET
cd /home/dev/open-prompt-code/open-prompt/go-engine
OP_SOCKET_SECRET=test-secret-32bytes-000000000000 ./bin/go-engine-linux-amd64 &
ENGINE_PID=$!
sleep 2
kill -0 $ENGINE_PID 2>/dev/null && echo "Engine OK" || echo "Engine FAIL"
kill $ENGINE_PID 2>/dev/null
echo "---"

# 2. Go tests pass (skip anthropic integration)
go test ./... -run 'Test' -skip 'TestAnthropicProvider' 2>&1 | grep -E "^ok|FAIL"
echo "---"

# 3. TypeScript build
cd /home/dev/open-prompt-code/open-prompt
npm run build 2>&1 | tail -3
echo "---"

# 4. Tauri binary tồn tại
ls -la src-tauri/target/release/open-prompt 2>/dev/null && echo "Tauri binary OK" || echo "Tauri binary MISSING — chạy Task 4 trước"
```

- [ ] **Step 6.2: Commit final**

```bash
cd /home/dev/open-prompt-code/open-prompt
git add -A
git commit -m "feat: hoàn thành Phase 1 - core loop overlay AI assistant

- Go engine: JSON-RPC server, auth, database, Anthropic streaming
- Tauri: global hotkey Ctrl+Space, overlay window, system tray
- React: onboarding, login, API key setup, streaming response UI
- CI: GitHub Actions test + build workflows"
```

---

## Phase 1 Checklist

- [ ] Go engine khởi động với `OP_SOCKET_SECRET`
- [ ] Go tests pass: auth, db, api (skip anthropic integration test)
- [ ] Tauri cargo build thành công
- [ ] `Ctrl+Space` mở overlay window
- [ ] First-run flow: tạo tài khoản → nhập API key
- [ ] Gõ query → nhận streaming response từ Claude (cần API key)
- [ ] System tray icon + Quit menu
- [ ] GitHub Actions CI workflows tồn tại (`ls .github/workflows/` → test.yml, build.yml)

---

## Troubleshooting

### Tauri build lỗi "No such file or directory: libwebkit2gtk"

```bash
sudo apt-get install -y libwebkit2gtk-4.1-dev
```

### Go engine không tạo socket

Kiểm tra permissions `/tmp/`:
```bash
ls -la /tmp/open-prompt.sock 2>/dev/null || echo "socket không tồn tại (engine chưa chạy)"
```

### App không hiện overlay

Kiểm tra display:
```bash
echo $DISPLAY   # phải có giá trị, ví dụ :0 hoặc :99
```

### React build lỗi TypeScript

```bash
cd /home/dev/open-prompt-code/open-prompt
npm run build 2>&1 | grep "error TS"
```

---

## Phase tiếp theo

Sau khi Phase 1 checklist hoàn thành:

- **Plan 2:** Provider System (auto-detect, OAuth Copilot/Gemini, token manager)
- **Plan 3:** Prompt & Skill System (slash commands, CRUD, template engine)
- **Plan 4:** Input Injection (clipboard backup/restore, simulated typing)
- **Plan 5:** Analytics & Full Settings UI
