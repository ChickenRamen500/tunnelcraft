// TunnelCraft Tauri 2.0 Backend
// Spawns the Go daemon as a sidecar and manages the application window.

mod commands;
mod grpc_client;

use std::process::{Child, Command};
use std::sync::Mutex;
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{Manager};

// Global state to manage daemon process
struct DaemonState {
    process: Option<Child>,
}

impl DaemonState {
    fn new() -> Self {
        DaemonState { process: None }
    }
}

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

    println!("[tauri] attempting to spawn daemon at: {:?}", daemon_path);

    Command::new(&daemon_path)
        .spawn()
}

/// Check if daemon HTTP API is reachable
fn wait_for_daemon_ready(max_retries: u8) -> bool {
    for i in 0..max_retries {
        if std::net::TcpStream::connect("127.0.0.1:50052").is_ok() {
            println!("[tauri] daemon is ready after {} retries", i);
            return true;
        }
        std::thread::sleep(std::time::Duration::from_millis(500));
    }
    false
}

#[tauri::command]
fn get_daemon_status() -> serde_json::Value {
    let healthy = std::net::TcpStream::connect("127.0.0.1:50052").is_ok();
    serde_json::json!({
        "running": healthy,
        "http_api": "http://127.0.0.1:50052"
    })
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // Try to spawn the daemon
    let daemon_started = match spawn_daemon() {
        Ok(child) => {
            println!("[tauri] tunnelcraftd.exe spawned successfully");
            // Wait for daemon to be ready
            if wait_for_daemon_ready(20) {
                Some(child)
            } else {
                eprintln!("[tauri] daemon failed to become ready");
                let _ = child.kill();
                None
            }
        }
        Err(e) => {
            eprintln!("[tauri] failed to spawn tunnelcraftd.exe: {}", e);
            eprintln!("[tauri] will try to connect to existing daemon");
            None
        }
    };

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(Mutex::new(DaemonState { process: daemon_started }))
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
            
            // Update tray tooltip based on daemon status
            let app_handle = app.handle().clone();
            std::thread::spawn(move || {
                std::thread::sleep(std::time::Duration::from_secs(2));
                let status = if std::net::TcpStream::connect("127.0.0.1:50052").is_ok() {
                    "TunnelCraft - Работает"
                } else {
                    "TunnelCraft - Ошибка демона"
                };
                if let Some(tray) = app_handle.tray_by_id("main-tray") {
                    let _ = tray.set_tooltip(Some(status));
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running TunnelCraft");

    // Kill daemon on exit
    println!("[tauri] shutting down, cleaning up daemon process...");
}

fn main() {
    run();
}
