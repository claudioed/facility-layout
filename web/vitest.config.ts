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
    resolve: {
      // @warehouse/ui-kit is consumed via `file:../../warehouse-ui-kit` and
      // has its own installed react/react-dom (declared as peerDependencies
      // but still present in its own node_modules). Module Federation's
      // `shared: { singleton: true }` config resolves this at build/runtime,
      // but vitest never goes through that MF resolution -- without
      // dedupe, ui-kit's useFetch calls React.useState against a SEPARATE
      // React module instance than the one rendering the test tree,
      // producing "Invalid hook call" / "Cannot read properties of null".
      dedupe: ["react", "react-dom"],
    },
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
