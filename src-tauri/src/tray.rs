use tauri::menu::{MenuBuilder, MenuItemBuilder};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Error};

pub fn setup_tray(app: &AppHandle) -> Result<(), Error> {
    let open = MenuItemBuilder::with_id("open", "Open (Ctrl+Space)").build(app)?;
    let quit = MenuItemBuilder::with_id("quit", "Quit Open Prompt").build(app)?;
    let menu = MenuBuilder::new(app).items(&[&open, &quit]).build()?;

    let icon = app.default_window_icon().cloned()
        .expect("no app icon found");

    TrayIconBuilder::new()
        .icon(icon)
        .menu(&menu)
        .tooltip("Open Prompt")
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => crate::window::toggle_overlay(app),
            "quit" => app.exit(0),
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                crate::window::toggle_overlay(tray.app_handle());
            }
        })
        .build(app)?;

    Ok(())
}
