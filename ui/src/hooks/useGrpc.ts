// Tauri invoke wrapper for gRPC calls.
// Falls back to mock data when running outside Tauri (e.g. in browser).

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

// Check if we're running inside Tauri
const isTauri = (): boolean => {
  return typeof window !== "undefined" && "__TAURI__" in window;
};

// Lazy-loaded invoke function
let _invoke: Function | null = null;
async function getInvoke(): Promise<Function> {
  if (_invoke) return _invoke;
  if (!isTauri()) throw new Error("Not running in Tauri");
  const mod = await import("@tauri-apps/api/core");
  _invoke = mod.invoke;
  return _invoke;
}

// Mock defaults
const mockStatus: ConnectionStatus = {
  state: "DISCONNECTED",
  server_id: null,
  mode: "SYSTEM",
  socks_port: 1080,
  http_port: 8080,
  stats: { bytes_uploaded: 0, bytes_downloaded: 0, duration: null },
};

const mockSettings: Settings = {
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

export async function getConnectionStatus(): Promise<ConnectionStatus> {
  if (!isTauri()) return mockStatus;
  const invoke = await getInvoke();
  return invoke<ConnectionStatus>("get_connection_status");
}

export async function connectServer(serverId: string): Promise<any> {
  if (!isTauri()) return { state: "CONNECTED", server_id: serverId };
  const invoke = await getInvoke();
  return invoke("connect_server", { serverId });
}

export async function disconnectServer(force = false): Promise<any> {
  if (!isTauri()) return { state: "DISCONNECTED" };
  const invoke = await getInvoke();
  return invoke("disconnect_server", { force });
}

export async function listServers(): Promise<{ servers: Server[]; total: number }> {
  if (!isTauri()) return { servers: [], total: 0 };
  const invoke = await getInvoke();
  return invoke("list_servers");
}

export async function listSubscriptions(): Promise<{ subscriptions: Subscription[] }> {
  if (!isTauri()) return { subscriptions: [] };
  const invoke = await getInvoke();
  return invoke("list_subscriptions");
}

export async function refreshSubscription(id: string): Promise<any> {
  if (!isTauri()) return { added: 0, updated: 0, removed: 0 };
  const invoke = await getInvoke();
  return invoke("refresh_subscription", { id });
}

export async function getSettings(): Promise<Settings> {
  if (!isTauri()) return { ...mockSettings };
  const invoke = await getInvoke();
  return invoke<Settings>("get_settings");
}

export async function getRoutingRules(): Promise<any> {
  if (!isTauri()) return { domain_strategy: "IPIfNonMatch", rules: [] };
  const invoke = await getInvoke();
  return invoke("get_routing_rules");
}

export async function healthCheck(): Promise<{ healthy: boolean; version: string }> {
  if (!isTauri()) return { healthy: true, version: "0.1.0" };
  const invoke = await getInvoke();
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
