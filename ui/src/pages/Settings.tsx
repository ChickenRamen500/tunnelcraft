import { useEffect, useState } from "react";
import { useConnectionStore } from "@/stores/connection";
import { getSettings, saveSettings, type Settings } from "@/hooks/useGrpc";
import { Shield, Globe, Zap, Monitor, Languages, Palette, RotateCcw } from "lucide-react";

export default function Settings() {
  const { status } = useConnectionStore();
  const [settings, setSettings] = useState<Settings | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    getSettings().then(setSettings).catch(console.error);
  }, []);

  const handleSave = async () => {
    try {
      await saveSettings(settings);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      console.error("Failed to save settings:", e);
      alert("Не удалось сохранить: " + (e instanceof Error ? e.message : e));
    }
  };

  if (!settings) {
    return <div className="flex items-center justify-center h-full text-[var(--text-muted)]">Загрузка...</div>;
  }

  return (
    <div className="flex flex-col h-full fade-in">
      {/* Header */}
      <div className="px-6 py-4 border-b border-[var(--border-color)]">
        <h2 className="text-lg font-semibold">Настройки</h2>
      </div>

      {/* Settings list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {/* Connection */}
        <SettingsGroup title="Подключение" icon={<Zap size={15} />}>
          <SettingsToggle
            label="Автоподключение"
            desc="Подключаться к последнему серверу при запуске"
            value={settings.auto_connect}
            onChange={(v) => setSettings({ ...settings, auto_connect: v })}
          />
          <SettingsToggle
            label="Подключение при старте системы"
            desc="Запускать VPN вместе с Windows"
            value={settings.connect_on_startup}
            onChange={(v) => setSettings({ ...settings, connect_on_startup: v })}
          />
          <SettingsToggle
            label="Kill Switch"
            desc="Блокировать весь трафик при разрыве VPN"
            value={settings.kill_switch}
            onChange={(v) => setSettings({ ...settings, kill_switch: v })}
          />
          <SettingsToggle
            label="Разрешить LAN"
            desc="Доступ к локальной сети при подключении"
            value={settings.allow_lan}
            onChange={(v) => setSettings({ ...settings, allow_lan: v })}
          />
        </SettingsGroup>

        {/* Proxy */}
        <SettingsGroup title="Прокси" icon={<Globe size={15} />}>
          <SettingsInput
            label="SOCKS5 порт"
            value={String(settings.socks_port)}
            onChange={(v) => setSettings({ ...settings, socks_port: parseInt(v) || 1080 })}
          />
          <SettingsInput
            label="HTTP порт"
            value={String(settings.http_port)}
            onChange={(v) => setSettings({ ...settings, http_port: parseInt(v) || 8080 })}
          />
          <SettingsInput
            label="DNS серверы"
            value={settings.dns_servers}
            onChange={(v) => setSettings({ ...settings, dns_servers: v })}
          />
        </SettingsGroup>

        {/* Security */}
        <SettingsGroup title="Безопасность" icon={<Shield size={15} />}>
          <SettingsInput
            label="Таймаут подключения (сек)"
            value={String(settings.connection_timeout)}
            onChange={(v) => setSettings({ ...settings, connection_timeout: parseInt(v) || 30 })}
          />
          <SettingsInput
            label="Попыток переподключения"
            value={String(settings.reconnect_attempts)}
            onChange={(v) => setSettings({ ...settings, reconnect_attempts: parseInt(v) || 3 })}
          />
        </SettingsGroup>

        {/* Appearance */}
        <SettingsGroup title="Внешний вид" icon={<Palette size={15} />}>
          <div className="flex items-center justify-between py-2">
            <span className="text-sm">Язык</span>
            <select
              value={settings.language}
              onChange={(e) => setSettings({ ...settings, language: e.target.value })}
              className="bg-[var(--bg-card)] border border-[var(--border-color)] rounded-md px-3 py-1.5 text-sm text-[var(--text-primary)] focus:outline-none"
            >
              <option value="ru">Русский</option>
              <option value="en">English</option>
            </select>
          </div>
        </SettingsGroup>

        {/* Info */}
        <SettingsGroup title="О программе" icon={<Monitor size={15} />}>
          <div className="py-2 text-xs text-[var(--text-muted)]">
            <p>TunnelCraft v0.1.0</p>
            <p className="mt-1">Версия демона: {status.mode}</p>
          </div>
        </SettingsGroup>
      </div>

      {/* Footer */}
      <div className="px-6 py-3 border-t border-[var(--border-color)] flex items-center justify-between">
        <button
          onClick={handleSave}
          className="px-6 py-2 rounded-lg bg-purple-500 text-white text-sm font-medium hover:bg-purple-600 transition-colors"
        >
          {saved ? "✓ Сохранено" : "Сохранить"}
        </button>
        <div className="flex items-center gap-1 text-xs text-[var(--text-muted)]">
          <RotateCcw size={12} />
          <span>Настройки применяются мгновенно</span>
        </div>
      </div>
    </div>
  );
}

function SettingsGroup({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="bg-[var(--bg-card)] border border-[var(--border-color)] rounded-lg p-4">
      <div className="flex items-center gap-2 mb-3 text-sm font-medium">
        <span className="text-purple-400">{icon}</span>
        {title}
      </div>
      <div className="divide-y divide-[var(--border-color)]">{children}</div>
    </div>
  );
}

function SettingsToggle({
  label,
  desc,
  value,
  onChange,
}: {
  label: string;
  desc?: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between py-3">
      <div>
        <div className="text-sm">{label}</div>
        {desc && <div className="text-xs text-[var(--text-muted)] mt-0.5">{desc}</div>}
      </div>
      <button
        onClick={() => onChange(!value)}
        className={`w-10 h-5 rounded-full transition-colors relative ${value ? "bg-purple-500" : "bg-zinc-700"}`}
      >
        <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${value ? "translate-x-5" : "translate-x-0.5"}`} />
      </button>
    </div>
  );
}

function SettingsInput({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div className="flex items-center justify-between py-3 gap-4">
      <span className="text-sm whitespace-nowrap">{label}</span>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="bg-[var(--bg-secondary)] border border-[var(--border-color)] rounded-md px-3 py-1.5 text-sm text-[var(--text-primary)] text-right w-48 focus:outline-none focus:border-purple-500/40"
      />
    </div>
  );
}