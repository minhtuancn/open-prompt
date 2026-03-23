use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::io::{BufRead, BufReader, Write};
use tauri::{command, AppHandle, Emitter, Manager};

use crate::sidecar::EngineSecret;

#[derive(Serialize, Deserialize, Debug)]
struct RpcEnvelope {
    secret: String,
    request: RpcRequestInner,
}

#[derive(Serialize, Deserialize, Debug)]
struct RpcRequestInner {
    jsonrpc: String,
    method: String,
    params: Value,
    id: u64,
}

#[derive(Serialize, Deserialize, Debug)]
struct RpcResponse {
    jsonrpc: Option<String>,
    result: Option<Value>,
    error: Option<Value>,
    id: Option<Value>,
    // Notification fields
    method: Option<String>,
    params: Option<Value>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct StreamChunk {
    pub delta: String,
    pub done: bool,
    pub error: Option<String>,
}

/// call_engine gọi Go Engine qua socket
/// - Với query.stream: đọc notifications và emit "stream-chunk" events
/// - Với các method khác: đọc 1 response và return
#[command]
pub async fn call_engine(
    app: AppHandle,
    method: String,
    params: Value,
) -> Result<Value, String> {
    let secret = app.state::<EngineSecret>().0.clone();
    let is_streaming = method == "query.stream";

    let envelope = RpcEnvelope {
        secret,
        request: RpcRequestInner {
            jsonrpc: "2.0".into(),
            method: method.clone(),
            params,
            id: 1,
        },
    };

    let mut msg = serde_json::to_vec(&envelope).map_err(|e| e.to_string())?;
    msg.push(b'\n');

    let app_clone = app.clone();
    tauri::async_runtime::spawn_blocking(move || {
        #[cfg(unix)]
        {
            use std::os::unix::net::UnixStream;
            let mut conn = UnixStream::connect("/tmp/open-prompt.sock")
                .map_err(|e| format!("connect: {e}"))?;
            conn.write_all(&msg).map_err(|e| e.to_string())?;
            handle_response(BufReader::new(conn), is_streaming, &app_clone)
        }
        #[cfg(windows)]
        {
            let port = app_clone.state::<crate::sidecar::EnginePort>().0;
            let addr = format!("127.0.0.1:{port}");
            let mut conn = std::net::TcpStream::connect(&addr)
                .map_err(|e| format!("connect {addr}: {e}"))?;
            conn.write_all(&msg).map_err(|e| e.to_string())?;
            handle_response(BufReader::new(conn), is_streaming, &app_clone)
        }
    })
    .await
    .map_err(|e| e.to_string())?
}

/// call_engine_sync là blocking version của call_engine, dùng khi không ở trong async context.
/// Không hỗ trợ streaming.
pub fn call_engine_sync(
    app: &AppHandle,
    method: &str,
    params: serde_json::Value,
) -> Result<serde_json::Value, String> {
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

    #[cfg(not(any(unix, windows)))]
    Err("platform không được hỗ trợ".to_string())
}

/// handle_response đọc response(s) từ socket
fn handle_response<R: BufRead>(
    mut reader: R,
    is_streaming: bool,
    app: &AppHandle,
) -> Result<Value, String> {
    loop {
        let mut line = String::new();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let msg: RpcResponse = serde_json::from_str(line)
            .map_err(|e| format!("parse response: {e} — raw: {line}"))?;

        // Nếu là notification (stream.chunk)
        if msg.method.as_deref() == Some("stream.chunk") {
            if let Some(params) = msg.params {
                let chunk: StreamChunk = serde_json::from_value(params)
                    .map_err(|e| format!("parse chunk: {e}"))?;
                let done = chunk.done;
                app.emit("stream-chunk", chunk).map_err(|e| e.to_string())?;
                if done {
                    return Ok(Value::Null);
                }
            }
            continue;
        }

        // Regular JSON-RPC response
        if let Some(err) = msg.error {
            return Err(err.to_string());
        }
        return Ok(msg.result.unwrap_or(Value::Null));
    }
}
