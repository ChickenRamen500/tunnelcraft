// Tauri invoke wrapper for gRPC calls.
import { invoke } from "@tauri-apps/api/core";

export interface ConnectionStatus {
  state: string;
  server_id: string | null;
  mode: string;
  socks_port: number;
  http_port: number;
  stats: {
    bytes_uploaded: number;
    bytes_downloaded: number;
    duration: string | null;
  };
}

export interface Server {
  id: string;
  name: string;
  host: string;
  port: number;
  protocol: string;
  favorite: boolean;
  tags: string[];
  subscription_id: string;
}

export interface Subscription {
  id: string;
  name: string;
  url: string;
  enabled: boolean;
  refresh_interval: number;
  server_count: number;
}

export interface RoutingRule {
  id: string;
  name: string;
  enabled: boolean;
  action: string;
  domains: string[];
  ip_cidrs: string[];
  geoip_codes: string[];
  processes: string[];
}

export interface Settings {
  proxy_mode: string;
  socks_port: number;
  http_port: number;
  dns_servers: string;
  auto_connect: boolean;
  connect_on_startup: boolean;
  kill_switch: boolean;
  split_tunneling: boolean;
  allow_lan: boolean;
  connection_timeout: number;
  reconnect_attempts: number;
  language: string;
  theme: string;
}

export async function getConnectionStatus(): Promise<ConnectionStatus> {
  return invoke<ConnectionStatus>("get_connection_status");
}

export async function connectServer(serverId: string): Promise<any> {
  return invoke("connect_server", { serverId });
}

export async function disconnectServer(force = false): Promise<any> {
  return invoke("disconnect_server", { force });
}

export async function listServers(): Promise<{ servers: Server[]; total: number }> {
  return invoke("list_servers");
}

export async function listSubscriptions(): Promise<{ subscriptions: Subscription[] }> {
  return invoke("list_subscriptions");
}

export async function refreshSubscription(id: string): Promise<any> {
  return invoke("refresh_subscription", { id });
}

export async function getSettings(): Promise<Settings> {
  return invoke<Settings>("get_settings");
}

export async function getRoutingRules(): Promise<any> {
  return invoke("get_routing_rules");
}

export async function healthCheck(): Promise<{ healthy: boolean; version: string }> {
  return invoke("health_check");
}

// Format bytes to human-readable
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

// Format duration in seconds to HH:MM:SS
export function formatDuration(seconds: number | null): string {
  if (!seconds) return "00:00:00";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  return `${h.toString().padStart(2, "0")}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}