import type { LocationRole } from "./types";

/**
 * Per-role display metadata (ADR-0016): a short glyph and an accent color
 * for the role badge/filter chips in FacilityScreen. Storage is the
 * pre-ADR-0016 default and the overwhelming majority of real slots, so it
 * intentionally gets a muted, low-emphasis treatment rather than a loud
 * color -- the whole point of role-aware rendering is to make the
 * non-storage functional locations (dock doors, yard spots, work
 * centers...) POP against a sea of ordinary storage slots, not to color
 * every single chip in the warehouse.
 */
export const ROLE_ORDER: LocationRole[] = [
  "Storage",
  "Dock",
  "Yard",
  "WorkCenter",
  "Drop",
  "Staging",
  "QC",
  "Consolidation",
  "Shipping",
];

export interface RoleMeta {
  /** Short glyph shown on a slot chip's badge (kept to 1-2 chars so it
   *  doesn't dominate the compact chip). */
  glyph: string;
  /** Full label used in the filter chip row and tooltips. */
  label: string;
  fg: string;
  bg: string;
}

export const ROLE_META: Record<LocationRole, RoleMeta> = {
  Storage: { glyph: "S", label: "Storage", fg: "var(--wh-color-text-faint)", bg: "transparent" },
  Dock: { glyph: "D", label: "Dock", fg: "#4d8dff", bg: "#12233d" },
  Yard: { glyph: "Y", label: "Yard", fg: "#f5b942", bg: "#3a2c10" },
  WorkCenter: { glyph: "W", label: "Work Center", fg: "#3dd68c", bg: "#123729" },
  Drop: { glyph: "P", label: "Drop", fg: "#c792ea", bg: "#2a1f3a" },
  Staging: { glyph: "T", label: "Staging", fg: "#7dd3fc", bg: "#0f2a3a" },
  QC: { glyph: "Q", label: "QC", fg: "#f97066", bg: "#3a1414" },
  Consolidation: { glyph: "C", label: "Consolidation", fg: "#fbbf6b", bg: "#3a2a10" },
  Shipping: { glyph: "H", label: "Shipping", fg: "#60d9c6", bg: "#0f3a34" },
};

export function roleMeta(role: string): RoleMeta {
  return (ROLE_META as Record<string, RoleMeta>)[role] ?? ROLE_META.Storage;
}
