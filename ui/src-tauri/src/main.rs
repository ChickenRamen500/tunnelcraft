// TunnelCraft Tauri 2.0 Backend
// Connects to the Go daemon via gRPC and exposes commands to the React frontend.

mod grpc_client;
mod commands;

use commands::*;
use grpc_client::GrpcClient;
use std::sync::Mutex;
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::Manager;

#[tauri::command]
fn get_connection_status(client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.get_connection_status()
}

#[tauri::command]
async fn connect_server(server_id: String, client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.connect_server(&server_id).await
}

#[tauri::command]
async fn disconnect_server(force: bool, client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.disconnect_server(force).await
}

#[tauri::command]
fn list_servers(client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.list_servers()
}

#[tauri::command]
fn list_subscriptions(client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.list_subscriptions()
}

#[tauri::command]
async fn refresh_subscription(id: String, client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.refresh_subscription(&id).await
}

#[tauri::command]
fn get_settings(client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.get_settings()
}

#[tauri::command]
fn get_routing_rules(client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.get_routing_rules()
}

#[tauri::command]
fn health_check(client: tauri::State<'_, Mutex<GrpcClient>>) -> Result<serde_json::Value, String> {
    let client = client.lock().map_err(|e| e.to_string())?;
    client.health_check()
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let grpc_client = Mutex::new(GrpcClient::new("http://127.0.0.1:50051"));

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(grpc_client)
        .invoke_handler(tauri::generate_handler![
            get_connection_status,
            connect_server,
            disconnect_server,
            list_servers,
            list_subscriptions,
            refresh_subscription,
            get_settings,
            get_routing_rules,
            health_check,
        ])
        .setup(|app| {
            // Build system tray with connect/disconnect toggle
            let tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .tooltip("TunnelCraft - Disconnected")
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        // Toggle window visibility on tray click
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

            // Store tray handle for later tooltip updates
            app.manage(tray);

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running TunnelCraft");
}

fn main() {
    run();
}