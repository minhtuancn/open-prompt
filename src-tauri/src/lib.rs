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
