import { useState, type FormEvent } from "react";
import { Card, StatusPill, useFetch } from "@warehouse/ui-kit";
import { ZoneCanvas } from "../components/ZoneCanvas";
import {
  FormRow,
  InlineError,
  InlineSuccess,
  SelectField,
  SubmitButton,
  TextField,
} from "../components/formkit";
import { apiPost, ApiError } from "../api";
import { FACILITY_API_BASE } from "../config";
import type { LocationType, Site, Zone, ZoneGrid } from "../types";

interface PendingCell {
  columnIndex: number;
  rowIndex: number;
  aisleCode: string;
  bay: string;
  level: string;
}

/**
 * The "draw the warehouse" screen: pick a Site, pick one of its Zones, and
 * see + build that Zone's grid as an actual 2D canvas (Konva), not a text
 * table. Clicking any cell -- filled or an empty gap -- opens a form
 * scoped to that exact (Aisle, Bay, Level) coordinate; only Position and
 * LocationType are still free choices, so registering a slot is "click
 * where it goes, name it, submit" rather than typing a 7-segment
 * LocationCode by hand.
 *
 * GET /zones/{zoneId}/grid is re-fetched after every successful
 * registration so the canvas always reflects what the backend actually
 * has (POST /locations enforces the full chain-of-custody + PlacementRule
 * checks per CLAUDE.md -- this screen never assumes success client-side).
 */
export function LayoutDesignerScreen() {
  const [siteCode, setSiteCode] = useState("");
  const [zoneId, setZoneId] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [pending, setPending] = useState<PendingCell | null>(null);
  const [position, setPosition] = useState("");
  const [locationType, setLocationType] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const { data: sites } = useFetch<Site[]>(`${FACILITY_API_BASE}/sites`);
  const { data: zones } = useFetch<Zone[]>(
    siteCode ? `${FACILITY_API_BASE}/sites/${encodeURIComponent(siteCode)}/zones` : null,
  );
  const { data: types } = useFetch<LocationType[]>(`${FACILITY_API_BASE}/location-types`);
  const { data: grid, loading: gridLoading, error: gridError } = useFetch<ZoneGrid>(
    zoneId ? `${FACILITY_API_BASE}/zones/${encodeURIComponent(zoneId)}/grid?_r=${refreshKey}` : null,
  );

  function onCellClick(args: { column: ZoneGrid["columns"][number]; level: string }, columnIndex: number, rowIndex: number) {
    setError(null);
    setSuccess(null);
    setPosition("");
    setPending({
      columnIndex,
      rowIndex,
      aisleCode: args.column.aisleCode,
      bay: args.column.bay,
      level: args.level,
    });
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!pending || !grid) return;
    setError(null);
    setSuccess(null);
    setSubmitting(true);
    const locationCode = [
      grid.zone.siteCode,
      grid.zone.areaCode,
      grid.zone.zoneCode,
      pending.aisleCode,
      pending.bay,
      pending.level,
      position.trim().toUpperCase(),
    ].join("-");
    try {
      await apiPost("/locations", { locationCode, locationType });
      setSuccess(`Slot ${locationCode} registered.`);
      setPending(null);
      setPosition("");
      setLocationType("");
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to register slot.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-5)" }}>
      <div>
        <h1 style={{ fontSize: "var(--wh-font-size-2xl)", margin: 0 }}>Layout designer</h1>
        <p style={{ color: "var(--wh-color-text-muted)", marginTop: 4 }}>
          facility-layout · draw a zone's grid and place coded storage slots directly on the canvas
        </p>
      </div>

      <Card title="Choose a zone to draw">
        <FormRow>
          <SelectField
            label="Site"
            value={siteCode}
            onChange={(v) => {
              setSiteCode(v);
              setZoneId("");
              setPending(null);
            }}
            options={(sites ?? []).map((s) => ({ value: s.siteCode, label: `${s.siteCode} — ${s.name}` }))}
            required
          />
          <SelectField
            label="Zone"
            value={zoneId}
            onChange={(v) => {
              setZoneId(v);
              setPending(null);
            }}
            options={(zones ?? []).map((z) => ({ value: z.zoneId, label: `${z.zoneId} (${z.temperatureClass})` }))}
            required
          />
        </FormRow>
      </Card>

      {gridError && (
        <Card>
          <InlineError message={gridError.message} />
        </Card>
      )}

      {zoneId && (
        <Card
          title={grid ? `Grid — ${grid.zone.zoneId}` : "Grid"}
          actions={
            grid && (
              <div style={{ display: "flex", gap: "var(--wh-space-2)" }}>
                <StatusPill status={grid.zone.temperatureClass} size="sm" />
                {grid.zone.hazmat && <StatusPill status="Hazmat" tone="danger" size="sm" />}
              </div>
            )
          }
        >
          {gridLoading && (
            <div style={{ color: "var(--wh-color-text-muted)", fontSize: "var(--wh-font-size-sm)" }}>
              Loading grid…
            </div>
          )}
          {grid && !gridLoading && grid.columns.length === 0 && (
            <div style={{ color: "var(--wh-color-text-muted)", fontSize: "var(--wh-font-size-sm)" }}>
              No aisles registered for this zone yet — register one under Configure → Aisles first.
            </div>
          )}
          {grid && !gridLoading && grid.columns.length > 0 && (
            <ZoneCanvas
              grid={grid}
              selected={pending ? { columnIndex: pending.columnIndex, rowIndex: pending.rowIndex } : null}
              onCellClick={(args) => {
                const columnIndex = grid.columns.findIndex((c) => c === args.column);
                const rowIndex = grid.rows.findIndex((r) => r.level === args.level);
                onCellClick(args, columnIndex, rowIndex);
              }}
            />
          )}
        </Card>
      )}

      {pending && grid && (
        <Card
          title={`Register a slot — Aisle ${pending.aisleCode} · Bay ${pending.bay} · Level ${pending.level}`}
          actions={
            <button
              type="button"
              onClick={() => setPending(null)}
              style={{
                padding: "6px 12px",
                borderRadius: "var(--wh-radius-md)",
                border: "1px solid var(--wh-color-border)",
                background: "transparent",
                color: "var(--wh-color-text-muted)",
                fontSize: "var(--wh-font-size-xs)",
                cursor: "pointer",
              }}
            >
              Cancel
            </button>
          }
        >
          <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-3)" }}>
            <FormRow>
              <TextField label="Position" value={position} onChange={setPosition} placeholder="B" required />
              <SelectField
                label="Location type"
                value={locationType}
                onChange={setLocationType}
                options={(types ?? []).map((t) => ({ value: t.name, label: t.name }))}
                required
              />
              <SubmitButton disabled={submitting || !position.trim() || !locationType}>
                {submitting ? "Registering…" : "Register slot"}
              </SubmitButton>
            </FormRow>
            <p style={{ margin: 0, fontSize: "var(--wh-font-size-xs)", color: "var(--wh-color-text-faint)", fontFamily: "var(--wh-font-mono)" }}>
              Location code: {grid.zone.siteCode}-{grid.zone.areaCode}-{grid.zone.zoneCode}-{pending.aisleCode}-{pending.bay}-{pending.level}-{position.trim().toUpperCase() || "?"}
            </p>
            <InlineError message={error} />
            <InlineSuccess message={success} />
          </form>
        </Card>
      )}
    </div>
  );
}
