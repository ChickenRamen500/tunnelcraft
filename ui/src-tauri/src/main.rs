// TunnelCraft Tauri 2.0 Backend
// Spawns the Go daemon and manages the application window.

mod commands;
mod grpc_client;

use std::process::{Child, Command};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::Manager;

static mut DAEMON_PROCESS: Option<Child> = None;

/// Spawn tunnelcraftd.exe from ../bin/ relative to the Tauri app.
fn spawn_daemon() -> std::io::Result<Child> {
    let exe_dir = std::env::current_exe()?
        .parent()
        .map(|p| p.to_path_buf())
        .unwrap_or_default();

    // In dev mode, the exe is in target/debug/, daemon is at ../../bin/
    // In production, both are in the same directory
    let daemon_path = exe_dir
        .join("..")
        .join("..")
        .join("bin")
        .join("tunnelcraftd.exe");

    // Fallback: same directory as the Tauri exe (production)
    let daemon_path = if daemon_path.exists() {
        daemon_path
    } else {
        exe_dir.join("tunnelcraftd.exe")
    };

    Command::new(&daemon_path)
        .spawn()
}

#[tauri::command]
fn get_daemon_status() -> serde_json::Value {
    // Try to reach the daemon's HTTP API
    let healthy = std::net::TcpStream::connect("127.0.0.1:50052").is_ok();
    serde_json::json!({
        "running": healthy,
        "http_api": "http://127.0.0.1:50052"
    })
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // Try to spawn the daemon
    match spawn_daemon() {
        Ok(_child) => {
            unsafe { DAEMON_PROCESS = Some(_child); }
            println!("[tauri] tunnelcraftd.exe spawned");
        }
        Err(e) => {
            eprintln!("[tauri] failed to spawn tunnelcraftd.exe: {}, will try to connect to existing daemon", e);
        }
    }

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![get_daemon_status])
        .setup(|app| {
            let tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .tooltip("TunnelCraft - Запуск...")
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        if let Some(window) = tray.app_handle().get_webview_window("main") {
                            if window.is_visible().unwrap_or(false) {
                                let _ = window.hide();
                            } else {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                    }
                })
                .build(app)?;

            app.manage(tray);
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running TunnelCraft");

    // Kill daemon on exit
    unsafe {
        if let Some(mut child) = DAEMON_PROCESS.take() {
            let _ = child.kill();
        }
    }
}

fn main() {
    run();
}