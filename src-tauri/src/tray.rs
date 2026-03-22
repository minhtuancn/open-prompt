use tauri::menu::{MenuBuilder, MenuItemBuilder};
use tauri::tray::TrayIconBuilder;
use tauri::{AppHandle, Error};

pub fn setup_tray(app: &AppHandle) -> Result<(), Error> {
    let quit = MenuItemBuilder::with_id("quit", "Quit Open Prompt").build(app)?;
    let menu = MenuBuilder::new(app).items(&[&quit]).build()?;

    TrayIconBuilder::new()
        .menu(&menu)
        .on_menu_event(|app, event| {
            if event.id == "quit" {
                app.exit(0);
            }
        })
        .build(app)?;

    Ok(())
}
