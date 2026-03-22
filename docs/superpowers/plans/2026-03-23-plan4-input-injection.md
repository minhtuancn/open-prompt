# Input Injection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sau khi nhận AI response, user nhấn nút "Insert ↵" để inject toàn bộ response text vào ứng dụng đang được focus trước đó (editor, browser, terminal, v.v.).

**Architecture:** Rust module `injection.rs` dùng `enigo` crate để backup clipboard → paste text → restore clipboard; Go Engine thêm `context_detector.go` để detect tên app/window đang focus trước khi overlay mở; Tauri command `inject_text` được React gọi qua `invoke()`; `window.rs` lưu trữ focused window handle trước khi show overlay.

**Tech Stack:** Rust `enigo 0.2` (X11 feature on Linux), Go stdlib (`os/exec` + `xprop`), React `@tauri-apps/api/core`, zustand store, TailwindCSS

---

## Spec Reference

`docs/superpowers/specs/2026-03-22-open-prompt-design.md`

---

## File Map

```
src-tauri/
├── Cargo.toml                         ← thêm enigo dependency
└── src/
    ├── lib.rs                         ← đăng ký module injection + command
    ├── window.rs                      ← thêm FocusedWindow state + save/restore focus
    └── injection.rs                   ← NEW: inject_text command, clipboard logic

go-engine/
└── engine/
    └── context_detector.go            ← NEW: detect active window trước khi overlay mở
    └── context_detector_test.go       ← NEW: unit test cho context detector

src/
├── components/overlay/
│   └── ResponsePanel.tsx              ← thêm nút "Insert ↵"
└── store/
    └── overlayStore.ts                ← thêm activeApp field (nhận từ Go)
```

---

## Task 1: Thêm `enigo` vào Cargo.toml

**Files:**
- Modify: `src-tauri/Cargo.toml`

- [ ] **Step 1.1: Đọc Cargo.toml hiện tại để xác nhận vị trí thêm dependency**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cat Cargo.toml
  ```

- [ ] **Step 1.2: Thêm `enigo` vào `[dependencies]`**

  Trong block `[dependencies]`, thêm dòng sau (sau `rand = "0.8"`):

  ```toml
  # Enigo: keyboard/mouse simulation và clipboard access để inject text
  [target.'cfg(target_os = "linux")'.dependencies]
  enigo = { version = "0.2", features = ["x11"] }

  [target.'cfg(not(target_os = "linux"))'.dependencies]
  enigo = { version = "0.2" }
  ```

  > **Lưu ý:** `enigo 0.2` trên Linux yêu cầu feature `x11`. Trên macOS/Windows không cần flag đặc biệt.

- [ ] **Step 1.3: Verify compile được**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo check 2>&1 | tail -20
  ```

  Kết quả mong đợi: `Finished` hoặc chỉ warnings, không có errors.

- [ ] **Step 1.4: Commit**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && git add src-tauri/Cargo.toml src-tauri/Cargo.lock && git commit -m "chore: thêm enigo 0.2 dependency cho input injection"
  ```

---

## Task 2: Go Engine — Context Detector

**Files:**
- Create: `go-engine/engine/context_detector.go`
- Create: `go-engine/engine/context_detector_test.go`

> Detect tên ứng dụng đang focus **trước khi** overlay mở. Go Engine gọi hàm này và trả về trong RPC response (hoặc lưu state nội bộ để Rust query qua method mới `context.get_active`).

- [ ] **Step 2.1: Tạo thư mục `engine/` nếu chưa có**

  ```bash
  mkdir -p /home/dev/open-prompt-code/open-prompt/go-engine/engine
  ```

- [ ] **Step 2.2: Viết failing test trước**

  Tạo file `/home/dev/open-prompt-code/open-prompt/go-engine/engine/context_detector_test.go`:

  ```go
  package engine_test

  import (
      "testing"

      "github.com/minhtuancn/open-prompt/go-engine/engine"
  )

  func TestGetActiveWindow_ReturnsStruct(t *testing.T) {
      // Kiểm tra hàm trả về struct hợp lệ (không panic, không nil)
      info := engine.GetActiveWindow()
      // AppName và WindowTitle có thể rỗng trong CI/headless, nhưng struct phải non-nil
      if info == nil {
          t.Fatal("GetActiveWindow() trả về nil, mong đợi *WindowInfo")
      }
  }

  func TestGetActiveWindow_IgnoresOpenPrompt(t *testing.T) {
      // Kiểm tra hàm không trả về thông tin của chính open-prompt
      // (test này pass nếu binary không chạy trong open-prompt window)
      info := engine.GetActiveWindow()
      if info == nil {
          t.Skip("không có display, bỏ qua")
      }
      // AppName không được là "open-prompt" khi chạy từ terminal test
      // (chỉ verify struct fields tồn tại và có type đúng)
      _ = info.AppName
      _ = info.WindowTitle
      _ = info.IsTerminal
  }

  func TestIsTerminalApp(t *testing.T) {
      cases := []struct {
          name     string
          appName  string
          expected bool
      }{
          {"alacritty là terminal", "alacritty", true},
          {"kitty là terminal", "kitty", true},
          {"gnome-terminal là terminal", "gnome-terminal", true},
          {"wt là terminal", "wt", true},
          {"WindowsTerminal là terminal", "WindowsTerminal", true},
          {"code không phải terminal", "code", false},
          {"chrome không phải terminal", "google-chrome", false},
          {"empty string", "", false},
      }

      for _, tc := range cases {
          t.Run(tc.name, func(t *testing.T) {
              got := engine.IsTerminalApp(tc.appName)
              if got != tc.expected {
                  t.Errorf("IsTerminalApp(%q) = %v, want %v", tc.appName, got, tc.expected)
              }
          })
      }
  }
  ```

- [ ] **Step 2.3: Chạy test để xác nhận FAIL (compile error vì chưa có implementation)**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./engine/... 2>&1 | head -20
  ```

  Kết quả mong đợi: compile error `undefined: engine.GetActiveWindow`.

- [ ] **Step 2.4: Implement `context_detector.go`**

  Tạo file `/home/dev/open-prompt-code/open-prompt/go-engine/engine/context_detector.go`:

  ```go
  // Package engine chứa các utilities nội bộ của Go Engine
  package engine

  import (
      "os/exec"
      "runtime"
      "strings"
  )

  // WindowInfo chứa thông tin về cửa sổ đang được focus
  type WindowInfo struct {
      AppName     string // tên executable của ứng dụng (vd: "code", "firefox")
      WindowTitle string // tiêu đề cửa sổ
      IsTerminal  bool   // true nếu là terminal emulator
  }

  // terminalApps là danh sách các terminal emulator phổ biến
  var terminalApps = []string{
      "alacritty", "kitty", "gnome-terminal", "konsole",
      "xterm", "urxvt", "tilix", "wezterm",
      "wt", "WindowsTerminal",         // Windows Terminal
      "Terminal", "iTerm2",             // macOS
  }

  // IsTerminalApp kiểm tra xem appName có phải là terminal emulator không
  func IsTerminalApp(appName string) bool {
      if appName == "" {
          return false
      }
      lower := strings.ToLower(appName)
      for _, t := range terminalApps {
          if strings.ToLower(t) == lower {
              return true
          }
      }
      return false
  }

  // GetActiveWindow trả về thông tin cửa sổ đang được focus.
  // Trả về nil nếu không thể detect (headless, không có display, v.v.).
  func GetActiveWindow() *WindowInfo {
      switch runtime.GOOS {
      case "linux":
          return getActiveWindowLinux()
      case "darwin":
          return getActiveWindowMacOS()
      case "windows":
          return getActiveWindowWindows()
      default:
          return &WindowInfo{}
      }
  }

  // getActiveWindowLinux đọc active window trên Linux qua xprop + _NET_ACTIVE_WINDOW
  func getActiveWindowLinux() *WindowInfo {
      // Bước 1: lấy window ID đang active
      out, err := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
      if err != nil {
          // xprop không có hoặc không có DISPLAY → trả về empty struct
          return &WindowInfo{}
      }

      // Kết quả dạng: "_NET_ACTIVE_WINDOW(WINDOW): window id # 0x3200003"
      line := strings.TrimSpace(string(out))
      parts := strings.Fields(line)
      if len(parts) < 5 {
          return &WindowInfo{}
      }
      windowID := parts[len(parts)-1]

      // Bước 2: lấy WM_CLASS (tên app) của window đó
      classOut, err := exec.Command("xprop", "-id", windowID, "WM_CLASS").Output()
      if err != nil {
          return &WindowInfo{}
      }

      // Bước 3: lấy WM_NAME (window title)
      titleOut, _ := exec.Command("xprop", "-id", windowID, "_NET_WM_NAME").Output()

      appName := parseWMClass(string(classOut))
      windowTitle := parseWMName(string(titleOut))

      return &WindowInfo{
          AppName:     appName,
          WindowTitle: windowTitle,
          IsTerminal:  IsTerminalApp(appName),
      }
  }

  // parseWMClass trích xuất tên app từ output của xprop WM_CLASS
  // Input: `WM_CLASS(STRING) = "alacritty", "Alacritty"`
  // Output: "alacritty"
  func parseWMClass(raw string) string {
      // Tìm phần sau dấu "="
      idx := strings.Index(raw, "=")
      if idx < 0 {
          return ""
      }
      rest := strings.TrimSpace(raw[idx+1:])
      // Lấy giá trị đầu tiên trong quotes
      parts := strings.Split(rest, ",")
      if len(parts) == 0 {
          return ""
      }
      name := strings.Trim(strings.TrimSpace(parts[0]), `"`)
      return name
  }

  // parseWMName trích xuất window title từ output của xprop _NET_WM_NAME
  // Input: `_NET_WM_NAME(UTF8_STRING) = "main.go - open-prompt"`
  // Output: "main.go - open-prompt"
  func parseWMName(raw string) string {
      idx := strings.Index(raw, "=")
      if idx < 0 {
          return ""
      }
      title := strings.TrimSpace(raw[idx+1:])
      title = strings.Trim(title, `"`)
      return title
  }

  // getActiveWindowMacOS stub cho macOS (CGWindowListCopy)
  // TODO Plan 5: implement CGWindowListCopy qua cgo hoặc osascript
  func getActiveWindowMacOS() *WindowInfo {
      // Dùng osascript để lấy tên app đang active (không cần cgo)
      out, err := exec.Command("osascript", "-e",
          `tell application "System Events" to get name of first process whose frontmost is true`,
      ).Output()
      if err != nil {
          return &WindowInfo{}
      }
      appName := strings.TrimSpace(string(out))
      return &WindowInfo{
          AppName:    appName,
          IsTerminal: IsTerminalApp(appName),
      }
  }

  // getActiveWindowWindows stub cho Windows (GetForegroundWindow)
  // TODO Plan 5: implement qua syscall GetForegroundWindow + GetWindowText
  func getActiveWindowWindows() *WindowInfo {
      // Stub: trả về empty struct, Windows injection vẫn hoạt động
      // vì enigo trên Windows dùng SendInput (không cần window handle)
      return &WindowInfo{}
  }
  ```

- [ ] **Step 2.5: Chạy test để xác nhận PASS**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./engine/... -v 2>&1
  ```

  Kết quả mong đợi:
  - `TestIsTerminalApp` — PASS (8/8 sub-tests)
  - `TestGetActiveWindow_ReturnsStruct` — PASS (trả về non-nil struct)
  - `TestGetActiveWindow_IgnoresOpenPrompt` — PASS hoặc SKIP (headless)

- [ ] **Step 2.6: Commit**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && git add go-engine/engine/ && git commit -m "feat: thêm context detector để detect active window trước khi overlay mở"
  ```

---

## Task 3: Go Engine — Thêm RPC method `context.get_active`

**Files:**
- Modify: `go-engine/api/router.go`
- Create: `go-engine/api/handlers_context.go`

> Rust cần query tên app đang focus ngay trước khi `toggle_overlay()` hiển thị overlay. Go Engine expose method `context.get_active` trả về `{app_name, window_title, is_terminal}`.

- [ ] **Step 3.1: Tạo handler file**

  Tạo file `/home/dev/open-prompt-code/open-prompt/go-engine/api/handlers_context.go`:

  ```go
  package api

  import (
      "github.com/minhtuancn/open-prompt/go-engine/engine"
  )

  // handleContextGetActive trả về thông tin cửa sổ đang focus
  func (r *Router) handleContextGetActive(req *Request) (interface{}, *RPCError) {
      info := engine.GetActiveWindow()
      if info == nil {
          info = &engine.WindowInfo{}
      }

      return map[string]interface{}{
          "app_name":     info.AppName,
          "window_title": info.WindowTitle,
          "is_terminal":  info.IsTerminal,
      }, nil
  }
  ```

- [ ] **Step 3.2: Đăng ký method vào router**

  Mở `/home/dev/open-prompt-code/open-prompt/go-engine/api/router.go`, trong hàm `dispatch()`, thêm case mới trước `default`:

  ```go
  case "context.get_active":
      return r.handleContextGetActive(req)
  ```

  Vị trí chính xác — thêm sau dòng `case "query.stream":`:

  ```go
  // ... existing cases ...
  case "query.stream":
      return r.handleQueryStream(conn, req)
  case "context.get_active":
      // Trả về thông tin ứng dụng đang được focus trước khi overlay mở
      return r.handleContextGetActive(req)
  default:
  ```

- [ ] **Step 3.3: Build Go engine để verify**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/go-engine && go build ./... 2>&1
  ```

  Kết quả mong đợi: không có output (build thành công).

- [ ] **Step 3.4: Chạy toàn bộ Go tests**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/go-engine && go test ./... -count=1 2>&1
  ```

  Kết quả mong đợi: tất cả PASS.

- [ ] **Step 3.5: Commit**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && git add go-engine/api/handlers_context.go go-engine/api/router.go && git commit -m "feat: thêm RPC method context.get_active để query active window"
  ```

---

## Task 4: Rust — `window.rs` lưu trữ focused window state

**Files:**
- Modify: `src-tauri/src/window.rs`
- Modify: `src-tauri/src/lib.rs`

> Trước khi show overlay, Rust gọi `context.get_active` để lấy app name, lưu vào Tauri state `FocusedApp`. Sau khi inject, cần biết app nào để xử lý fallback.

- [ ] **Step 4.1: Xem lại `window.rs` và `lib.rs` hiện tại** (đã đọc ở trên — xác nhận `window.rs` chỉ có `toggle_overlay`)

- [ ] **Step 4.2: Thêm `FocusedApp` state vào `window.rs`**

  Thay toàn bộ nội dung `/home/dev/open-prompt-code/open-prompt/src-tauri/src/window.rs`:

  ```rust
  use std::sync::Mutex;
  use tauri::{AppHandle, Manager};

  /// FocusedApp lưu thông tin ứng dụng đang được focus trước khi overlay hiện ra
  pub struct FocusedApp(pub Mutex<AppContext>);

  #[derive(Debug, Clone, Default)]
  pub struct AppContext {
      pub app_name: String,
      pub window_title: String,
      pub is_terminal: bool,
  }

  /// toggle_overlay hiện/ẩn overlay window.
  /// Trước khi show, query Go Engine để lấy active window context.
  pub fn toggle_overlay(app: &AppHandle) {
      if let Some(window) = app.get_webview_window("overlay") {
          if window.is_visible().unwrap_or(false) {
              // Ẩn overlay
              let _ = window.hide();
          } else {
              // Lưu context của app đang focus trước khi show overlay
              if let Ok(ctx) = query_active_context(app) {
                  // Chỉ lưu nếu không phải chính open-prompt
                  if ctx.app_name != "open-prompt" && ctx.app_name != "open_prompt" {
                      if let Ok(mut guard) = app.state::<FocusedApp>().0.lock() {
                          *guard = ctx;
                      }
                  }
              }
              let _ = window.show();
              let _ = window.set_focus();
          }
      }
  }

  /// query_active_context gọi Go Engine qua socket để lấy active window info.
  /// Dùng blocking call vì được gọi từ hotkey handler (không trong async context).
  fn query_active_context(app: &AppHandle) -> Result<AppContext, String> {
      use crate::ipc::call_engine_sync;
      let result = call_engine_sync(app, "context.get_active", serde_json::Value::Null)?;

      let app_name = result
          .get("app_name")
          .and_then(|v| v.as_str())
          .unwrap_or("")
          .to_string();
      let window_title = result
          .get("window_title")
          .and_then(|v| v.as_str())
          .unwrap_or("")
          .to_string();
      let is_terminal = result
          .get("is_terminal")
          .and_then(|v| v.as_bool())
          .unwrap_or(false);

      Ok(AppContext {
          app_name,
          window_title,
          is_terminal,
      })
  }
  ```

- [ ] **Step 4.3: Thêm `call_engine_sync` vào `ipc.rs`**

  Mở `/home/dev/open-prompt-code/open-prompt/src-tauri/src/ipc.rs`, thêm function sau (append vào cuối file):

  ```rust
  /// call_engine_sync là blocking version của call_engine, dùng khi không ở trong async context
  /// (vd: hotkey handler). Không hỗ trợ streaming.
  pub fn call_engine_sync(
      app: &AppHandle,
      method: &str,
      params: serde_json::Value,
  ) -> Result<serde_json::Value, String> {
      use std::io::{BufRead, BufReader, Write};

      let secret = app.state::<crate::sidecar::EngineSecret>().0.clone();

      let envelope = RpcEnvelope {
          secret,
          request: RpcRequestInner {
              jsonrpc: "2.0".into(),
              method: method.to_string(),
              params,
              id: 9999,
          },
      };

      let mut msg = serde_json::to_vec(&envelope).map_err(|e| e.to_string())?;
      msg.push(b'\n');

      #[cfg(unix)]
      {
          use std::os::unix::net::UnixStream;
          let mut conn =
              UnixStream::connect("/tmp/open-prompt.sock").map_err(|e| format!("connect: {e}"))?;
          conn.write_all(&msg).map_err(|e| e.to_string())?;
          let mut reader = BufReader::new(conn);
          let mut line = String::new();
          reader.read_line(&mut line).map_err(|e| e.to_string())?;
          let resp: RpcResponse =
              serde_json::from_str(line.trim()).map_err(|e| format!("parse: {e}"))?;
          if let Some(err) = resp.error {
              return Err(err.to_string());
          }
          return Ok(resp.result.unwrap_or(serde_json::Value::Null));
      }

      #[cfg(windows)]
      {
          use std::net::TcpStream;
          let port = app.state::<crate::sidecar::EnginePort>().0;
          let addr = format!("127.0.0.1:{port}");
          let mut conn = TcpStream::connect(&addr).map_err(|e| format!("connect {addr}: {e}"))?;
          conn.write_all(&msg).map_err(|e| e.to_string())?;
          let mut reader = BufReader::new(conn);
          let mut line = String::new();
          reader.read_line(&mut line).map_err(|e| e.to_string())?;
          let resp: RpcResponse =
              serde_json::from_str(line.trim()).map_err(|e| format!("parse: {e}"))?;
          if let Some(err) = resp.error {
              return Err(err.to_string());
          }
          return Ok(resp.result.unwrap_or(serde_json::Value::Null));
      }

      #[allow(unreachable_code)]
      Err("platform không được hỗ trợ".to_string())
  }
  ```

- [ ] **Step 4.4: Đăng ký `FocusedApp` state vào `lib.rs`**

  Mở `/home/dev/open-prompt-code/open-prompt/src-tauri/src/lib.rs`, thêm:

  1. Thêm `mod injection;` vào đầu file (sau `mod window;`)
  2. Thêm `manage(window::FocusedApp(std::sync::Mutex::new(window::AppContext::default())))` vào builder
  3. Thêm `injection::inject_text` vào `invoke_handler`

  Kết quả `lib.rs`:

  ```rust
  mod hotkey;
  mod injection;
  mod ipc;
  mod sidecar;
  mod tray;
  mod window;

  pub use sidecar::{EnginePort, EngineSecret, SidecarState};

  pub fn run() {
      tauri::Builder::default()
          .plugin(tauri_plugin_global_shortcut::Builder::new().build())
          .plugin(tauri_plugin_shell::init())
          .manage(SidecarState(std::sync::Mutex::new(None)))
          .manage(EnginePort(0))
          .manage(window::FocusedApp(std::sync::Mutex::new(
              window::AppContext::default(),
          )))
          .invoke_handler(tauri::generate_handler![
              ipc::call_engine,
              injection::inject_text,
          ])
          .setup(|app| {
              sidecar::spawn_engine(app.handle())
                  .expect("failed to spawn go engine");
              tray::setup_tray(app.handle())?;
              hotkey::register_hotkey(app.handle())?;
              Ok(())
          })
          .run(tauri::generate_context!())
          .expect("error running tauri application");
  }
  ```

- [ ] **Step 4.5: `cargo check` để verify (injection.rs chưa có nên sẽ báo lỗi module — bình thường)**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo check 2>&1 | grep -E "^error" | head -10
  ```

  Kết quả mong đợi: lỗi `file not found for module \`injection\`` — đúng, sẽ fix ở Task 5.

---

## Task 5: Rust — `injection.rs` — Tauri command `inject_text`

**Files:**
- Create: `src-tauri/src/injection.rs`

> Module này backup clipboard, set text vào clipboard, simulate Ctrl+V, đợi 200ms, restore clipboard. Fallback cho terminal: dùng enigo gõ từng ký tự.

- [ ] **Step 5.1: Tạo `injection.rs`**

  Tạo file `/home/dev/open-prompt-code/open-prompt/src-tauri/src/injection.rs`:

  ```rust
  use std::thread;
  use std::time::Duration;

  use enigo::{
      Direction::{Click, Press, Release},
      Enigo, Key, Keyboard, Settings,
  };
  use tauri::{command, AppHandle, Manager};

  use crate::window::FocusedApp;

  /// inject_text là Tauri command được React gọi để inject text vào ứng dụng đang focus.
  /// Flow:
  ///   1. Đọc FocusedApp state để biết app_name và is_terminal
  ///   2. Nếu app là terminal → dùng enigo type simulation
  ///   3. Ngược lại → backup clipboard, paste, restore clipboard
  #[command]
  pub async fn inject_text(app: AppHandle, text: String) -> Result<(), String> {
      // Đọc context của app đang focus
      let (app_name, is_terminal) = {
          let state = app.state::<FocusedApp>();
          let guard = state.0.lock().map_err(|e| e.to_string())?;
          (guard.app_name.clone(), guard.is_terminal)
      };

      // Không inject vào chính open-prompt
      if app_name == "open-prompt" || app_name == "open_prompt" || app_name.is_empty() {
          // app_name rỗng nghĩa là chưa detect được → vẫn thử inject
          if app_name == "open-prompt" || app_name == "open_prompt" {
              return Err("không inject vào chính open-prompt".to_string());
          }
      }

      let text_clone = text.clone();

      // Chạy blocking operation trên thread riêng (enigo không async)
      tauri::async_runtime::spawn_blocking(move || {
          if is_terminal {
              inject_via_typing(&text_clone)
          } else {
              inject_via_clipboard(&text_clone)
          }
      })
      .await
      .map_err(|e| e.to_string())?
  }

  /// inject_via_clipboard: backup clipboard → set text → Ctrl+V → wait 200ms → restore
  fn inject_via_clipboard(text: &str) -> Result<(), String> {
      // Bước 1: Đọc clipboard hiện tại để backup
      let original_clipboard = read_clipboard();

      // Bước 2: Set text vào clipboard
      write_clipboard(text).map_err(|e| format!("ghi clipboard thất bại: {e}"))?;

      // Bước 3: Đợi 50ms để clipboard propagate
      thread::sleep(Duration::from_millis(50));

      // Bước 4: Simulate Ctrl+V để paste
      let mut enigo = Enigo::new(&Settings::default()).map_err(|e| format!("init enigo: {e}"))?;

      enigo
          .key(Key::Control, Press)
          .map_err(|e| format!("press Ctrl: {e}"))?;
      enigo
          .key(Key::Unicode('v'), Click)
          .map_err(|e| format!("press V: {e}"))?;
      enigo
          .key(Key::Control, Release)
          .map_err(|e| format!("release Ctrl: {e}"))?;

      // Bước 5: Đợi 200ms sau khi paste
      thread::sleep(Duration::from_millis(200));

      // Bước 6: Restore clipboard gốc
      if let Some(original) = original_clipboard {
          let _ = write_clipboard(&original); // không fail nếu restore lỗi
      }

      Ok(())
  }

  /// inject_via_typing: gõ từng ký tự bằng enigo (dùng cho terminal)
  fn inject_via_typing(text: &str) -> Result<(), String> {
      let mut enigo = Enigo::new(&Settings::default()).map_err(|e| format!("init enigo: {e}"))?;

      // enigo::text() gõ toàn bộ string một lần (hiệu quả hơn char by char)
      enigo
          .text(text)
          .map_err(|e| format!("type text: {e}"))?;

      Ok(())
  }

  /// read_clipboard đọc nội dung clipboard hiện tại.
  /// Trả về None nếu clipboard rỗng hoặc không đọc được.
  #[cfg(target_os = "linux")]
  fn read_clipboard() -> Option<String> {
      // Dùng xclip hoặc xsel để đọc clipboard trên Linux
      let output = std::process::Command::new("xclip")
          .args(["-selection", "clipboard", "-o"])
          .output()
          .or_else(|_| {
              std::process::Command::new("xsel")
                  .args(["--clipboard", "--output"])
                  .output()
          })
          .ok()?;

      if output.status.success() {
          Some(String::from_utf8_lossy(&output.stdout).to_string())
      } else {
          None
      }
  }

  #[cfg(target_os = "macos")]
  fn read_clipboard() -> Option<String> {
      let output = std::process::Command::new("pbpaste").output().ok()?;
      if output.status.success() {
          Some(String::from_utf8_lossy(&output.stdout).to_string())
      } else {
          None
      }
  }

  #[cfg(target_os = "windows")]
  fn read_clipboard() -> Option<String> {
      // Windows: dùng PowerShell Get-Clipboard
      let output = std::process::Command::new("powershell")
          .args(["-Command", "Get-Clipboard"])
          .output()
          .ok()?;
      if output.status.success() {
          Some(String::from_utf8_lossy(&output.stdout).to_string())
      } else {
          None
      }
  }

  /// write_clipboard ghi text vào clipboard.
  #[cfg(target_os = "linux")]
  fn write_clipboard(text: &str) -> Result<(), String> {
      use std::io::Write;
      use std::process::{Command, Stdio};

      // Thử xclip trước
      let result = (|| -> Result<(), String> {
          let mut child = Command::new("xclip")
              .args(["-selection", "clipboard"])
              .stdin(Stdio::piped())
              .spawn()
              .map_err(|e| e.to_string())?;
          if let Some(stdin) = child.stdin.as_mut() {
              stdin.write_all(text.as_bytes()).map_err(|e| e.to_string())?;
          }
          child.wait().map_err(|e| e.to_string())?;
          Ok(())
      })();

      if result.is_ok() {
          return Ok(());
      }

      // Fallback: xsel
      let mut child = std::process::Command::new("xsel")
          .args(["--clipboard", "--input"])
          .stdin(Stdio::piped())
          .spawn()
          .map_err(|e| format!("xsel không có: {e}"))?;
      if let Some(stdin) = child.stdin.as_mut() {
          use std::io::Write;
          stdin.write_all(text.as_bytes()).map_err(|e| e.to_string())?;
      }
      child.wait().map_err(|e| e.to_string())?;
      Ok(())
  }

  #[cfg(target_os = "macos")]
  fn write_clipboard(text: &str) -> Result<(), String> {
      use std::io::Write;
      use std::process::{Command, Stdio};
      let mut child = Command::new("pbcopy")
          .stdin(Stdio::piped())
          .spawn()
          .map_err(|e| format!("pbcopy: {e}"))?;
      if let Some(stdin) = child.stdin.as_mut() {
          stdin.write_all(text.as_bytes()).map_err(|e| e.to_string())?;
      }
      child.wait().map_err(|e| e.to_string())?;
      Ok(())
  }

  #[cfg(target_os = "windows")]
  fn write_clipboard(text: &str) -> Result<(), String> {
      use std::process::Command;
      // Set-Clipboard qua PowerShell
      Command::new("powershell")
          .args(["-Command", &format!("Set-Clipboard -Value '{}'", text.replace('\'', "''"))])
          .status()
          .map_err(|e| format!("powershell: {e}"))?;
      Ok(())
  }
  ```

- [ ] **Step 5.2: `cargo check` để verify toàn bộ Rust compile được**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo check 2>&1
  ```

  Kết quả mong đợi: `Finished` hoặc chỉ warnings (không có errors).

  > **Nếu có lỗi về enigo API:** `enigo 0.2` thay đổi API so với `0.1`. Kiểm tra với:
  > ```bash
  > cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo doc --open --package enigo 2>&1 | head -5
  > ```
  > Điều chỉnh import paths nếu cần (`enigo::Enigo`, `enigo::Key`, `enigo::Keyboard` trait, `enigo::Direction`).

- [ ] **Step 5.3: Build release để đảm bảo không có linker errors**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo build 2>&1 | tail -20
  ```

- [ ] **Step 5.4: Commit**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && git add src-tauri/src/injection.rs src-tauri/src/ipc.rs src-tauri/src/window.rs src-tauri/src/lib.rs && git commit -m "feat: implement inject_text Tauri command dùng enigo để inject vào focused app"
  ```

---

## Task 6: React — Thêm nút "Insert ↵" vào `ResponsePanel.tsx`

**Files:**
- Modify: `src/components/overlay/ResponsePanel.tsx`

> Nút chỉ hiện khi có text và không đang streaming. Khi click: gọi `invoke('inject_text', {text})`, show loading state, handle error.

- [ ] **Step 6.1: Xem lại `ResponsePanel.tsx` hiện tại** (đã đọc — component hiện tại chỉ có text display)

- [ ] **Step 6.2: Cập nhật `ResponsePanel.tsx`**

  Thay toàn bộ nội dung `/home/dev/open-prompt-code/open-prompt/src/components/overlay/ResponsePanel.tsx`:

  ```tsx
  import { useState } from 'react'
  import { invoke } from '@tauri-apps/api/core'
  import { useOverlayStore } from '../../store/overlayStore'

  export function ResponsePanel() {
    const { chunks, isStreaming, error } = useOverlayStore()
    const text = chunks.join('')

    // State cho inject button
    const [isInjecting, setIsInjecting] = useState(false)
    const [injectError, setInjectError] = useState<string | null>(null)
    const [injected, setInjected] = useState(false)

    if (!text && !isStreaming && !error) return null

    // Xử lý khi user nhấn nút "Insert ↵"
    const handleInject = async () => {
      if (!text || isInjecting) return
      setIsInjecting(true)
      setInjectError(null)
      setInjected(false)
      try {
        await invoke('inject_text', { text })
        setInjected(true)
        // Reset trạng thái injected sau 2 giây
        setTimeout(() => setInjected(false), 2000)
      } catch (err) {
        setInjectError(err as string)
      } finally {
        setIsInjecting(false)
      }
    }

    return (
      <div className="px-5 pb-4 max-h-80 overflow-y-auto">
        <div className="border-t border-white/10 pt-3">
          {error ? (
            <p className="text-red-400 text-sm">{error}</p>
          ) : (
            <>
              <p className="text-white/90 text-sm leading-relaxed whitespace-pre-wrap">
                {text}
                {isStreaming && <span className="animate-pulse">▌</span>}
              </p>

              {/* Nút Insert chỉ hiện khi có text và không đang stream */}
              {text && !isStreaming && (
                <div className="mt-3 flex items-center gap-2">
                  <button
                    onClick={handleInject}
                    disabled={isInjecting}
                    className={`
                      flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium
                      transition-all duration-150
                      ${injected
                        ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                        : 'bg-white/10 text-white/70 border border-white/20 hover:bg-white/20 hover:text-white'
                      }
                      disabled:opacity-50 disabled:cursor-not-allowed
                    `}
                    title="Chèn text vào ứng dụng đang focus"
                  >
                    {isInjecting ? (
                      <>
                        <span className="animate-spin">⟳</span>
                        <span>Đang chèn...</span>
                      </>
                    ) : injected ? (
                      <>
                        <span>✓</span>
                        <span>Đã chèn</span>
                      </>
                    ) : (
                      <>
                        <span>Insert ↵</span>
                      </>
                    )}
                  </button>

                  {/* Hiện lỗi nếu inject thất bại */}
                  {injectError && (
                    <span className="text-red-400 text-xs">{injectError}</span>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    )
  }
  ```

- [ ] **Step 6.3: TypeScript check**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && npx tsc --noEmit 2>&1
  ```

  Kết quả mong đợi: không có errors.

- [ ] **Step 6.4: Build React để verify**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && npm run build 2>&1 | tail -20
  ```

- [ ] **Step 6.5: Commit**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && git add src/components/overlay/ResponsePanel.tsx && git commit -m "feat: thêm nút Insert vào ResponsePanel để inject AI response vào focused app"
  ```

---

## Task 7: Integration Test — Chạy full app và test thủ công

> Không có unit test tự động cho injection (cần display, clipboard, window focus thực tế). Test thủ công theo checklist.

- [ ] **Step 7.1: Build toàn bộ Go engine**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt/go-engine && go build -o ../src-tauri/binaries/go-engine-x86_64-unknown-linux-gnu . 2>&1
  ```

- [ ] **Step 7.2: Build Tauri app ở dev mode**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && source ~/.cargo/env && npm run tauri dev 2>&1 &
  ```

- [ ] **Step 7.3: Test checklist thủ công**

  Mở một text editor (vd: gedit, VS Code, terminal). Nhấn `Ctrl+Space` để mở overlay:

  - [ ] Gõ query, nhận streaming response
  - [ ] Sau khi stream xong, nút "Insert ↵" hiện ra
  - [ ] Click "Insert ↵" → text được paste vào editor (kiểm tra clipboard restore)
  - [ ] Mở terminal (alacritty/kitty), nhấn `Ctrl+Space`, nhận response
  - [ ] Click "Insert ↵" trong terminal → text được gõ bằng keyboard simulation
  - [ ] Không inject khi active window là open-prompt
  - [ ] Nút hiện trạng thái "Đang chèn..." khi đang xử lý
  - [ ] Sau 2 giây: trạng thái trở về bình thường

- [ ] **Step 7.4: Commit cuối**

  ```bash
  cd /home/dev/open-prompt-code/open-prompt && git add -p && git commit -m "feat: hoàn thành Plan 4 - Input Injection feature"
  ```

---

## Xử lý lỗi thường gặp

### `enigo` API khác với 0.2

Nếu `cargo check` báo lỗi về API:

```bash
cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo add enigo@0.2 --features x11 2>&1
```

Kiểm tra API thực tế:
```bash
cd /home/dev/open-prompt-code/open-prompt/src-tauri && source ~/.cargo/env && cargo doc -p enigo 2>&1 && ls target/doc/enigo/
```

Tham chiếu chính xác từ docs:
- `Enigo::new(&Settings::default())` → `Result<Enigo, NewConError>`
- `enigo.key(Key::Control, Direction::Press)` → `Result<(), InputError>`
- `enigo.text("hello")` → `Result<(), InputError>`

### `xclip`/`xsel` không có trên hệ thống

```bash
sudo apt-get install xclip   # Ubuntu/Debian
# hoặc
sudo apt-get install xsel
```

### Wayland thay vì X11

Nếu desktop dùng Wayland, `xdotool`/`xprop` sẽ không hoạt động:
- Dùng `wl-clipboard` thay `xclip`: `sudo apt-get install wl-clipboard`
- `wl-copy` để write, `wl-paste` để read
- Cập nhật `read_clipboard()` và `write_clipboard()` để detect và dùng `wl-paste`/`wl-copy`

### `context.get_active` timeout

Nếu Go Engine chưa ready khi hotkey được nhấn, `query_active_context` sẽ fail:
- `toggle_overlay` đã handle lỗi gracefully (`if let Ok(ctx)`) → vẫn show overlay
- `inject_text` sẽ dùng clipboard mode mặc định (vì `app_name` sẽ là empty string từ default `AppContext`)

---

## Summary

| Task | Files | Trạng thái |
|------|-------|------------|
| 1. Thêm enigo | `Cargo.toml` | - [ ] |
| 2. Go context detector | `go-engine/engine/context_detector.go` + test | - [ ] |
| 3. RPC method context.get_active | `go-engine/api/handlers_context.go`, `router.go` | - [ ] |
| 4. Rust FocusedApp state | `src-tauri/src/window.rs`, `ipc.rs`, `lib.rs` | - [ ] |
| 5. Rust injection.rs | `src-tauri/src/injection.rs` | - [ ] |
| 6. React Insert button | `src/components/overlay/ResponsePanel.tsx` | - [ ] |
| 7. Integration test | Manual checklist | - [ ] |
