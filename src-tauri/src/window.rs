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
            let _ = window.hide();
        } else {
            // Lưu context của app đang focus trước khi show overlay
            if let Ok(ctx) = query_active_context(app) {
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
