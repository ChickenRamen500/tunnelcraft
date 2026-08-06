import { useEffect, useState } from "react";
import { Plus, Trash2, Shield, Globe, Ban, ArrowRight } from "lucide-react";
import { getRoutingRules, type RoutingRule } from "@/hooks/useGrpc";

export default function SplitTunnel() {
  const [rules, setRules] = useState<RoutingRule[]>([]);
  const [domainStrategy, setDomainStrategy] = useState("IPIfNonMatch");
  const [showAdd, setShowAdd] = useState(false);
  const [newDomain, setNewDomain] = useState("");
  const [newAction, setNewAction] = useState("proxy");

  const fetchRules = async () => {
    try {
      const res = await getRoutingRules();
      setRules(res.rules || []);
      setDomainStrategy(res.domain_strategy || "IPIfNonMatch");
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const actionIcon = (action: string) => {
    switch (action) {
      case "proxy": return <Shield size={14} className="text-purple-400" />;
      case "direct": return <Globe size={14} className="text-emerald-400" />;
      case "block": return <Ban size={14} className="text-red-400" />;
      default: return <ArrowRight size={14} className="text-zinc-400" />;
    }
  };

  const actionLabel = (action: string) => {
    switch (action) {
      case "proxy": return "Через VPN";
      case "direct": return "Напрямую";
      case "block": return "Заблокировать";
      default: return action;
    }
  };

  const handleAdd = () => {
    if (!newDomain) return;
    const rule: RoutingRule = {
      id: crypto.randomUUID(),
      name: newDomain,
      enabled: true,
      action: newAction,
      domains: [newDomain],
      ip_cidrs: [],
      geoip_codes: [],
      processes: [],
    };
    setRules([...rules, rule]);
    setNewDomain("");
    setShowAdd(false);
    // TODO: call gRPC CreateRule
  };

  const handleDelete = (id: string) => {
    setRules(rules.filter((r) => r.id !== id));
    // TODO: call gRPC DeleteRule
  };

  const handleToggle = (id: string) => {
    setRules(rules.map((r) => r.id === id ? { ...r, enabled: !r.enabled } : r));
    // TODO: call gRPC UpdateRule
  };

  return (
    <div className="flex flex-col h-full fade-in">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">Split Tunneling</h2>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">
              Управление правилами маршрутизации трафика
            </p>
          </div>
          <button
            onClick={() => setShowAdd(!showAdd)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-500/10 border border-purple-500/30 text-purple-300 text-xs hover:bg-purple-500/20 transition-colors"
          >
            <Plus size={14} /> Правило
          </button>
        </div>
      </div>

      {/* Add form */}
      {showAdd && (
        <div className="px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]">
          <div className="flex gap-3">
            <input
              value={newDomain}
              onChange={(e) => setNewDomain(e.target.value)}
              placeholder="Домен или IP (например: *.ru, 10.0.0.0/8)"
              className="flex-1 bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg px-3 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:border-purple-500/40"
            />
            <select
              value={newAction}
              onChange={(e) => setNewAction(e.target.value)}
              className="bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-purple-500/40"
            >
              <option value="proxy">Через VPN</option>
              <option value="direct">Напрямую</option>
              <option value="block">Заблокировать</option>
            </select>
            <button
              onClick={handleAdd}
              className="px-4 py-2 rounded-lg bg-purple-500 text-white text-sm hover:bg-purple-600 transition-colors"
            >
              Добавить
            </button>
          </div>
        </div>
      )}

      {/* Rules list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-1.5">
        {rules.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)] gap-2">
            <Shield size={32} />
            <p className="text-sm">Нет правил маршрутизации</p>
            <p className="text-xs">Весь трафик будет идти через VPN</p>
          </div>
        ) : (
          rules.map((rule) => (
            <div
              key={rule.id}
              className={`flex items-center gap-3 px-4 py-3 rounded-lg bg-[var(--bg-card)] border transition-colors ${
                rule.enabled ? "border-[var(--border-color)]" : "border-[var(--border-color)] opacity-50"
              }`}
            >
              <button
                onClick={() => handleToggle(rule.id)}
                className={`w-4 h-4 rounded border-2 flex items-center justify-center transition-colors ${
                  rule.enabled ? "bg-purple-500 border-purple-500" : "border-[var(--border-color)]"
                }`}
              >
                {rule.enabled && <span className="text-white text-xs">✓</span>}
              </button>
              {actionIcon(rule.action)}
              <div className="flex-1 min-w-0">
                <div className="text-sm truncate">{rule.name}</div>
                <div className="text-xs text-[var(--text-muted)]">{actionLabel(rule.action)}</div>
              </div>
              <button
                onClick={() => handleDelete(rule.id)}
                className="p-1.5 rounded-md hover:bg-red-500/10 text-[var(--text-muted)] hover:text-red-400 transition-colors"
              >
                <Trash2 size={13} />
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}