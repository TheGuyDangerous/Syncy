use std::path::PathBuf;

use tauri::menu::{Menu, MenuItem, PredefinedMenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{Manager, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;
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

fn show_main(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.set_focus();
    }
}

fn spawn_engine(app: &tauri::AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let dir = engine_dir(app)?;
    kill_stale_engine(&dir);
    let (mut rx, child) = app
        .shell()
        .sidecar("syncyd")?
        .args(["--data-dir", &dir.to_string_lossy(), "--api", API_ADDR])
        .spawn()?;
    let _ = std::fs::write(dir.join("syncyd.pid"), child.pid().to_string());
    tauri::async_runtime::spawn(async move {
        let _child = child;
        while let Some(event) = rx.recv().await {
            if matches!(event, CommandEvent::Terminated(_)) {
                break;
            }
        }
    });
    Ok(())
}

fn kill_stale_engine(dir: &std::path::Path) {
    let Ok(raw) = std::fs::read_to_string(dir.join("syncyd.pid")) else {
        return;
    };
    let Ok(pid) = raw.trim().parse::<u32>() else {
        return;
    };
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        let _ = std::process::Command::new("taskkill")
            .args(["/F", "/PID", &pid.to_string()])
            .creation_flags(0x0800_0000)
            .output();
    }
    #[cfg(not(windows))]
    {
        let _ = std::process::Command::new("kill")
            .args(["-9", &pid.to_string()])
            .output();
    }
}

fn build_tray(app: &tauri::AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let open = MenuItem::with_id(app, "open", "Open Syncy", true, None::<&str>)?;
    let sep = PredefinedMenuItem::separator(app)?;
    let quit = MenuItem::with_id(app, "quit", "Quit Syncy", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&open, &sep, &quit])?;

    TrayIconBuilder::with_id("main")
        .icon(app.default_window_icon().unwrap().clone())
        .tooltip("Syncy")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => show_main(app),
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
                show_main(tray.app_handle());
            }
        })
        .build(app)?;
    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut builder = tauri::Builder::default();
    #[cfg(desktop)]
    {
        builder = builder.plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            show_main(app);
        }));
    }
    builder
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec![]),
        ))
        .setup(|app| {
            spawn_engine(app.handle())?;
            build_tray(app.handle())?;
            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                let _ = window.hide();
                api.prevent_close();
            }
        })
        .invoke_handler(tauri::generate_handler![app_version, daemon_info])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
