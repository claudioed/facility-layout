import { defineConfig, mergeConfig } from "vitest/config";
import viteConfig from "./vite.config.ts";

// Reuses the app's own Vite config (React plugin, etc.) rather than
// duplicating it -- Module Federation's plugin is harmless under vitest
// (it just never triggers a remote fetch in a test run) so no override
// is needed to strip it out.
export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: "jsdom",
      globals: true,
      setupFiles: ["./src/test/setup.ts"],
      css: false,
    },
  }),
);
