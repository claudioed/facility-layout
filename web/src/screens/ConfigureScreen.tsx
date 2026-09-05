import { useState } from "react";
import { Tabs } from "../components/formkit";
import { SitesPanel } from "./configure/SitesPanel";
import { ZonesPanel } from "./configure/ZonesPanel";
import { AislesPanel } from "./configure/AislesPanel";
import { LocationTypesPanel } from "./configure/LocationTypesPanel";
import { PlacementRulesPanel } from "./configure/PlacementRulesPanel";

const TABS = [
  { id: "sites", label: "Sites" },
  { id: "zones", label: "Zones" },
  { id: "aisles", label: "Aisles" },
  { id: "types", label: "Location types" },
  { id: "rules", label: "Placement rules" },
];

/**
 * Facility structural configuration: the write side of facility-layout's
 * hierarchy (Site -> Zone -> Aisle -> LocationType -> PlacementRule). Each
 * tab is a focused Card + DataTable pair (register form on top, live list
 * below) hitting exactly the use case CLAUDE.md names for it. Deliberately
 * ordered coarsest-to-finest, matching the domain's own scoping chain --
 * you cannot usefully register a Zone before a Site exists, an Aisle
 * before a Zone, etc.
 *
 * This is the "configure the warehouse" half of the facility remote; the
 * companion Layout Designer screen is the "draw the warehouse" half, where
 * individual LocationSlots get placed onto a real 2D canvas.
 */
export function ConfigureScreen() {
  const [active, setActive] = useState("sites");

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "var(--wh-space-5)" }}>
      <div>
        <h1 style={{ fontSize: "var(--wh-font-size-2xl)", margin: 0 }}>Configure facility</h1>
        <p style={{ color: "var(--wh-color-text-muted)", marginTop: 4 }}>
          facility-layout · register sites, zones, aisles, location types &amp; placement rules
        </p>
      </div>

      <Tabs tabs={TABS} active={active} onChange={setActive} />

      {active === "sites" && <SitesPanel />}
      {active === "zones" && <ZonesPanel />}
      {active === "aisles" && <AislesPanel />}
      {active === "types" && <LocationTypesPanel />}
      {active === "rules" && <PlacementRulesPanel />}
    </div>
  );
}
