/// <reference types="vitest" />

import path from "node:path";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  // Fast Refresh's runtime preamble is only injected by Vite's dev-server
  // HTML transform, so it must stay off under Vitest (mode: 'test') or
  // @vitejs/plugin-react throws "can't detect preamble" on first render.
  plugins: [react({ fastRefresh: mode !== "test" })],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
  },
}));
