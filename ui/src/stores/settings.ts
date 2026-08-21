import { create } from "zustand";
import { getSettings, saveSettings, type Settings } from "@/hooks/useApi";

const STORAGE_KEY = "tunnelcraft_settings";

// Load settings from localStorage or API
async function loadSettings(): Promise<Settings> {
  // Try localStorage first for instant UI response
  const cached = localStorage.getItem(STORAGE_KEY);
  if (cached) {
    try {
      const parsed = JSON.parse(cached);
      // Update daemon with cached language preference in background
      saveSettings({ language: parsed.language }).catch(console.error);
      return parsed;
    } catch (e) {
      console.error("Failed to parse cached settings:", e);
    }
  }

  // Fetch from daemon
  try {
    const settings = await getSettings();
    // Cache in localStorage
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
    return settings;
  } catch (e) {
    console.error("Failed to fetch settings from daemon:", e);
    // Return defaults
    return {
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
  }
}

interface SettingsState {
  settings: Settings | null;
  loading: boolean;
  error: string | null;
  // Actions
  loadSettings: () => Promise<void>;
  updateSettings: (settings: Partial<Settings>) => Promise<void>;
  setLanguage: (lang: string) => Promise<void>;
}

export const useSettingsStore = create<SettingsState>((set, get) => ({
  settings: null,
  loading: true,
  error: null,

  loadSettings: async () => {
    set({ loading: true, error: null });
    try {
      const settings = await loadSettings();
      set({ settings, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  updateSettings: async (partial: Partial<Settings>) => {
    const current = get().settings;
    if (!current) return;

    const updated = { ...current, ...partial };
    try {
      await saveSettings(partial);
      // Update both daemon and localStorage
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      set({ settings: updated });
    } catch (e) {
      console.error("Failed to save settings:", e);
      throw e;
    }
  },

  setLanguage: async (lang: string) => {
    await get().updateSettings({ language: lang });
  },
}));
