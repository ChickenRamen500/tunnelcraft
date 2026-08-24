import type { Server } from "@/hooks/useApi";
import { Wifi, Globe, Shield, Zap } from "lucide-react";

interface ServerCardProps {
  server: Server;
  isSelected?: boolean;
  isConnected?: boolean;
  onClick: (server: Server) => void;
}

const protocolIcons: Record<string, React.ReactNode> = {
  vless: <Zap size={14} className="text-purple-400" />,
  vmess: <Shield size={14} className="text-blue-400" />,
  wireguard: <Globe size={14} className="text-emerald-400" />,
  hysteria: <Wifi size={14} className="text-orange-400" />,
  hysteria2: <Wifi size={14} className="text-orange-400" />,
  amneziawg: <Globe size={14} className="text-yellow-400" />,
};

const protocolLabels: Record<string, string> = {
  vless: "VLESS",
  vmess: "VMESS",
  wireguard: "WireGuard",
  hysteria: "Hysteria",
  hysteria2: "Hysteria2",
  amneziawg: "AmneziaWG",
};

export default function ServerCard({ server, isSelected, isConnected, onClick }: ServerCardProps) {
  const highlight = isSelected || isConnected;
  return (
    <button
      onClick={() => onClick(server)}
      className={`w-full text-left px-4 py-3 rounded-lg transition-all duration-200 flex items-center gap-3 group
        ${highlight
          ? "bg-purple-500/10 border border-purple-500/30"
          : "bg-[var(--bg-card)] border border-[var(--border-color)] hover:bg-[var(--bg-hover)] hover:border-purple-500/20"
        }`}
    >
      <div className="w-8 h-8 rounded-md bg-[var(--bg-secondary)] flex items-center justify-center flex-shrink-0">
        {protocolIcons[server.protocol] || <Globe size={14} className="text-zinc-400" />}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate">{server.name}</span>
          {server.favorite && <span className="text-yellow-400 text-xs">★</span>}
        </div>
        <div className="flex items-center gap-2 text-xs text-[var(--text-muted)]">
          <span>{protocolLabels[server.protocol] || server.protocol}</span>
          <span>·</span>
          <span>{server.host}:{server.port}</span>
        </div>
      </div>
      {isConnected && (
        <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse flex-shrink-0" />
      )}
      {isSelected && !isConnected && (
        <span className="w-2 h-2 rounded-full bg-purple-400 flex-shrink-0" />
      )}
    </button>
  );
}
