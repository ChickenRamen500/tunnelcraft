import { useEffect, useState } from "react";
import { Plus, RefreshCw, Trash2, ExternalLink, CheckCircle, XCircle } from "lucide-react";
import { listSubscriptions, addSubscription, refreshSubscription, type Subscription } from "@/hooks/useApi";

export default function Subscriptions() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [newUrl, setNewUrl] = useState("");
  const [newName, setNewName] = useState("");

  const fetchSubs = async () => {
    try {
      const res = await listSubscriptions();
      setSubscriptions(res.subscriptions);
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchSubs();
  }, []);

  const handleRefresh = async (id: string) => {
    setLoading(true);
    try {
      await refreshSubscription(id);
      await fetchSubs();
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = async () => {
    if (!newUrl) return;
    try {
      await addSubscription(newName || "Подписка", newUrl);
      await fetchSubs();
      setShowAdd(false);
      setNewUrl("");
      setNewName("");
    } catch (e) {
      console.error("Failed to add subscription:", e);
      alert("Не удалось добавить подписку: " + (e instanceof Error ? e.message : e));
    }
  };

  return (
    <div className="flex flex-col h-full fade-in">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Подписки</h2>
          <button
            onClick={() => setShowAdd(!showAdd)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-500/10 border border-purple-500/30 text-purple-300 text-xs hover:bg-purple-500/20 transition-colors"
          >
            <Plus size={14} /> Добавить
          </button>
        </div>
      </div>

      {/* Add form */}
      {showAdd && (
        <div className="px-6 py-4 border-b border-[var(--border-color)] bg-[var(--bg-secondary)]">
          <div className="space-y-3">
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="Название подписки"
              className="w-full bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg px-3 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:border-purple-500/40"
            />
            <input
              value={newUrl}
              onChange={(e) => setNewUrl(e.target.value)}
              placeholder="https://example.com/subscribe?token=..."
              className="w-full bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg px-3 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:outline-none focus:border-purple-500/40"
            />
            <div className="flex gap-2">
              <button
                onClick={handleAdd}
                className="px-4 py-1.5 rounded-lg bg-purple-500 text-white text-xs hover:bg-purple-600 transition-colors"
              >
                Сохранить
              </button>
              <button
                onClick={() => setShowAdd(false)}
                className="px-4 py-1.5 rounded-lg bg-[var(--bg-card)] border border-[var(--border-color)] text-[var(--text-muted)] text-xs hover:text-[var(--text-primary)] transition-colors"
              >
                Отмена
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Subscription list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-2">
        {subscriptions.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)] gap-2">
            <ExternalLink size={32} />
            <p className="text-sm">Нет подписок</p>
            <p className="text-xs">Добавьте URL подписки для загрузки серверов</p>
          </div>
        ) : (
          subscriptions.map((sub) => (
            <div
              key={sub.id}
              className="bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg p-4 hover:bg-[var(--bg-hover)] transition-colors"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  {sub.enabled ? (
                    <CheckCircle size={16} className="text-emerald-400" />
                  ) : (
                    <XCircle size={16} className="text-zinc-500" />
                  )}
                  <span className="font-medium text-sm">{sub.name || "Без названия"}</span>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleRefresh(sub.id)}
                    className="p-1.5 rounded-md hover:bg-[var(--bg-secondary)] transition-colors"
                    title="Обновить"
                  >
                    <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
                  </button>
                  <button
                    className="p-1.5 rounded-md hover:bg-red-500/10 text-[var(--text-muted)] hover:text-red-400 transition-colors"
                    title="Удалить"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
              <div className="text-xs text-[var(--text-muted)] truncate">
                {sub.url}
              </div>
              <div className="flex items-center gap-3 mt-2 text-xs text-[var(--text-muted)]">
                <span>{sub.server_count} серверов</span>
                {sub.refresh_interval > 0 && <span>Обновление: каждые {sub.refresh_interval} мин</span>}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}