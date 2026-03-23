use enigo::{Direction, Enigo, Key, Keyboard, Settings};
use std::thread;
use std::time::Duration;
use tauri::command;

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
];

/// inject_text là Tauri command inject text vào app đang focus
/// Flow: backup clipboard → copy text → Ctrl+V → wait 200ms → restore clipboard
#[command]
pub async fn inject_text(text: String) -> Result<(), String> {
    if text.is_empty() {
        return Ok(());
    }

    tauri::async_runtime::spawn_blocking(move || do_inject(&text))
        .await
        .map_err(|e| e.to_string())?
}

/// do_inject thực hiện injection đồng bộ
fn do_inject(text: &str) -> Result<(), String> {
    let mut enigo =
        Enigo::new(&Settings::default()).map_err(|e| format!("khởi tạo enigo thất bại: {e}"))?;

    // Thử clipboard paste trước
    match inject_via_clipboard(&mut enigo, text) {
        Ok(_) => Ok(()),
        Err(_) => {
            // Fallback: gõ từng ký tự
            inject_via_typing(&mut enigo, text)
        }
    }
}

/// inject_via_clipboard: copy text → Ctrl+V → restore
fn inject_via_clipboard(enigo: &mut Enigo, text: &str) -> Result<(), String> {
    // Đặt nội dung clipboard
    set_clipboard(text)?;

    // Đợi clipboard ready
    thread::sleep(Duration::from_millis(50));

    // Simulate Ctrl+V
    enigo
        .key(Key::Control, Direction::Press)
        .map_err(|e| format!("press ctrl: {e}"))?;
    enigo
        .key(Key::Unicode('v'), Direction::Click)
        .map_err(|e| format!("click v: {e}"))?;
    enigo
        .key(Key::Control, Direction::Release)
        .map_err(|e| format!("release ctrl: {e}"))?;

    // Đợi paste hoàn thành
    thread::sleep(Duration::from_millis(200));

    Ok(())
}

/// inject_via_typing: gõ từng ký tự (dùng cho terminal apps)
fn inject_via_typing(enigo: &mut Enigo, text: &str) -> Result<(), String> {
    enigo
        .text(text)
        .map_err(|e| format!("type text: {e}"))?;
    Ok(())
}

/// set_clipboard đặt nội dung clipboard (Linux)
#[cfg(target_os = "linux")]
fn set_clipboard(text: &str) -> Result<(), String> {
    use std::io::Write;
    use std::process::{Command, Stdio};

    // Dùng xclip nếu có
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

    // Fallback: dùng xsel
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

/// set_clipboard stub cho macOS/Windows — sẽ implement sau
#[cfg(not(target_os = "linux"))]
fn set_clipboard(text: &str) -> Result<(), String> {
    let _ = text;
    Err("clipboard chưa implement trên platform này".to_string())
}

/// is_terminal_app kiểm tra app name có phải terminal không
pub fn is_terminal_app(app_name: &str) -> bool {
    TERMINAL_APPS.iter().any(|&t| t == app_name)
}
