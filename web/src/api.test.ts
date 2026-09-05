import { describe, expect, it, vi, afterEach } from "vitest";
import { apiPost, ApiError } from "./api";

describe("apiPost", () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("returns the parsed JSON body on success", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({ siteCode: "WH1", name: "Test", status: "Active" }),
    }) as unknown as typeof fetch;

    const result = await apiPost("/sites", { siteCode: "WH1", name: "Test" });
    expect(result).toEqual({ siteCode: "WH1", name: "Test", status: "Active" });
  });

  it("returns undefined on a 204 No Content response", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
    }) as unknown as typeof fetch;

    const result = await apiPost("/locations/WH1-STOR-AMB-A07-03-02-B/decommission", {});
    expect(result).toBeUndefined();
  });

  it("throws ApiError with the parsed RFC 7807 problem detail on failure", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      statusText: "Conflict",
      json: async () => ({
        type: "https://errors.facility-layout.warehouse-systems.dev/duplicate-site-code",
        title: "A site with this code already exists",
        status: 409,
        detail: "a site with this code already exists",
      }),
    }) as unknown as typeof fetch;

    await expect(apiPost("/sites", { siteCode: "WH1", name: "Test" })).rejects.toThrow(
      ApiError,
    );
    await expect(apiPost("/sites", { siteCode: "WH1", name: "Test" })).rejects.toThrow(
      "a site with this code already exists",
    );
  });

  it("falls back to statusText when the error body is not JSON", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: async () => {
        throw new Error("not json");
      },
    }) as unknown as typeof fetch;

    await expect(apiPost("/sites", {})).rejects.toThrow("500 Internal Server Error");
  });
});
