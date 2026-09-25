/**
 * Runtime endpoint resolution for facility_mfe.
 *
 * This remote is deployed as one image that must work in more than one
 * environment, so the API location cannot be a build-time constant. The
 * console shell publishes `window.__WAREHOUSE_CONFIG__` from a runtime
 * /config.json before any remote mounts; this module turns that origin into
 * this context's own API base.
 *
 * The fleet's localhost topology puts APIs on a DIFFERENT origin from the
 * frontend (Kong on :8000, Nginx on :80), so this is a real cross-origin URL
 * rather than a same-origin path -- Kong carries the matching CORS policy.
 *
 * In a production build a missing/malformed origin throws rather than falling
 * back to a developer port: a silent fallback would mean a deployed console
 * quietly talking to nothing.
 */
export interface WarehouseRuntimeConfig {
  apiOrigin?: string;
}

declare global {
  interface Window {
    __WAREHOUSE_CONFIG__?: WarehouseRuntimeConfig;
  }
}

const API_PATH = "/api/facility-layout";
const DEV_API_BASE = "http://localhost:8081";

export function resolveFacilityApiBase(
  runtimeConfig: WarehouseRuntimeConfig,
  isProduction: boolean,
): string {
  const apiOrigin = runtimeConfig.apiOrigin?.replace(/\/+$/, "");
  if (!apiOrigin) {
    if (isProduction) {
      throw new Error(
        "window.__WAREHOUSE_CONFIG__.apiOrigin is required in production",
      );
    }
    return DEV_API_BASE;
  }
  return `${apiOrigin}${API_PATH}`;
}

export const FACILITY_API_BASE = resolveFacilityApiBase(
  typeof window === "undefined" ? {} : (window.__WAREHOUSE_CONFIG__ ?? {}),
  import.meta.env.PROD,
);
