import { useEffect, useState, useMemo } from "react";
import { Search, Star, RefreshCw, Filter } from "lucide-react";
import { useConnectionStore } from "@/stores/connection";
import { listServers, connectServer, type Server } from "@/hooks/useGrpc";
import ServerCard from "@/components/ServerCard";

export default function Servers() {
  const { servers, setServers, status, setActiveServer } = useConnectionStore();
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<string>("all");
  const [loading, setLoading] = useState(false);

  const fetchServers = async () => {
    setLoading(true);
    try {
      const res = await listServers();
      setServers(res.servers);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServers();
  }, []);

  const handleConnect = async (server: Server) => {
    try {
      setActiveServer(server);
      await connectServer(server.id);
    } catch (e) {
      console.error(e);
    }
  };

  // Group servers by protocol
  const grouped = useMemo(() => {
    let filtered = servers;
    if (search) {
      const q = search.toLowerCase();
      filtered = filtered.filter(
        (s) =>
          s.name.toLowerCase().includes(q) ||
          s.host.toLowerCase().includes(q) ||
          s.protocol.toLowerCase().includes(q)
      );
    }
    if (filter !== "all") {
      filtered = filtered.filter((s) => s.protocol === filter);
    }
    if (filter === "favorites") {
      filtered = servers.filter((s) => s.favorite);
    }

    const groups: Record<string, Server[]> = {};
    for (const s of filtered) {
      const label = s.protocol.toUpperCase();
      if (!groups[label]) groups[label] = [];
      groups[label].push(s);
    }
    return groups;
  }, [servers, search, filter]);

  const protocols = Array.from(new Set(servers.map((s) => s.protocol)));

  return (
    <div className="flex flex-col h-full fade-in">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold">Серверы</h2>
          <button
            onClick={fetchServers}
            className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
          >
            <RefreshCw size={13} className={loading ? "animate-spin" : ""} />
            Обновить
          </button>
        </div>

        {/* Search */}
        <div className="relative mb-3">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Поиск серверов..."
            className="w-full bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg pl-9 pr-3 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:border-purple-500/40"
          />
        </div>

        {/* Filters */}
        <div className="flex items-center gap-2 flex-wrap">
          <button
            onClick={() => setFilter("all")}
            className={`px-3 py-1 rounded-md text-xs transition-colors ${
              filter === "all" ? "bg-purple-500/20 text-purple-300 border border-purple-500/30" : "bg-[var(--bg-card)] border border-[var(--border-color)] text-[var(--text-muted)] hover:text-[var(--text-primary)]"
            }`}
          >
            Все ({servers.length})
          </button>
          <button
            onClick={() => setFilter("favorites")}
            className={`flex items-center gap-1 px-3 py-1 rounded-md text-xs transition-colors ${
              filter === "favorites" ? "bg-yellow-500/20 text-yellow-300 border border-yellow-500/30" : "bg-[var(--bg-card)] border border-[var(--border-color)] text-[var(--text-muted)] hover:text-[var(--text-primary)]"
            }`}
          >
            <Star size={11} /> Избранные
          </button>
          {protocols.map((p) => (
            <button
              key={p}
              onClick={() => setFilter(p)}
              className={`px-3 py-1 rounded-md text-xs transition-colors ${
                filter === p ? "bg-purple-500/20 text-purple-300 border border-purple-500/30" : "bg-[var(--bg-card)] border border-[var(--border-color)] text-[var(--text-muted)] hover:text-[var(--text-primary)]"
              }`}
            >
              {p.toUpperCase()}
            </button>
          ))}
        </div>
      </div>

      {/* Server list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-1.5">
        {Object.keys(grouped).length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)] gap-2">
            <Filter size={32} />
            <p className="text-sm">Серверы не найдены</p>
            <p className="text-xs">Добавьте подписку или сервер вручную</p>
          </div>
        ) : (
          Object.entries(grouped).map(([protocol, items]) => (
            <div key={protocol}>
              <div className="text-xs font-medium text-[var(--text-muted)] px-2 py-1.5 uppercase tracking-wider">
                {protocol} ({items.length})
              </div>
              <div className="space-y-1">
                {items.map((server) => (
                  <ServerCard
                    key={server.id}
                    server={server}
                    isActive={status.server_id === server.id}
                    onClick={handleConnect}
                  />
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}