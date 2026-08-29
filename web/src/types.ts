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
