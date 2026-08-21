import { create } from "zustand";
import type { ConnectionStatus, Server } from "@/hooks/useApi";

interface ConnectionState {
  status: ConnectionStatus;
  activeServer: Server | null;
  servers: Server[];
  speedUp: number;
  speedDown: number;
  speedHistory: { time: number; up: number; down: number }[];
  isConnecting: boolean;
  fallbackMode: boolean;
  // Actions
  setStatus: (status: ConnectionStatus) => void;
  setServers: (servers: Server[]) => void;
  setActiveServer: (server: Server | null) => void;
  setConnecting: (v: boolean) => void;
  setFallbackMode: (v: boolean) => void;
  updateSpeed: (up: number, down: number) => void;
}

export const useConnectionStore = create<ConnectionState>((set) => ({
  status: {
    state: "DISCONNECTED",
    server_id: null,
    mode: "SYSTEM",
    socks_port: 1080,
    http_port: 8080,
    stats: { bytes_uploaded: 0, bytes_downloaded: 0, duration: null },
  },
  activeServer: null,
  servers: [],
  speedUp: 0,
  speedDown: 0,
  speedHistory: [],
  isConnecting: false,
  fallbackMode: false,

  setStatus: (status) =>
    set((state) => ({
      status,
      isConnecting: status.state === "CONNECTING",
    })),

  setServers: (servers) => set({ servers }),
  setActiveServer: (server) => set({ activeServer: server }),
  setConnecting: (v) => set({ isConnecting: v }),
  setFallbackMode: (v) => set({ fallbackMode: v }),

  updateSpeed: (up, down) =>
    set((state) => ({
      speedUp: up,
      speedDown: down,
      speedHistory: [
        ...state.speedHistory.slice(-59),
        { time: Date.now(), up, down },
      ],
    })),
}));
