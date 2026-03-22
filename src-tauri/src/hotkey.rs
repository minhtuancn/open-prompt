use tauri::AppHandle;
use tauri_plugin_global_shortcut::{GlobalShortcutExt, ShortcutState};

pub fn register_hotkey(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    app.global_shortcut()
        .on_shortcut("Ctrl+Space", move |app_handle, _shortcut, event| {
            if event.state == ShortcutState::Pressed {
                crate::window::toggle_overlay(app_handle);
            }
        })?;
    Ok(())
}
