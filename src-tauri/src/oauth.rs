use serde::Deserialize;
use tauri::{command, AppHandle, Manager, WebviewUrl, WebviewWindowBuilder};

use crate::ipc::call_engine_sync;

#[derive(Deserialize)]
struct OAuthStartResponse {
    method: String,
    url: Option<String>,
    user_code: Option<String>,
    verification_uri: Option<String>,
    device_code: Option<String>,
    message: Option<String>,
}

/// start_oauth khởi tạo OAuth flow cho provider.
/// - "webview": mở WebView window với authorization URL
/// - "device_flow": trả về device code cho user nhập thủ công
/// - "browser": mở browser hệ thống (fallback)
#[command]
pub async fn start_oauth(app: AppHandle, provider: String) -> Result<serde_json::Value, String> {
    // Gọi Go Engine để lấy OAuth URL/method
    let token = ""; // OAuth start không cần auth token
    let params = serde_json::json!({ "token": token, "provider": provider });
    let result = call_engine_sync(&app, "providers.oauth_start", params)
        .map_err(|e| format!("oauth_start failed: {e}"))?;

    let response: OAuthStartResponse =
        serde_json::from_value(result.clone()).map_err(|e| format!("parse response: {e}"))?;

    match response.method.as_str() {
        "webview" => {
            // Mở WebView window nhỏ cho OAuth login
            if let Some(url) = &response.url {
                let oauth_window = WebviewWindowBuilder::new(
                    &app,
                    "oauth",
                    WebviewUrl::External(url.parse().map_err(|e| format!("parse url: {e}"))?),
                )
                .title("Đăng nhập OAuth")
                .inner_size(480.0, 640.0)
                .resizable(true)
                .center()
                .build()
                .map_err(|e| format!("create oauth window: {e}"))?;

                // Monitor navigation — detect redirect về open-prompt://
                let app_handle = app.clone();
                oauth_window.on_navigation(move |url| {
                    let url_str = url.as_str();
                    if url_str.starts_with("open-prompt://oauth") {
                        // Extract code từ URL
                        if let Some(code) = url.query_pairs().find(|(k, _)| k == "code").map(|(_, v)| v.to_string()) {
                            // Gọi oauth_finish
                            let params = serde_json::json!({
                                "token": "",
                                "provider": &provider,
                                "code": code,
                            });
                            let _ = call_engine_sync(&app_handle, "providers.oauth_finish", params);
                        }
                        // Đóng OAuth window
                        if let Some(win) = app_handle.get_webview_window("oauth") {
                            let _ = win.close();
                        }
                        return false; // Không navigate
                    }
                    true // Cho phép navigate
                });
            }
            Ok(result)
        }
        "device_flow" => {
            // Device Flow — trả về device code cho frontend hiển thị
            Ok(result)
        }
        _ => {
            // Browser fallback — mở URL trong browser hệ thống
            if let Some(url) = &response.url {
                let _ = open::that(url);
            }
            Ok(result)
        }
    }
}

/// poll_oauth polling Device Flow
#[command]
pub async fn poll_oauth(app: AppHandle, provider: String, device_code: String) -> Result<serde_json::Value, String> {
    let params = serde_json::json!({
        "token": "",
        "provider": provider,
        "device_code": device_code,
    });
    call_engine_sync(&app, "providers.oauth_poll", params)
        .map_err(|e| format!("poll failed: {e}"))
}
