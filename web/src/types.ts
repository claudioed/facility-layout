/** Wire types mirroring facility-layout's dto.go response shapes exactly
 *  (siteResponse / zoneResponse / aisleResponse / locationSlotResponse /
 *  siteLayoutResponse / locationClassificationResponse) -- kept hand-in-
 *  sync with the Go DTOs rather than code-generated for v1, same
 *  convention as order-mgmt-mfe's types.ts. */

export interface Site {
  siteCode: string;
  name: string;
  status: string;
}

export interface Zone {
  zoneId: string;
  siteCode: string;
  areaCode: string;
  zoneCode: string;
  temperatureClass: string;
  hazmat: boolean;
  status: string;
}

export interface Aisle {
  aisleId: string;
  zoneId: string;
  aisleCode: string;
  sequenceHint: number;
  direction: string;
  status: string;
}

export interface Capacity {
  maxWeightKg: number;
  maxVolumeM3: number;
}

export interface LocationCoordinates {
  site: string;
  area: string;
  zone: string;
  aisle: string;
  bay: string;
  level: string;
  position: string;
}

export interface LocationSlot {
  locationCode: string;
  zoneId: string;
  aisleId: string;
  coordinates: LocationCoordinates;
  locationType: string;
  capacity: Capacity;
  status: string;
}

/** locationClassificationResponse -- the denormalized Hazmat/TemperatureClass
 *  read a downstream consumer (inventory-storage) resolves from a
 *  LocationCode's parent Zone. Not rendered by this screen yet, but kept
 *  here since it's part of this service's published DTO shape. */
export interface LocationClassification {
  hazmat: boolean;
  temperatureClass: string;
}

// ------------------------------------------------------ configuration -----

export interface Capacity {
  maxWeightKg: number;
  maxVolumeM3: number;
}

/** locationTypeResponse. */
export interface LocationType {
  name: string;
  defaultCapacity: Capacity;
}

/** zonePredicateResponse -- a PlacementRule's target: a specific zone code
 *  and/or a temperature class and/or a hazmat flag. All fields optional;
 *  any combination present must ALL match for the rule to apply. */
export interface ZonePredicate {
  zoneCode?: string;
  temperatureClass?: string;
  hazmat?: boolean;
}

/** placementRuleResponse. */
export interface PlacementRule {
  ruleId: string;
  locationType: string;
  effect: "Allow" | "Deny";
  zone: ZonePredicate;
  description: string;
}

// --------------------------------------------- "draw the warehouse" -----

/** aisleLayoutResponse: aisleResponse embedded + Slots. */
export interface AisleLayout extends Aisle {
  slots: LocationSlot[];
}

/** zoneLayoutResponse: zoneResponse embedded + Aisles. */
export interface ZoneLayout extends Zone {
  aisles: AisleLayout[];
}

export interface SiteLayoutTotals {
  zones: number;
  aisles: number;
  slots: number;
}

/** siteLayoutResponse -- the full nested, drawable structure of one site:
 *  zones -> aisles -> slots, as returned by GET /sites/{siteCode}/layout. */
export interface SiteLayout {
  site: Site;
  zones: ZoneLayout[];
  totals: SiteLayoutTotals;
}

// --------------------------------------------------------- zone grid -----

/** zoneGridResponse: one zone's slots as an explicit matrix -- rows are
 *  Levels, columns are (Aisle, Bay) pairs in aisle walk order, and a cell
 *  is null where the rack has a gap. Returned by GET /zones/{zoneId}/grid. */
export interface GridColumn {
  aisleId: string;
  aisleCode: string;
  bay: string;
  sequenceHint: number;
}

export interface GridPosition {
  locationCode: string;
  position: string;
  locationType: string;
  status: string;
}

export interface GridCell {
  positions: GridPosition[];
}

export interface GridRow {
  level: string;
  /** Index-aligned with ZoneGrid.columns. Null entry = a gap in the rack. */
  cells: (GridCell | null)[];
}

export interface ZoneGrid {
  zone: Zone;
  columns: GridColumn[];
  levels: string[];
  rows: GridRow[];
}

// ------------------------------------------------------- bulk import -----

/** importRowRequest -- one fully-specified row of a facility layout
 *  import: a whole Site/Area/Zone/Aisle/Bay/Level/Position address plus
 *  the LocationType for the slot at it. Structural parents named by a row
 *  are created on first sight (POST /locations/import), which is what
 *  lets the Rack Planner deploy a brand-new Site/Zone/Aisle and all its
 *  slots in one call instead of requiring each to be pre-registered. */
export interface ImportRow {
  siteCode: string;
  siteName?: string;
  areaCode: string;
  zoneCode: string;
  temperatureClass: string;
  hazmat: boolean;
  aisleCode: string;
  sequenceHint: number;
  direction?: string;
  bay: string;
  level: string;
  position: string;
  locationType: string;
  capacityOverride?: Capacity;
}

export interface ImportRowResult {
  index: number;
  locationCode: string;
  succeeded: boolean;
  error?: string;
}

/** importReportResponse -- the full outcome of POST /locations/import:
 *  every row is processed, partial success is reported per row rather
 *  than aborting the whole import on the first bad row. */
export interface ImportReport {
  rowsSubmitted: number;
  slotsImported: number;
  rowsRejected: number;
  results: ImportRowResult[];
}
