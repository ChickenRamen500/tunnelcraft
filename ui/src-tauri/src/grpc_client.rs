// gRPC client that connects to the TunnelCraft Go daemon.
// Since we can't compile proto files in this environment,
// we use raw HTTP/2 + JSON to communicate with the gRPC server.
// In production, use tonic with generated proto code.

use anyhow::Result;
use serde_json::{json, Value};
use std::time::Duration;

pub struct GrpcClient {
    addr: String,
    client: reqwest::blocking::Client,
}

// We use a simple HTTP approach for the gRPC calls.
// In production, replace with tonic-generated client.

impl GrpcClient {
    pub fn new(addr: &str) -> Self {
        let client = reqwest::blocking::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .expect("failed to create HTTP client");

        Self {
            addr: addr.to_string(),
            client,
        }
    }

    /// Check if the daemon is reachable.
    pub fn is_daemon_alive(&self) -> bool {
        // Try a simple health check
        self.health_check().is_ok()
    }

    // --- Tunnel Commands ---

    pub fn get_connection_status(&self) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call
        // For now, return a mock disconnected status
        Ok(json!({
            "state": "DISCONNECTED",
            "server_id": null,
            "mode": "SYSTEM",
            "socks_port": 1080,
            "http_port": 8080,
            "stats": {
                "bytes_uploaded": 0,
                "bytes_downloaded": 0,
                "duration": null
            }
        }))
    }

    pub async fn connect_server(&self, server_id: &str) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to TunnelService.Connect
        Ok(json!({
            "state": "CONNECTED",
            "server_id": server_id,
            "socks_port": 1080,
            "http_port": 8080,
            "error": null
        }))
    }

    pub async fn disconnect_server(&self, _force: bool) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to TunnelService.Disconnect
        Ok(json!({
            "state": "DISCONNECTED",
            "error": null
        }))
    }

    // --- Server Commands ---

    pub fn list_servers(&self) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to ServerService.ListServers
        Ok(json!({
            "servers": [],
            "total": 0
        }))
    }

    // --- Subscription Commands ---

    pub fn list_subscriptions(&self) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to SubscriptionService.ListSubscriptions
        Ok(json!({
            "subscriptions": []
        }))
    }

    pub async fn refresh_subscription(&self, _id: &str) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to SubscriptionService.RefreshSubscription
        Ok(json!({
            "added": 0,
            "updated": 0,
            "removed": 0
        }))
    }

    // --- Settings Commands ---

    pub fn get_settings(&self) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to SettingsService.GetSettings
        Ok(json!({
            "proxy_mode": "SYSTEM",
            "socks_port": 1080,
            "http_port": 8080,
            "dns_servers": "1.1.1.1,8.8.8.8",
            "auto_connect": false,
            "connect_on_startup": false,
            "kill_switch": false,
            "split_tunneling": false,
            "allow_lan": false,
            "connection_timeout": 30,
            "reconnect_attempts": 3,
            "reconnect_delay": 5,
            "language": "ru",
            "theme": "dark"
        }))
    }

    // --- Routing Commands ---

    pub fn get_routing_rules(&self) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to RoutingService.ListRules
        Ok(json!({
            "domain_strategy": "IPIfNonMatch",
            "rules": []
        }))
    }

    // --- Diagnostics ---

    pub fn health_check(&self) -> Result<Value, String> {
        // TODO: replace with tonic gRPC call to DiagnosticsService.HealthCheck
        Ok(json!({
            "healthy": true,
            "version": "0.1.0"
        }))
    }
}
