import { useState } from "react";
import { LayoutDashboard, Server, Rss, GitBranch, Settings, Activity } from "lucide-react";
import { useConnectionStore } from "@/stores/connection";
import { getConnectionStatus } from "@/hooks/useGrpc";
import Dashboard from "@/pages/Dashboard";
import Servers from "@/pages/Servers";
import Subscriptions from "@/pages/Subscriptions";
import SplitTunnel from "@/pages/SplitTunnel";
import SettingsPage from "@/pages/Settings";
import StatusBadge from "@/components/StatusBadge";

const tabs = [
  { id: "dashboard", label: "Главная", icon: LayoutDashboard },
  { id: "servers", label: "Серверы", icon: Server },
  { id: "subscriptions", label: "Подписки", icon: Rss },
  { id: "split-tunnel", label: "Маршруты", icon: GitBranch },
  { id: "settings", label: "Настройки", icon: Settings },
] as const;

type TabId = (typeof tabs)[number]["id"];

export default function App() {
  const [activeTab, setActiveTab] = useState<TabId>("dashboard");
  const { status, fallbackMode, setStatus } = useConnectionStore();

  // Initial status fetch
  useState(() => {
    getConnectionStatus().then(setStatus).catch(console.error);
  });

  const renderPage = () => {
    switch (activeTab) {
      case "dashboard": return <Dashboard />;
      case "servers": return <Servers />;
      case "subscriptions": return <Subscriptions />;
      case "split-tunnel": return <SplitTunnel />;
      case "settings": return <SettingsPage />;
    }
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-[var(--bg-primary)] select-none">
      {/* Title bar (draggable area) */}
      <div data-tauri-drag-region className="h-10 flex items-center px-4 border-b border-[var(--border-color)] flex-shrink-0">
        <div className="flex items-center gap-2" data-tauri-drag-region>
          <Activity size={16} className="text-purple-400" />
          <span className="text-sm font-semibold">TunnelCraft</span>
        </div>
        <div className="ml-auto flex items-center gap-3">
          <StatusBadge state={status.state} fallbackMode={fallbackMode} />
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar */}
        <nav className="w-48 border-r border-[var(--border-color)] bg-[var(--bg-secondary)] flex flex-col py-2 flex-shrink-0">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2.5 px-4 py-2.5 mx-2 rounded-lg text-sm transition-all duration-150
                  ${isActive
                    ? "bg-purple-500/10 text-purple-300 border-l-2 border-purple-500"
                    : "text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)] border-l-2 border-transparent"
                  }`}
              >
                <Icon size={16} />
                <span>{tab.label}</span>
              </button>
            );
          })}

          {/* Spacer */}
          <div className="flex-1" />

          {/* Version */}
          <div className="px-4 py-2 text-xs text-[var(--text-muted)]">v0.1.0</div>
        </nav>

        {/* Main content */}
        <main className="flex-1 overflow-hidden">
          {renderPage()}
        </main>
      </div>

      {/* Footer */}
      <footer className="h-8 flex items-center justify-center border-t border-[var(--border-color)] text-xs text-[var(--text-muted)] flex-shrink-0">
        TunnelCraft © 2025 · Все права защищены
      </footer>
    </div>
  );
}
