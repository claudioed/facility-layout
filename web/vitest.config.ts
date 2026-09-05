import { defineConfig, mergeConfig } from "vitest/config";
import { fileURLToPath } from "node:url";
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
      alias: {
        // react-konva/konva render to a real <canvas> 2D context that
        // jsdom does not implement. Substitute a lightweight DOM-based
        // test double (see src/test/mocks/react-konva.tsx) so
        // ZoneCanvas/RackPlanCanvas's own click-to-coordinate logic is
        // testable without a native canvas dependency.
        "react-konva": fileURLToPath(
          new URL("./src/test/mocks/react-konva.tsx", import.meta.url),
        ),
      },
    },
  }),
);
