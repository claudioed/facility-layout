/** Local-dev base URL for facility-layout's own REST API. Mirrors
 *  e2e-tests/env.sh's FACILITY_HTTP_PORT=8081. See warehouse-console's
 *  src/config.ts for the note on swapping to runtime config before
 *  multi-environment deployment. */
export const FACILITY_API_BASE = "http://localhost:8081";
