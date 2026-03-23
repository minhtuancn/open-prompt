use enigo::{Direction, Enigo, Key, Keyboard, Settings};
use std::thread;
use std::time::Duration;
use tauri::{command, AppHandle, Manager};

use crate::window::FocusedApp;

/// Danh sách tên process của terminal emulators
const TERMINAL_APPS: &[&str] = &[
    "alacritty",
    "kitty",
    "gnome-terminal",
    "xterm",
    "xfce4-terminal",
    "konsole",
    "tilix",
    "wt",
    "WindowsTerminal",
    "iTerm2",
    "Terminal",
    "wezterm",
    "foot",
];

/// inject_text inject text vào app đang focus.
/// Nếu app là terminal → dùng typing (tránh paste issue).
/// Nếu app khác → clipboard paste với backup/restore.
#[command]
pub async fn inject_text(app: AppHandle, text: String) -> Result<String, String> {
    if text.is_empty() {
        return Ok("empty".to_string());
    }

    // Lấy context focused app
    let is_terminal = {
        let state = app.state::<FocusedApp>();
        let guard = state.0.lock().map_err(|e| format!("lock: {e}"))?;
        guard.is_terminal || is_terminal_app(&guard.app_name)
    };

    let app_name = {
        let state = app.state::<FocusedApp>();
        let guard = state.0.lock().map_err(|e| format!("lock: {e}"))?;
        guard.app_name.clone()
    };

    tauri::async_runtime::spawn_blocking(move || do_inject(&text, is_terminal))
        .await
        .map_err(|e| e.to_string())??;

    Ok(app_name)
}

/// do_inject thực hiện injection đồng bộ
fn do_inject(text: &str, is_terminal: bool) -> Result<(), String> {
    let mut enigo =
        Enigo::new(&Settings::default()).map_err(|e| format!("khởi tạo enigo thất bại: {e}"))?;

    if is_terminal {
        // Terminal: gõ từng ký tự (tránh paste issues trong terminal)
        inject_via_typing(&mut enigo, text)
    } else {
        // Non-terminal: clipboard paste với backup/restore
        let backup = get_clipboard();
        let result = inject_via_clipboard(&mut enigo, text);
        // Restore clipboard nếu có backup
        if let Ok(ref original) = backup {
            thread::sleep(Duration::from_millis(100));
            let _ = set_clipboard(original);
        }
        result
    }
}

/// inject_via_clipboard: copy text → Ctrl+V
fn inject_via_clipboard(enigo: &mut Enigo, text: &str) -> Result<(), String> {
    set_clipboard(text)?;
    thread::sleep(Duration::from_millis(50));

    enigo
        .key(Key::Control, Direction::Press)
        .map_err(|e| format!("press ctrl: {e}"))?;
    enigo
        .key(Key::Unicode('v'), Direction::Click)
        .map_err(|e| format!("click v: {e}"))?;
    enigo
        .key(Key::Control, Direction::Release)
        .map_err(|e| format!("release ctrl: {e}"))?;

    thread::sleep(Duration::from_millis(200));
    Ok(())
}

/// inject_via_typing: gõ text qua enigo
fn inject_via_typing(enigo: &mut Enigo, text: &str) -> Result<(), String> {
    enigo
        .text(text)
        .map_err(|e| format!("type text: {e}"))?;
    Ok(())
}

/// get_clipboard đọc clipboard hiện tại (Linux)
#[cfg(target_os = "linux")]
fn get_clipboard() -> Result<String, String> {
    use std::process::Command;
    // Thử xclip
    if let Ok(output) = Command::new("xclip")
        .args(["-selection", "clipboard", "-o"])
        .output()
    {
        if output.status.success() {
            return Ok(String::from_utf8_lossy(&output.stdout).to_string());
        }
    }
    // Fallback: xsel
    if let Ok(output) = Command::new("xsel")
        .args(["--clipboard", "--output"])
        .output()
    {
        if output.status.success() {
            return Ok(String::from_utf8_lossy(&output.stdout).to_string());
        }
    }
    Err("không thể đọc clipboard".to_string())
}

#[cfg(not(target_os = "linux"))]
fn get_clipboard() -> Result<String, String> {
    Err("clipboard read chưa implement trên platform này".to_string())
}

/// set_clipboard đặt nội dung clipboard (Linux)
#[cfg(target_os = "linux")]
fn set_clipboard(text: &str) -> Result<(), String> {
    use std::io::Write;
    use std::process::{Command, Stdio};

    if let Ok(mut child) = Command::new("xclip")
        .args(["-selection", "clipboard"])
        .stdin(Stdio::piped())
        .spawn()
    {
        if let Some(mut stdin) = child.stdin.take() {
            let _ = stdin.write_all(text.as_bytes());
        }
        let _ = child.wait();
        return Ok(());
    }

    if let Ok(mut child) = Command::new("xsel")
        .args(["--clipboard", "--input"])
        .stdin(Stdio::piped())
        .spawn()
    {
        if let Some(mut stdin) = child.stdin.take() {
            let _ = stdin.write_all(text.as_bytes());
        }
        let _ = child.wait();
        return Ok(());
    }

    Err("xclip và xsel đều không có sẵn".to_string())
}

#[cfg(not(target_os = "linux"))]
fn set_clipboard(text: &str) -> Result<(), String> {
    let _ = text;
    Err("clipboard chưa implement trên platform này".to_string())
}

/// is_terminal_app kiểm tra app name có phải terminal không
pub fn is_terminal_app(app_name: &str) -> bool {
    TERMINAL_APPS
        .iter()
        .any(|&t| t.eq_ignore_ascii_case(app_name))
}
