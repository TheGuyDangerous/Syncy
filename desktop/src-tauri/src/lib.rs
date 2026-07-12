use std::path::PathBuf;

use tauri::Manager;
use tauri_plugin_shell::process::CommandEvent;
use tauri_plugin_shell::ShellExt;

const API_ADDR: &str = "127.0.0.1:22062";

#[derive(serde::Serialize)]
struct DaemonInfo {
    base_url: String,
    token: String,
}

#[tauri::command]
fn app_version() -> String {
    env!("CARGO_PKG_VERSION").to_string()
}

#[tauri::command]
fn daemon_info(app: tauri::AppHandle) -> Result<DaemonInfo, String> {
    let dir = engine_dir(&app).map_err(|e| e.to_string())?;
    let token = std::fs::read_to_string(dir.join("api-token")).map_err(|e| e.to_string())?;
    Ok(DaemonInfo {
        base_url: format!("http://{API_ADDR}"),
        token: token.trim().to_string(),
    })
}

fn engine_dir(app: &tauri::AppHandle) -> Result<PathBuf, Box<dyn std::error::Error>> {
    let dir = app.path().app_data_dir()?.join("engine");
    std::fs::create_dir_all(&dir)?;
    Ok(dir)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            let dir = engine_dir(app.handle())?;
            let (mut rx, child) = app
                .shell()
                .sidecar("syncyd")?
                .args(["--data-dir", &dir.to_string_lossy(), "--api", API_ADDR])
                .spawn()?;
            tauri::async_runtime::spawn(async move {
                let _child = child;
                while let Some(event) = rx.recv().await {
                    if matches!(event, CommandEvent::Terminated(_)) {
                        break;
                    }
                }
            });
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![app_version, daemon_info])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
