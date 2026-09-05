import { useState, type FormEvent } from "react";
import { Card, DataTable, useFetch } from "@warehouse/ui-kit";
import { apiPost, ApiError } from "../../api";
import { FACILITY_API_BASE } from "../../config";
import type { LocationType } from "../../types";
import { FormRow, InlineError, InlineSuccess, SubmitButton, TextField } from "../../components/formkit";

/** RegisterLocationType: POST /location-types -- a reusable classification
 *  of physical slot shape/kind (PalletRack, Shelf, ToteWall, BulkFloor,
 *  Staging, Amnesty per CLAUDE.md), carrying a default capacity envelope.
 *  Every LocationSlot and PlacementRule references one of these by name. */
export function LocationTypesPanel() {
  const [refreshKey, setRefreshKey] = useState(0);
  const [name, setName] = useState("");
  const [maxWeightKg, setMaxWeightKg] = useState("");
  const [maxVolumeM3, setMaxVolumeM3] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const { data: types, loading, error: listError } = useFetch<LocationType[]>(
    `${FACILITY_API_BASE}/location-types?_r=${refreshKey}`,
  );

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setSubmitting(true);
    try {
      await apiPost<LocationType>("/location-types", {
        name: name.trim(),
        defaultCapacity: {
          maxWeightKg: Number(maxWeightKg) || 0,
          maxVolumeM3: Number(maxVolumeM3) || 0,
        },
      });
      setSuccess(`Location type ${name.trim()} registered.`);
      setName("");
      setMaxWeightKg("");
      setMaxVolumeM3("");
      setRefreshKey((k) => k + 1);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to register location type.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-5)" }}>
      <Card title="Register a location type">
        <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-3)" }}>
          <FormRow>
            <TextField label="Name" value={name} onChange={setName} placeholder="PalletRack" required />
            <TextField
              label="Default max weight (kg)"
              type="number"
              value={maxWeightKg}
              onChange={setMaxWeightKg}
              placeholder="1200"
              required
            />
            <TextField
              label="Default max volume (m³)"
              type="number"
              value={maxVolumeM3}
              onChange={setMaxVolumeM3}
              placeholder="1.5"
              required
            />
            <SubmitButton disabled={submitting || !name.trim()}>
              {submitting ? "Registering…" : "Register type"}
            </SubmitButton>
          </FormRow>
          <InlineError message={error} />
          <InlineSuccess message={success} />
        </form>
      </Card>

      <Card title="Location types">
        {listError && <InlineError message={listError.message} />}
        <DataTable
          rowKey={(t) => t.name}
          rows={types ?? []}
          loading={loading}
          emptyState={<span>No location types registered yet.</span>}
          columns={[
            { key: "name", header: "Name", render: (t) => t.name },
            { key: "maxWeightKg", header: "Max weight (kg)", render: (t) => t.defaultCapacity.maxWeightKg, align: "right" },
            { key: "maxVolumeM3", header: "Max volume (m³)", render: (t) => t.defaultCapacity.maxVolumeM3, align: "right" },
          ]}
        />
      </Card>
    </div>
  );
}
