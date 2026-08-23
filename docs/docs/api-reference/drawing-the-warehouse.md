---
id: drawing-the-warehouse
title: Drawing the warehouse
sidebar_label: Drawing the warehouse
description: The two headline read endpoints, with the actual JSON and the actual SVG a running instance returns.
---

# Drawing the warehouse

This is the capability the service exists to expose, and the reason its read
models are first-class rather than reporting afterthoughts: **a warehouse map
that cannot be drawn is a database table.**

Two endpoints return *renderable structure*, not raw rows a client has to
join and sort:

| Endpoint | Shape | Use it for |
|---|---|---|
| `GET /sites/{siteCode}/layout` | Nested zones → aisles → slots, pre-ordered | A floor plan, a tree view, a whole-building export |
| `GET /zones/{zoneId}/grid` | An explicit 2D matrix | Painting one zone as a rack elevation / warehouse map |
| `GET /sites/{siteCode}/layout?format=svg` | Server-rendered SVG | A picture you can `curl` straight into a file and open |

Every response on this page is **real output captured from a running
instance** seeded with the six calls from
[Getting started](../overview/getting-started.md), plus a bulk import that
added aisle `A08`.

## `GET /sites/{siteCode}/layout`

```bash
curl localhost:8080/sites/WH1/layout
```

```json
{
    "site": {
        "siteCode": "WH1",
        "name": "Fulfilment Centre One",
        "status": "Active"
    },
    "zones": [
        {
            "zoneId": "WH1-STOR-AMB",
            "siteCode": "WH1",
            "areaCode": "STOR",
            "zoneCode": "AMB",
            "temperatureClass": "Ambient",
            "hazmat": false,
            "status": "Active",
            "aisles": [
                {
                    "aisleId": "WH1-STOR-AMB-A07",
                    "zoneId": "WH1-STOR-AMB",
                    "aisleCode": "A07",
                    "sequenceHint": 7,
                    "direction": "TwoWay",
                    "status": "Active",
                    "slots": [
                        {
                            "locationCode": "WH1-STOR-AMB-A07-03-01-A",
                            "zoneId": "WH1-STOR-AMB",
                            "aisleId": "WH1-STOR-AMB-A07",
                            "coordinates": {
                                "site": "WH1", "area": "STOR", "zone": "AMB", "aisle": "A07",
                                "bay": "03", "level": "01", "position": "A"
                            },
                            "locationType": "PalletRack",
                            "capacity": { "maxWeightKg": 1200, "maxVolumeM3": 2.4 },
                            "status": "Active"
                        },
                        {
                            "locationCode": "WH1-STOR-AMB-A07-03-02-A",
                            "zoneId": "WH1-STOR-AMB",
                            "aisleId": "WH1-STOR-AMB-A07",
                            "coordinates": {
                                "site": "WH1", "area": "STOR", "zone": "AMB", "aisle": "A07",
                                "bay": "03", "level": "02", "position": "A"
                            },
                            "locationType": "PalletRack",
                            "capacity": { "maxWeightKg": 1200, "maxVolumeM3": 2.4 },
                            "status": "Active"
                        },
                        {
                            "locationCode": "WH1-STOR-AMB-A07-03-02-B",
                            "zoneId": "WH1-STOR-AMB",
                            "aisleId": "WH1-STOR-AMB-A07",
                            "coordinates": {
                                "site": "WH1", "area": "STOR", "zone": "AMB", "aisle": "A07",
                                "bay": "03", "level": "02", "position": "B"
                            },
                            "locationType": "PalletRack",
                            "capacity": { "maxWeightKg": 1200, "maxVolumeM3": 2.4 },
                            "status": "Active"
                        }
                    ]
                },
                {
                    "aisleId": "WH1-STOR-AMB-A08",
                    "zoneId": "WH1-STOR-AMB",
                    "aisleCode": "A08",
                    "sequenceHint": 8,
                    "direction": "TwoWay",
                    "status": "Active",
                    "slots": [
                        {
                            "locationCode": "WH1-STOR-AMB-A08-01-01-A",
                            "zoneId": "WH1-STOR-AMB",
                            "aisleId": "WH1-STOR-AMB-A08",
                            "coordinates": {
                                "site": "WH1", "area": "STOR", "zone": "AMB", "aisle": "A08",
                                "bay": "01", "level": "01", "position": "A"
                            },
                            "locationType": "PalletRack",
                            "capacity": { "maxWeightKg": 1200, "maxVolumeM3": 2.4 },
                            "status": "Active"
                        }
                    ]
                }
            ]
        },
        {
            "zoneId": "WH1-STOR-FRZ",
            "siteCode": "WH1",
            "areaCode": "STOR",
            "zoneCode": "FRZ",
            "temperatureClass": "Frozen",
            "hazmat": false,
            "status": "Active",
            "aisles": [
                {
                    "aisleId": "WH1-STOR-FRZ-A02",
                    "zoneId": "WH1-STOR-FRZ",
                    "aisleCode": "A02",
                    "sequenceHint": 2,
                    "direction": "OneWay",
                    "status": "Active",
                    "slots": []
                }
            ]
        }
    ],
    "totals": {
        "zones": 2,
        "aisles": 3,
        "slots": 4
    }
}
```

### What the ordering guarantees buy you

| Level | Ordered by | Why it matters |
|---|---|---|
| `zones[]` | zone id | Stable, diffable output between calls |
| `aisles[]` | **`sequenceHint` walk order**, not registration order | The array order *is* the pick path |
| `slots[]` | bay → level → position | Renders as a rack elevation with no client-side sort |

A client renders this top-down with **no joins and no sorting**. That is the
design intent: the server already knows the walk order, so making every
consumer re-derive it would be duplicating the one thing this context is the
source of truth for.

`totals` is included so a UI can show "2 zones · 3 aisles · 4 slots" without
walking the tree, and so a bulk-load can be sanity-checked at a glance.

Note the empty `slots: []` under `WH1-STOR-FRZ-A02` — the frozen aisle exists
but has no slots, because the only registered LocationType (`PalletRack`) is
denied there by `RULE-FRZ-NO-SHELF`. An aisle with no legal slots still
appears in the layout: it is real physical structure, and hiding it would
make the drawn map disagree with the building.

## `GET /zones/{zoneId}/grid`

```bash
curl localhost:8080/zones/WH1-STOR-AMB/grid
```

```json
{
    "zone": {
        "zoneId": "WH1-STOR-AMB",
        "siteCode": "WH1",
        "areaCode": "STOR",
        "zoneCode": "AMB",
        "temperatureClass": "Ambient",
        "hazmat": false,
        "status": "Active"
    },
    "columns": [
        { "aisleId": "WH1-STOR-AMB-A07", "aisleCode": "A07", "bay": "03", "sequenceHint": 7 },
        { "aisleId": "WH1-STOR-AMB-A08", "aisleCode": "A08", "bay": "01", "sequenceHint": 8 }
    ],
    "levels": [ "01", "02" ],
    "rows": [
        {
            "level": "01",
            "cells": [
                { "positions": [
                    { "locationCode": "WH1-STOR-AMB-A07-03-01-A", "position": "A", "locationType": "PalletRack", "status": "Active" }
                ] },
                { "positions": [
                    { "locationCode": "WH1-STOR-AMB-A08-01-01-A", "position": "A", "locationType": "PalletRack", "status": "Active" }
                ] }
            ]
        },
        {
            "level": "02",
            "cells": [
                { "positions": [
                    { "locationCode": "WH1-STOR-AMB-A07-03-02-A", "position": "A", "locationType": "PalletRack", "status": "Active" },
                    { "locationCode": "WH1-STOR-AMB-A07-03-02-B", "position": "B", "locationType": "PalletRack", "status": "Active" }
                ] },
                null
            ]
        }
    ]
}
```

Read the matrix: two columns — aisle `A07` bay `03`, then aisle `A08` bay
`01`, in `sequenceHint` order — and two level rows. Aisle `A08` has nothing on
level `02`, so `rows[1].cells[1]` is a literal **`null`**: the gap in the rack
is explicit in the payload, at the exact index the renderer expects it.

### The contract that makes this paintable

- **`rows[i].cells` is index-aligned with `columns`.** `cells[j]` is always
  the cell for `columns[j]`. No lookup, no matching on ids.
- **A cell is literal JSON `null` where the rack has a gap.** Not an empty
  object, not an omitted index — `null`, so the array length always equals
  `columns.length` and the grid never shifts.
- **`columns` are `(Aisle, Bay)` pairs in aisle `sequenceHint` order.** Read
  left to right and you are walking the zone.
- **`levels` is the row axis**, ascending — the same values as
  `rows[i].level`, provided separately so a renderer can size the grid before
  iterating.

So the entire client-side rendering is:

```js
for (const row of grid.rows) {
  for (const [j, cell] of row.cells.entries()) {
    paint(grid.columns[j], row.level, cell); // cell may be null
  }
}
```

No layout maths, no grouping, no sorting.

## `GET /sites/{siteCode}/layout?format=svg`

The same read model, rendered server-side as a minimal floor plan.

```bash
curl "localhost:8080/sites/WH1/layout?format=svg" -o wh1.svg
open wh1.svg
```

```
200 image/svg+xml
```

Here is the actual file that command produced, embedded:

![WH1 facility layout, rendered server-side as SVG](/img/wh1-layout.svg)

And its actual source — 14 lines, no external references, no scripts:

```xml
<svg xmlns="http://www.w3.org/2000/svg" width="640" height="280" viewBox="0 0 640 280">
  <title>WH1 facility layout</title>
  <rect x="0" y="0" width="640" height="280" fill="#ffffff"/>
  <text x="24" y="26" font-family="monospace" font-size="16" font-weight="bold" fill="#1f2933">WH1 — Fulfilment Centre One</text>
  <rect x="24" y="40" width="592" height="114" fill="#f5f7fa" stroke="#7b8794" stroke-width="1" rx="6"/>
  <text x="34" y="61" font-family="monospace" font-size="13" font-weight="bold" fill="#1f2933">WH1-STOR-AMB  (Ambient)</text>
  <text x="34" y="86" font-family="monospace" font-size="11" fill="#3e4c59">aisle A07 (seq 7, TwoWay)</text>
  <rect x="174" y="92" width="26" height="16" fill="#9fb3c8" stroke="#52606d" stroke-width="0.5" rx="2"><title>WH1-STOR-AMB-A07-03-01-A (PalletRack, Active)</title></rect>
  <rect x="203" y="92" width="26" height="16" fill="#9fb3c8" stroke="#52606d" stroke-width="0.5" rx="2"><title>WH1-STOR-AMB-A07-03-02-A (PalletRack, Active)</title></rect>
  <rect x="232" y="92" width="26" height="16" fill="#9fb3c8" stroke="#52606d" stroke-width="0.5" rx="2"><title>WH1-STOR-AMB-A07-03-02-B (PalletRack, Active)</title></rect>
  <text x="34" y="126" font-family="monospace" font-size="11" fill="#3e4c59">aisle A08 (seq 8, TwoWay)</text>
  <rect x="174" y="132" width="26" height="16" fill="#9fb3c8" stroke="#52606d" stroke-width="0.5" rx="2"><title>WH1-STOR-AMB-A08-01-01-A (PalletRack, Active)</title></rect>
  <rect x="24" y="170" width="592" height="74" fill="#e0f0ff" stroke="#0b69a3" stroke-width="1" rx="6"/>
  <text x="34" y="191" font-family="monospace" font-size="13" font-weight="bold" fill="#1f2933">WH1-STOR-FRZ  (Frozen)</text>
  <text x="34" y="216" font-family="monospace" font-size="11" fill="#3e4c59">aisle A02 (seq 2, OneWay)</text>
</svg>
```

Reading it against the data:

- One **colored band per zone**, hue chosen from the zone's
  `temperatureClass` — grey `#f5f7fa` for Ambient, blue `#e0f0ff` for Frozen
  (amber for hazmat).
- One **labelled row per aisle**, in `sequenceHint` order, annotated with its
  sequence and direction: `aisle A07 (seq 7, TwoWay)`.
- One **`<rect>` per slot**, each carrying a `<title>` so a browser shows the
  full LocationCode, type and status on hover.
- The empty frozen aisle draws its band and label with no slot rects —
  exactly matching the JSON above.

### It stays in the adapter

SVG rendering is a thin, adapter-only concern. The render function lives in
`internal/adapters/inbound/http/svg.go` and consumes **the same read model**
the JSON endpoint does. No domain code, no application code and no port knows
that SVG exists. Swapping it for PNG, DXF or a canvas payload would touch one
file.

This was a stretch goal in the build plan, explicitly ranked below shipping
the JSON layout and grid solidly. It got built because those landed first.
