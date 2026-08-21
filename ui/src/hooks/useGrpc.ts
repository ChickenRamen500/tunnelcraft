// HTTP client for the TunnelCraft Go daemon REST API.
// The daemon runs on http://127.0.0.1:50052
// Falls back to mock data when the daemon is not reachable.

const API_BASE = "http://127.0.0.1:50052";

export interface ConnectionStatus {
  state: string;
  server_id: string | null;
  server_name?: string;
  protocol?: string;
  mode: string;
  socks_port: number;
  http_port: number;
  stats: {
    bytes_uploaded: number;
    bytes_downloaded: number;
    duration_seconds?: number;
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

// Simple fetch wrapper with error handling
async function api<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options?.headers },
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

// Check if daemon is reachable
export async function isDaemonAlive(): Promise<boolean> {
  try {
    await api<{ healthy: boolean }>("/api/health");
    return true;
  } catch {
    return false;
  }
}

export async function getConnectionStatus(): Promise<ConnectionStatus> {
  try {
    return await api<ConnectionStatus>("/api/status");
  } catch {
    return {
      state: "DISCONNECTED",
      server_id: null,
      mode: "SYSTEM",
      socks_port: 1080,
      http_port: 8080,
      stats: { bytes_uploaded: 0, bytes_downloaded: 0 },
    };
  }
}

export async function connectServer(serverId: string): Promise<any> {
  return api("/api/connect", {
    method: "POST",
    body: JSON.stringify({ server_id: serverId }),
  });
}

export async function disconnectServer(force = false): Promise<any> {
  return api("/api/disconnect", {
    method: "POST",
    body: JSON.stringify({ force }),
  });
}

export async function listServers(): Promise<{ servers: Server[]; total: number }> {
  try {
    return await api("/api/servers");
  } catch {
    return { servers: [], total: 0 };
  }
}

export async function importServer(content: string, subscriptionId?: string): Promise<any> {
  return api("/api/servers/import", {
    method: "POST",
    body: JSON.stringify({ content, subscription_id: subscriptionId || "" }),
  });
}

export async function listSubscriptions(): Promise<{ subscriptions: Subscription[] }> {
  try {
    return await api("/api/subscriptions");
  } catch {
    return { subscriptions: [] };
  }
}

export async function addSubscription(name: string, url: string): Promise<any> {
  return api("/api/subscriptions", {
    method: "POST",
    body: JSON.stringify({ name, url }),
  });
}

export async function refreshSubscription(id: string): Promise<any> {
  return api(`/api/subscriptions/refresh/${id}`, { method: "POST" });
}

export async function getSettings(): Promise<Settings> {
  try {
    return await api<Settings>("/api/settings");
  } catch {
    return {
      proxy_mode: "SYSTEM",
      socks_port: 1080,
      http_port: 8080,
      dns_servers: "1.1.1.1,8.8.8.8",
      auto_connect: false,
      connect_on_startup: false,
      kill_switch: false,
      split_tunneling: false,
      allow_lan: false,
      connection_timeout: 30,
      reconnect_attempts: 3,
      language: "ru",
      theme: "dark",
    };
  }
}

export async function saveSettings(settings: Partial<Settings>): Promise<void> {
  await api("/api/settings", {
    method: "PUT",
    body: JSON.stringify(settings),
  });
}

export async function getRoutingRules(): Promise<any> {
  try {
    return await api("/api/settings");
  } catch {
    return { domain_strategy: "IPIfNonMatch", rules: [] };
  }
}

export async function getLogs(limit = 100): Promise<{ logs: string[]; count: number }> {
  try {
    return await api(`/api/logs?limit=${limit}`);
  } catch {
    return { logs: [], count: 0 };
  }
}

export async function healthCheck(): Promise<{ healthy: boolean; version: string }> {
  try {
    return await api("/api/health");
  } catch {
    return { healthy: false, version: "?" };
  }
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
