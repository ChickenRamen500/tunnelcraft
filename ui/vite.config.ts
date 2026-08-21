import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig(async () => ({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  clearScreen: false,
  server: {
    host: "127.0.0.1",
    port: 5173,
    strictPort: true,
    watch: {
      // Ignore the Cargo target/ directory to prevent EBUSY errors.
      // Cargo locks .dll files during compilation, which makes Vite's
      // file watcher crash with "resource busy or locked".
      ignored: ["**/src-tauri/target/**"],
    },
  },
}));
