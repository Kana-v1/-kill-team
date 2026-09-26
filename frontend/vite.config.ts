import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The backend owns all rules logic; in dev we proxy /api to it so the browser
// can talk to a single origin. Override the target with KT_API_TARGET.
const apiTarget = process.env.KT_API_TARGET || "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: apiTarget, changeOrigin: true },
    },
  },
});
