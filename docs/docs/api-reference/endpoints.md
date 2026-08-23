---
id: endpoints
title: Endpoint catalogue
sidebar_label: Endpoint catalogue
description: All 22 operations across 17 paths, grouped by OpenAPI tag and cross-checked against the chi router.
---

# Endpoint catalogue

Every route the chi router mounts, grouped by its OpenAPI tag. Click any
operation to reach the generated, interactive reference for it.

:::tip[Coverage]
**17 / 17** router paths have a corresponding `paths` entry in
`apis/openapi.yaml`, and **22 / 22** operations are documented. The
cross-check is at the [bottom of this page](#coverage-cross-check).
:::

## Sites

Physical facilities/buildings. A Site is the root of the location hierarchy
and the first segment of every LocationCode inside it; no Zone, Aisle or
LocationSlot can exist without one.

| Method | Path | Operation | Success |
|---|---|---|---|
| `POST` | `/sites` | [Register a physical facility](./rest/register-site.api.mdx) | `201` + `Location` |
| `GET` | `/sites` | [List every registered site](./rest/list-sites.api.mdx) | `200` |
| `GET` | `/sites/{siteCode}` | [Get one site](./rest/get-site.api.mdx) | `200` |

## Zones

Behavioral classifications scoped to a Site (ambient, chilled, frozen,
hazmat, forward-pick, reserve), bundling a LocationCode's Area and Zone
segments. Their `TemperatureClass` and `Hazmat` flag are exactly what
PlacementRules match on.

| Method | Path | Operation | Success |
|---|---|---|---|
| `POST` | `/sites/{siteCode}/zones` | [Register a zone](./rest/register-zone.api.mdx) | `201` + `Location` |
| `GET` | `/sites/{siteCode}/zones` | [List a site's zones](./rest/list-zones.api.mdx) | `200` |
| `GET` | `/zones/{zoneId}` | [Get one zone](./rest/get-zone.api.mdx) | `200` |

The zone identifier in the path is the composite `Site-Area-Zone`, e.g.
`WH1-STOR-AMB`.

## Aisles

Physical corridors scoped to a Zone. An Aisle carries a `SequenceHint` — its
walk-order position, the concrete travel-distance input WES-tier planning
needs — and a `Direction`.

| Method | Path | Operation | Success |
|---|---|---|---|
| `POST` | `/zones/{zoneId}/aisles` | [Register an aisle](./rest/register-aisle.api.mdx) | `201` + `Location` |
| `GET` | `/zones/{zoneId}/aisles` | [List a zone's aisles, in walk order](./rest/list-aisles.api.mdx) | `200` |
| `GET` | `/zones/{zoneId}/aisles/{aisleCode}` | [Get one aisle](./rest/get-aisle.api.mdx) | `200` |

`GET /zones/{zoneId}/aisles` returns aisles ordered by `sequenceHint`, not by
registration order. That ordering *is* the walk order.

## Location Types

Reusable classifications of physical slot shape/kind — `PalletRack`, `Shelf`,
`ToteWall`, `BulkFloor`, `Staging`, `Amnesty` — each carrying the default
capacity envelope its slots inherit.

| Method | Path | Operation | Success |
|---|---|---|---|
| `POST` | `/location-types` | [Register a location type](./rest/register-location-type.api.mdx) | `201` + `Location` |
| `GET` | `/location-types` | [List location types](./rest/list-location-types.api.mdx) | `200` |
| `GET` | `/location-types/{name}` | [Get one location type](./rest/get-location-type.api.mdx) | `200` |

## Placement Rules

Declarations of which LocationTypes are legal in which Zones. Deny always
wins; declaring any Allow rule for a zone turns that zone into an allow-list.

| Method | Path | Operation | Success |
|---|---|---|---|
| `POST` | `/placement-rules` | [Define a placement rule](./rest/define-placement-rule.api.mdx) | `201` + `Location` |
| `GET` | `/placement-rules` | [List placement rules](./rest/list-placement-rules.api.mdx) | `200` |
| `GET` | `/placement-rules/{ruleId}` | [Get one placement rule](./rest/get-placement-rule.api.mdx) | `200` |

## Locations

The coded leaf slots themselves — registering, reading, decommissioning, and
bulk-importing a whole building's layout from a structured export.

| Method | Path | Operation | Success |
|---|---|---|---|
| `POST` | `/locations` | [Register a location slot](./rest/register-location-slot.api.mdx) | `201` + `Location` |
| `GET` | `/locations/{locationCode}` | [Get one slot](./rest/get-location-slot.api.mdx) | `200` |
| `POST` | `/locations/{locationCode}/decommission` | [Decommission a slot](./rest/decommission-location-slot.api.mdx) | `204` |
| `POST` | `/locations/import` | [Import a facility layout](./rest/import-facility-layout.api.mdx) | `200` (partial-success report) |

`POST /locations` is the endpoint where the whole
[chain-of-custody invariant](../ddd/invariants.md) and every applicable
[PlacementRule](../business-context/domain-vision.md#placementrules-the-enforcement-point)
are enforced. See [Bulk import](./bulk-import.md) for the import report
shape.

## Layout — "draw the warehouse"

The readable, drawable projections. These are assembled across the aggregates
on read, never separately stored state.

| Method | Path | Operation | Returns |
|---|---|---|---|
| `GET` | `/sites/{siteCode}/layout` | [Get a site's full layout](./rest/get-site-layout.api.mdx) | Nested zones → aisles → slots, pre-ordered |
| `GET` | `/sites/{siteCode}/layout?format=svg` | same operation, `format` query parameter | `image/svg+xml` floor plan |
| `GET` | `/zones/{zoneId}/grid` | [Get a zone's 2D grid](./rest/get-zone-grid.api.mdx) | Explicit matrix: rows = Level, columns = (Aisle, Bay) in walk order |

Real output for both is on [Drawing the warehouse](./drawing-the-warehouse.md).

## Health

| Method | Path | Operation | Returns |
|---|---|---|---|
| `GET` | `/healthz` | [Liveness probe](./rest/get-healthz.api.mdx) | `200` |

## Coverage cross-check

The router in `internal/adapters/inbound/http/server.go` mounts exactly these
paths:

```go
r.Get("/healthz", s.handleHealthz)

r.Route("/sites", func(r chi.Router) {
    r.Post("/", s.handleRegisterSite)
    r.Get("/", s.handleListSites)
    r.Get("/{siteCode}", s.handleGetSite)
    r.Get("/{siteCode}/layout", s.handleGetSiteLayout)
    r.Post("/{siteCode}/zones", s.handleRegisterZone)
    r.Get("/{siteCode}/zones", s.handleListZones)
})

r.Route("/zones", func(r chi.Router) {
    r.Get("/{zoneId}", s.handleGetZone)
    r.Get("/{zoneId}/grid", s.handleGetZoneGrid)
    r.Post("/{zoneId}/aisles", s.handleRegisterAisle)
    r.Get("/{zoneId}/aisles", s.handleListAisles)
    r.Get("/{zoneId}/aisles/{aisleCode}", s.handleGetAisle)
})

r.Route("/location-types", func(r chi.Router) {
    r.Post("/", s.handleRegisterLocationType)
    r.Get("/", s.handleListLocationTypes)
    r.Get("/{name}", s.handleGetLocationType)
})

r.Route("/placement-rules", func(r chi.Router) {
    r.Post("/", s.handleDefinePlacementRule)
    r.Get("/", s.handleListPlacementRules)
    r.Get("/{ruleId}", s.handleGetPlacementRule)
})

r.Route("/locations", func(r chi.Router) {
    r.Post("/", s.handleRegisterLocationSlot)
    r.Post("/import", s.handleImportFacilityLayout)
    r.Get("/{locationCode}", s.handleGetLocationSlot)
    r.Post("/{locationCode}/decommission", s.handleDecommissionLocationSlot)
})
```

That is **17 distinct paths** and **22 operations**, all of which appear in
`apis/openapi.yaml`:

| Tag | Paths | Operations |
|---|---:|---:|
| Sites | 2 | 3 |
| Zones | 2 | 3 |
| Aisles | 2 | 3 |
| Location Types | 2 | 3 |
| Placement Rules | 2 | 3 |
| Locations | 4 | 4 |
| Layout | 2 | 2 |
| Health | 1 | 1 |
| **Total** | **17** | **22** |
