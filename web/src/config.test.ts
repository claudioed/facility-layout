import { describe, expect, it } from "vitest";
import { resolveFacilityApiBase } from "./config";

describe("resolveFacilityApiBase", () => {
  it("builds the production API base from the runtime API origin", () => {
    expect(resolveFacilityApiBase({ apiOrigin: "http://localhost:8000" }, true)).toBe(
      "http://localhost:8000/api/facility-layout",
    );
  });

  it("normalizes a trailing slash on the runtime API origin", () => {
    expect(resolveFacilityApiBase({ apiOrigin: "https://warehouse.example/" }, true)).toBe(
      "https://warehouse.example/api/facility-layout",
    );
  });

  it("fails loudly when production runtime configuration has no API origin", () => {
    expect(() => resolveFacilityApiBase({}, true)).toThrow(
      "window.__WAREHOUSE_CONFIG__.apiOrigin is required in production",
    );
  });

  it("retains the existing standalone API origin in Vite development", () => {
    expect(resolveFacilityApiBase({}, false)).toBe("http://localhost:8081");
  });
});
