import { useEffect, useState } from "react";
import { FileText, Trash2, Download, AlertCircle } from "lucide-react";
import { getLogs } from "@/hooks/useApi";

export default function Logs() {
  const [logs, setLogs] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  const fetchLogs = async () => {
    try {
      const res = await getLogs(500);
      setLogs(res.logs || []);
      setError(null);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  // Poll logs every 2 seconds
  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 2000);
    return () => clearInterval(interval);
  }, []);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll) {
      const container = document.getElementById("logs-container");
      if (container) {
        container.scrollTop = container.scrollHeight;
      }
    }
  }, [logs, autoScroll]);

  const handleClear = () => {
    setLogs([]);
  };

  const handleExport = () => {
    const blob = new Blob([logs.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `tunnelcraft-logs-${new Date().toISOString().slice(0, 19)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex flex-col h-full fade-in">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileText size={18} className="text-purple-400" />
            <h2 className="text-lg font-semibold">Логи демона</h2>
          </div>
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] cursor-pointer">
              <input
                type="checkbox"
                checked={autoScroll}
                onChange={(e) => setAutoScroll(e.target.checked)}
                className="rounded border-[var(--border-color)] bg-[var(--bg-card)]"
              />
              Автопрокрутка
            </label>
            <button
              onClick={handleClear}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[var(--bg-card)] border border-[var(--border-color)] text-[var(--text-muted)] text-xs hover:text-red-400 hover:border-red-500/30 transition-colors"
            >
              <Trash2 size={14} /> Очистить
            </button>
            <button
              onClick={handleExport}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-500/10 border border-purple-500/30 text-purple-300 text-xs hover:bg-purple-500/20 transition-colors"
            >
              <Download size={14} /> Экспорт
            </button>
          </div>
        </div>
      </div>

      {/* Logs content */}
      <div className="flex-1 overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center h-full text-[var(--text-muted)]">
            Загрузка логов...
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)] gap-2">
            <AlertCircle size={32} className="text-red-400" />
            <span>Ошибка загрузки логов: {error}</span>
            <button onClick={fetchLogs} className="px-4 py-2 rounded-lg bg-purple-500 text-white text-sm">
              Повторить
            </button>
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-[var(--text-muted)] gap-2">
            <FileText size={32} />
            <p className="text-sm">Нет логов</p>
            <p className="text-xs">Логи появятся после запуска операций</p>
          </div>
        ) : (
          <div
            id="logs-container"
            className="h-full overflow-y-auto p-4 bg-[var(--bg-secondary)] font-mono text-xs text-[var(--text-primary)]"
          >
            {logs.map((log, i) => (
              <div key={i} className="py-0.5 whitespace-pre-wrap break-all border-b border-[var(--border-color)]/30 last:border-0">
                {log}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Footer stats */}
      {!loading && !error && (
        <div className="px-6 py-2 border-t border-[var(--border-color)] text-xs text-[var(--text-muted)] flex items-center justify-between">
          <span>{logs.length} записей</span>
          <span>Обновлено: {new Date().toLocaleTimeString()}</span>
        </div>
      )}
    </div>
  );
}
