interface StatusBadgeProps {
  state: string;
  fallbackMode?: boolean;
}

const stateConfig: Record<string, { label: string; color: string; dot: string }> = {
  DISCONNECTED: { label: "Отключено", color: "text-zinc-400", dot: "bg-zinc-500" },
  CONNECTING: { label: "Подключение...", color: "text-yellow-400", dot: "bg-yellow-400 animate-pulse" },
  CONNECTED: { label: "Подключено", color: "text-emerald-400", dot: "bg-emerald-400" },
  RECONNECTING: { label: "Переподключение", color: "text-yellow-400", dot: "bg-yellow-400 animate-pulse" },
  DISCONNECTING: { label: "Отключение...", color: "text-zinc-400", dot: "bg-zinc-400 animate-pulse" },
  ERROR: { label: "Ошибка", color: "text-red-400", dot: "bg-red-400" },
  FALLBACK: { label: "Fallback (WG)", color: "text-yellow-400", dot: "bg-yellow-400" },
};

export default function StatusBadge({ state, fallbackMode }: StatusBadgeProps) {
  const key = fallbackMode ? "FALLBACK" : state;
  const config = stateConfig[key] || stateConfig.DISCONNECTED;

  return (
    <div className={`flex items-center gap-2 ${config.color}`}>
      <span className={`w-2.5 h-2.5 rounded-full ${config.dot}`} />
      <span className="text-sm font-medium">{config.label}</span>
    </div>
  );
}