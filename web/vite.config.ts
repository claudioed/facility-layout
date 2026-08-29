import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { federation } from "@module-federation/vite";

// facility-mfe: the facility-layout remote. Exposes ./App -- the shell
// lazy-loads it at /facility/*. Also runnable standalone on :5186 for
// local development without the shell (see main.tsx).
export default defineConfig({
  plugins: [
    react(),
    federation({
      name: "facility_mfe",
      filename: "remoteEntry.js",
      exposes: {
        "./App": "./src/App.tsx",
      },
      shared: {
        react: { singleton: true, requiredVersion: "^19.2.8" },
        "react-dom": { singleton: true, requiredVersion: "^19.2.8" },
        "react-router-dom": { singleton: true, requiredVersion: "^7.18.3" },
        "@warehouse/ui-kit": { singleton: true },
      },
    }),
  ],
  server: {
    port: 5186,
    strictPort: true,
    cors: true,
    origin: "http://localhost:5186",
  },
  preview: {
    port: 5186,
    strictPort: true,
    cors: true,
  },
  build: {
    target: "esnext",
    modulePreload: false,
  },
});
