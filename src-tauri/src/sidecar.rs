use std::io::{BufRead, BufReader};
use std::process::{Child, Command, Stdio};
use std::sync::Mutex;
use tauri::{AppHandle, Manager};

/// SidecarState giữ process handle của Go engine
pub struct SidecarState(pub Mutex<Option<Child>>);

/// EngineSecret lưu shared secret để IPC dùng
pub struct EngineSecret(pub String);

/// EnginePort lưu TCP port (chỉ dùng trên Windows)
#[allow(dead_code)]
pub struct EnginePort(pub u16);

/// Spawn Go engine sidecar và đợi "ready" signal từ stdout
pub fn spawn_engine(app: &AppHandle) -> Result<(), String> {
    // Tạo shared secret ngẫu nhiên (64 hex chars = 32 bytes)
    let secret: String = (0..32)
        .map(|_| format!("{:02x}", rand::random::<u8>()))
        .collect();

    // Lưu secret vào app state để IPC dùng
    app.manage(EngineSecret(secret.clone()));

    let sidecar_path = app
        .path()
        .resolve("binaries/go-engine", tauri::path::BaseDirectory::Resource)
        .map_err(|e| format!("resolve sidecar path: {e}"))?;

    let mut child = Command::new(&sidecar_path)
        .env("OP_SOCKET_SECRET", &secret)
        .stdout(Stdio::piped())
        .spawn()
        .map_err(|e| format!("spawn go engine: {e}"))?;

    // Đợi "ready" từ stdout
    let stdout = child.stdout.take().unwrap();
    let mut reader = BufReader::new(stdout);
    let mut line = String::new();
    reader
        .read_line(&mut line)
        .map_err(|e| format!("read stdout: {e}"))?;

    if !line.trim().eq("ready") {
        return Err(format!("unexpected startup output: {line}"));
    }

    // Lưu child process
    let state = app.state::<SidecarState>();
    *state.0.lock().unwrap() = Some(child);

    Ok(())
}
