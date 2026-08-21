import { useEffect, useCallback, useState } from "react";
import { Power, ArrowDown, ArrowUp, Clock, Zap, AlertTriangle } from "lucide-react";
import { useConnectionStore } from "@/stores/connection";
import { getConnectionStatus, connectServer, disconnectServer, formatBytes } from "@/hooks/useApi";
import StatusBadge from "@/components/StatusBadge";
import SpeedGraph from "@/components/SpeedGraph";

export default function Dashboard() {
  const {
    status,
    activeServer,
    speedUp,
    speedDown,
    speedHistory,
    isConnecting,
    fallbackMode,
    setStatus,
    updateSpeed,
    setConnecting,
  } = useConnectionStore();

  const [lastServerId, setLastServerId] = useState<string | null>(null);

  const isConnected = status.state === "CONNECTED";
  const isDisconnected = status.state === "DISCONNECTED" || status.state === "ERROR";

  const fetchStatus = useCallback(async () => {
    try {
      const s = await getConnectionStatus();
      setStatus(s);
    } catch (e) {
      console.error("Failed to get status:", e);
    }
  }, [setStatus]);

  // Poll status every 2 seconds
  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 2000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  // Simulate speed updates when connected (in real app, from gRPC stream)
  useEffect(() => {
    if (!isConnected) {
      updateSpeed(0, 0);
      return;
    }
    const interval = setInterval(() => {
      const up = Math.random() * 500000 + 100000;
      const down = Math.random() * 2000000 + 500000;
      updateSpeed(up, down);
    }, 1000);
    return () => clearInterval(interval);
  }, [isConnected, updateSpeed]);

  const handleToggle = async () => {
    if (isConnected || isConnecting) {
      setConnecting(true);
      try {
        await disconnectServer(false);
      } finally {
        setConnecting(false);
      }
    } else {
      const serverId = lastServerId || status.server_id || "";
      if (!serverId) return;
      setConnecting(true);
      try {
        await connectServer(serverId);
        setLastServerId(serverId);
      } finally {
        setConnecting(false);
      }
    }
  };

  return (
    <div className="flex flex-col h-full fade-in">
      {/* Main connection area */}
      <div className="flex-1 flex flex-col items-center justify-center gap-6 px-6">
        {/* Status badge */}
        <StatusBadge state={status.state} fallbackMode={fallbackMode} />

        {/* Connect button */}
        <button
          onClick={handleToggle}
          disabled={isConnecting}
          className={`w-28 h-28 rounded-full flex items-center justify-center transition-all duration-300
            ${isConnected
              ? "bg-red-500/10 border-2 border-red-500/40 hover:bg-red-500/20 pulse-red"
              : isConnecting
                ? "bg-yellow-500/10 border-2 border-yellow-500/40 animate-pulse"
                : "bg-emerald-500/10 border-2 border-emerald-500/40 hover:bg-emerald-500/20 hover:border-emerald-400/60"
            }`}
        >
          <Power
            size={40}
            className={isConnected ? "text-red-400" : isConnecting ? "text-yellow-400" : "text-emerald-400"}
          />
        </button>

        {/* Server name */}
        <div className="text-center">
          {isConnected && activeServer && (
            <div className="text-lg font-medium">{activeServer.name}</div>
          )}
          {isConnected && fallbackMode && (
            <div className="flex items-center gap-1 text-yellow-400 text-sm mt-1">
              <AlertTriangle size={14} />
              <span>Fallback режим</span>
            </div>
          )}
          {isDisconnected && (
            <div className="text-[var(--text-muted)] text-sm">Выберите сервер для подключения</div>
          )}
        </div>

        {/* Speed indicators */}
        <div className="flex items-center gap-8">
          <div className="flex flex-col items-center gap-1">
            <div className="flex items-center gap-1.5 text-[var(--text-secondary)]">
              <ArrowDown size={16} className="text-emerald-400" />
              <span className="text-xs">Загрузка</span>
            </div>
            <span className="text-xl font-semibold text-emerald-400">
              {(speedDown / 1024 / 1024).toFixed(1)} MB/s
            </span>
          </div>
          <div className="w-px h-10 bg-[var(--border-color)]" />
          <div className="flex flex-col items-center gap-1">
            <div className="flex items-center gap-1.5 text-[var(--text-secondary)]">
              <ArrowUp size={16} className="text-purple-400" />
              <span className="text-xs">Отправка</span>
            </div>
            <span className="text-xl font-semibold text-purple-400">
              {(speedUp / 1024 / 1024).toFixed(1)} MB/s
            </span>
          </div>
        </div>
      </div>

      {/* Bottom stats panel */}
      <div className="border-t border-[var(--border-color)] bg-[var(--bg-secondary)]">
        {/* Speed graph */}
        {isConnected && (
          <div className="h-24 px-4 py-2">
            <SpeedGraph data={speedHistory} />
          </div>
        )}

        {/* Data usage stats */}
        <div className="flex items-center justify-around px-6 py-3 text-xs text-[var(--text-muted)]">
          <div className="flex items-center gap-1.5">
            <ArrowDown size={12} className="text-emerald-500" />
            <span>{formatBytes(status.stats.bytes_downloaded)}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <ArrowUp size={12} className="text-purple-500" />
            <span>{formatBytes(status.stats.bytes_uploaded)}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Clock size={12} />
            <span>{status.stats.duration_seconds ? formatDuration2(status.stats.duration_seconds) : "--:--:--"}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Zap size={12} />
            <span>SOCKS5 :{status.socks_port}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function formatDuration2(dur: string | number | null): string {
  if (!dur) return "00:00:00";
  const s = typeof dur === "string" ? parseInt(dur) : dur;
  if (isNaN(s)) return "00:00:00";
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  return `${h.toString().padStart(2, "0")}:${m.toString().padStart(2, "0")}:${sec.toString().padStart(2, "0")}`;
}