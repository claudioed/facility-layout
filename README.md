# facility-layout

The **Facility Layout** bounded context of the `warehouse-systems` platform:
the system of record for **where things physically are in the building**.

It owns the site's structural hierarchy — Site → Area → Zone → Aisle → Bay →
Level → Position — and the coded storage slots inside it. It does **not** own
occupancy or stock; that stays in `inventory-storage`'s `Bin`/`StockUnit`
aggregates. What it owns is whether a coded location **exists, is active, and
is legal for a given kind of storage unit**: the warehouse map that other
contexts read but never write.

**Strategic classification:** Generic Subdomain, extracted rather than
duplicated. It is an **Open Host Service** with a **Published Language** (its
domain events). It has no inbound dependency on any other service;
`inventory-storage`, `wes-work-planning`, `workforce-management` and
`fulfillment-execution` are downstream **Conformists** to what it publishes.

---

## Documentation

Full documentation is published at
**<https://claudioed.github.io/facility-layout/>** — business context and
ubiquitous language, the DDD model (subdomain classification, aggregates,
invariants, domain events), an interactive API reference generated from
`apis/openapi.yaml`, the ecosystem context map, and the Architecture Decision
Records. The site source lives in [`docs/`](docs/) and deploys to GitHub Pages
via `.github/workflows/docs.yml`.

---

## The location code

Not a made-up scheme — the widely-used WMS industry pattern, hyphen-joined
and human-parsable:

```
WH1-STOR-AMB-A07-03-02-B
 |    |    |   |   |  |  `-- Position: left-to-right slot on the level
 |    |    |   |   |  `----- Level:    vertical level / shelf
 |    |    |   |   `-------- Bay:      bay / section along the aisle
 |    |    |   `------------ Aisle:    physical corridor
 |    |    `---------------- Zone:     behavioral class (AMB/CHL/FRZ/HAZ/FWD/RSV)
 |    `--------------------- Area:     coarse functional area (STOR/RCV/PACK/STAGE)
 `-------------------------- Site:     the physical facility
```

`LocationCode` is a **value object**, never free text: seven typed segments,
each non-empty and `[A-Z0-9]` only, always round-tripping through
`String()`/`ParseLocationCode()`.

---

## Layering

Hexagonal / ports & adapters. Dependencies point **inward only**:

```
cmd/facility/                 main.go — composition root (the only place that knows all layers)
internal/
  domain/                     pure business logic; imports nothing but stdlib and itself
    shared/                   LocationCode, Capacity, TemperatureClass, Direction, Status, the 8 events
    site/  zone/  aisle/      the structural aggregates
    placement/                LocationType, PlacementRule, RuleSet evaluation
    slot/                     LocationSlot — the coded leaf aggregate
  application/
    ports/                    OUT interfaces only (repos, EventPublisher, Clock)
    usecases/                 one struct per use case + the two read-model assemblers
  adapters/
    inbound/http/             chi handlers, DTOs, RFC 7807 error mapping, SVG rendering
    outbound/postgres/        pgxpool repos + golang-migrate migrations
    outbound/memory/          thread-safe in-memory repos for tests and local runs
    outbound/events/          log + buffered publishers (broker-ready interface)
migrations/                   golang-migrate SQL
```

The rule is enforced as an executable test, not a convention:
`internal/architecture/architecture_test.go` (arch-go) fails the build if the
domain ever imports the application layer, if an adapter reaches sideways
into its sibling, or if `ports` grows a concrete type.

---

## Ubiquitous language

| Term | Meaning |
|---|---|
| **Site** | A physical facility/building. Root of the hierarchy. |
| **Zone** | A behavioral classification scoped to a Site, bundling the Area+Zone segments. Carries `TemperatureClass` (Ambient/Chilled/Frozen) and a `Hazmat` flag. Every `PlacementRule` is keyed by it. |
| **Aisle** | A physical corridor scoped to a Zone. Carries a `SequenceHint` (walk-order position — the concrete travel-distance input the WES tier needs) and a `Direction` (OneWay/TwoWay). |
| **LocationType** | A reusable classification of slot shape/kind (`PalletRack`, `Shelf`, `ToteWall`, `BulkFloor`, `Staging`, `Amnesty`) with a default capacity envelope. |
| **LocationSlot** | The coded leaf aggregate. Its identity **is** its `LocationCode`. Has a LocationType, a capacity envelope, and a Status (`Active`/`UnderMaintenance`/`Decommissioned`). |
| **PlacementRule** | Declares which LocationTypes are legal in which Zones. The mechanism that prevents "ambient product in the frozen zone" — enforced once, at registration time. |
| **Facility layout** | The readable, drawable projection of the whole structure: zones → aisles → slots, shaped for a UI to render directly. |

### How PlacementRules evaluate

A rule is `(LocationType, Effect, ZonePredicate)`. The predicate matches on
`zoneCode`, `temperatureClass` and/or `hazmat` — every set field must match
(AND); unset fields are wildcards; at least one must be set.

For a slot of type `T` going into a zone `Z`, `RuleSet.Check` runs, in order:

1. Any **matching `Deny` rule naming `T`** rejects it. Deny always wins.
2. If **any matching `Allow` rule exists for `Z` at all**, the zone becomes an
   allow-list: `T` must be named by one of them, or it is rejected.
3. Otherwise the zone is unconstrained and the placement is permitted.

The rejection error always names the specific rule that was violated.

### Lifecycle

Decommission is **one-way** in v1 for every aggregate: there is no
reactivation use case, and re-registering a decommissioned LocationCode is
rejected as a duplicate rather than quietly resurrecting the slot.
`UnderMaintenance` is a legal persisted state (e.g. loaded from an external
facility-management system) that the read models render; v1 exposes no use
case that sets it.

---

## Domain events (Published Language)

`SiteRegistered`, `ZoneRegistered`, `AisleRegistered`,
`LocationTypeRegistered`, `PlacementRuleDefined`, `LocationSlotRegistered`,
`LocationSlotDecommissioned`, `FacilityLayoutImported`.

CloudEvents `type` convention, identical to the other four services:

```
com.warehouse.<subdomain>.facility-layout.<entity>.<EventName>
```

This service's subdomain segment is `wms`. Examples:

```
com.warehouse.wms.facility-layout.locationslot.LocationSlotRegistered
com.warehouse.wms.facility-layout.zone.ZoneRegistered
com.warehouse.wms.facility-layout.placementrule.PlacementRuleDefined
```

---

## Running it

### Local, no database (in-memory adapters)

```sh
go run ./cmd/facility
# facility-layout 2026/08/22 09:00:00 DATABASE_URL not set; using in-memory adapters
# facility-layout 2026/08/22 09:00:00 listening on :8080
```

### With Postgres

```sh
docker compose up -d postgres
export DATABASE_URL="postgres://facility:facility@localhost:5432/facility?sslmode=disable"
go run ./cmd/facility          # migrations run automatically at startup
```

Migrations can also be applied standalone:

```sh
migrate -source file://migrations -database "$DATABASE_URL" up
```

### Configuration

| Env var | Default | Meaning |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Listen address |
| `DATABASE_URL` | *(unset)* | Postgres DSN. Unset ⇒ in-memory adapters + log-only event publishing |
| `MIGRATIONS_PATH` | `migrations` | Directory of golang-migrate SQL files |

### Container

```sh
docker build -t claudioed/facility-layout .
docker run --rm -p 8080:8080 claudioed/facility-layout
```

---

## API

Errors use **RFC 7807** (`application/problem+json`) throughout, from day
one. Every `201` carries a `Location` header that resolves.

### Structural (write side)

| Method | Path | Purpose | Success |
|---|---|---|---|
| `POST` | `/sites` | RegisterSite | `201` + `Location` |
| `GET` | `/sites` | List sites | `200` |
| `GET` | `/sites/{siteCode}` | Get one site | `200` |
| `POST` | `/sites/{siteCode}/zones` | RegisterZone | `201` + `Location` |
| `GET` | `/sites/{siteCode}/zones` | List a site's zones | `200` |
| `GET` | `/zones/{zoneId}` | Get one zone | `200` |
| `POST` | `/zones/{zoneId}/aisles` | RegisterAisle | `201` + `Location` |
| `GET` | `/zones/{zoneId}/aisles` | List a zone's aisles (walk order) | `200` |
| `GET` | `/zones/{zoneId}/aisles/{aisleCode}` | Get one aisle | `200` |
| `POST` | `/location-types` | RegisterLocationType | `201` + `Location` |
| `GET` | `/location-types` | List location types | `200` |
| `GET` | `/location-types/{name}` | Get one location type | `200` |
| `POST` | `/placement-rules` | DefinePlacementRule | `201` + `Location` |
| `GET` | `/placement-rules` | List placement rules | `200` |
| `GET` | `/placement-rules/{ruleId}` | Get one placement rule | `200` |
| `POST` | `/locations` | RegisterLocationSlot | `201` + `Location` |
| `GET` | `/locations/{locationCode}` | Get one slot | `200` |
| `POST` | `/locations/{locationCode}/decommission` | DecommissionLocationSlot | `204` |
| `POST` | `/locations/import` | ImportFacilityLayout (bulk, partial success) | `200` |

The four single-resource `GET`s (`/zones/{zoneId}`,
`/zones/{zoneId}/aisles/{aisleCode}`, `/location-types/{name}`,
`/placement-rules/{ruleId}`) exist so that every `201`'s `Location` header
points at something that actually has a representation — a `Location` with no
`GET` behind it is not REST maturity level 2.

### "Draw the warehouse" (read side)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/sites/{siteCode}/layout` | Full nested structure: zones → aisles → slots, pre-ordered for rendering |
| `GET` | `/sites/{siteCode}/layout?format=svg` | The same data rendered server-side as an SVG floor plan |
| `GET` | `/zones/{zoneId}/grid` | One zone as an explicit 2D matrix: rows = Level, columns = (Aisle, Bay) in walk order |
| `GET` | `/healthz` | Liveness |

### Status codes

| Code | When |
|---|---|
| `200` | Successful read, or a bulk-import report |
| `201` | Resource created (always with `Location`) |
| `204` | Successful action with no body (decommission) |
| `400` | Malformed input: bad JSON, a location code that is not seven `[A-Z0-9]` segments, a missing required identifier |
| `404` | The named site/zone/aisle/slot/type/rule does not exist |
| `409` | State conflict: a code already taken, a parent that exists but is not Active, a slot already decommissioned |
| `422` | Semantically invalid: non-positive capacity, unknown enum value, **PlacementRule violation** |

---

## Worked example

Every response below is **real output** captured from a running instance.

### 1 — Register the site

```sh
curl -i -X POST localhost:8080/sites -H 'Content-Type: application/json' \
  -d '{"siteCode":"WH1","name":"Fulfilment Centre One"}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /sites/WH1

{"siteCode":"WH1","name":"Fulfilment Centre One","status":"Active"}
```

### 2 — Register a zone

```sh
curl -i -X POST localhost:8080/sites/WH1/zones -H 'Content-Type: application/json' \
  -d '{"areaCode":"STOR","zoneCode":"AMB","temperatureClass":"Ambient","hazmat":false}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /zones/WH1-STOR-AMB

{"zoneId":"WH1-STOR-AMB","siteCode":"WH1","areaCode":"STOR","zoneCode":"AMB","temperatureClass":"Ambient","hazmat":false,"status":"Active"}
```

### 3 — Register an aisle

```sh
curl -i -X POST localhost:8080/zones/WH1-STOR-AMB/aisles -H 'Content-Type: application/json' \
  -d '{"aisleCode":"A07","sequenceHint":7,"direction":"TwoWay"}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /zones/WH1-STOR-AMB/aisles/A07

{"aisleId":"WH1-STOR-AMB-A07","zoneId":"WH1-STOR-AMB","aisleCode":"A07","sequenceHint":7,"direction":"TwoWay","status":"Active"}
```

### 4 — Register a location type

```sh
curl -i -X POST localhost:8080/location-types -H 'Content-Type: application/json' \
  -d '{"name":"PalletRack","defaultCapacity":{"maxWeightKg":1200,"maxVolumeM3":2.4}}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /location-types/PalletRack

{"name":"PalletRack","defaultCapacity":{"maxWeightKg":1200,"maxVolumeM3":2.4}}
```

### 5 — Define a placement rule

Pallet racking is not rated for the cold, so deny it anywhere `Frozen`:

```sh
curl -i -X POST localhost:8080/placement-rules -H 'Content-Type: application/json' \
  -d '{"ruleId":"RULE-FRZ-NO-SHELF","locationType":"PalletRack","effect":"Deny","zone":{"temperatureClass":"Frozen"}}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /placement-rules/RULE-FRZ-NO-SHELF

{"ruleId":"RULE-FRZ-NO-SHELF","locationType":"PalletRack","effect":"Deny","zone":{"temperatureClass":"Frozen"},"description":"RULE-FRZ-NO-SHELF: Deny PalletRack where temperatureClass=Frozen"}
```

### 6 — Register a slot

```sh
curl -i -X POST localhost:8080/locations -H 'Content-Type: application/json' \
  -d '{"locationCode":"WH1-STOR-AMB-A07-03-02-B","locationType":"PalletRack"}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /locations/WH1-STOR-AMB-A07-03-02-B

{"locationCode":"WH1-STOR-AMB-A07-03-02-B","zoneId":"WH1-STOR-AMB","aisleId":"WH1-STOR-AMB-A07","coordinates":{"site":"WH1","area":"STOR","zone":"AMB","aisle":"A07","bay":"03","level":"02","position":"B"},"locationType":"PalletRack","capacity":{"maxWeightKg":1200,"maxVolumeM3":2.4},"status":"Active"}
```

(Two more slots were added the same way for the examples below:
`WH1-STOR-AMB-A07-03-02-A` and `WH1-STOR-AMB-A07-03-01-A`.)

### 7 — GET the layout

```sh
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
                }
            ]
        }
    ],
    "totals": {
        "zones": 1,
        "aisles": 1,
        "slots": 3
    }
}
```

Zones come back ordered by id, aisles in **`sequenceHint` walk order** (not
registration order), and slots ordered bay → level → position. A client
renders it top-down with no joins and no sorting.

### 8 — GET the grid

```sh
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
        { "aisleId": "WH1-STOR-AMB-A07", "aisleCode": "A07", "bay": "03", "sequenceHint": 7 }
    ],
    "levels": [ "01", "02" ],
    "rows": [
        {
            "level": "01",
            "cells": [
                { "positions": [
                    { "locationCode": "WH1-STOR-AMB-A07-03-01-A", "position": "A", "locationType": "PalletRack", "status": "Active" }
                ] }
            ]
        },
        {
            "level": "02",
            "cells": [
                { "positions": [
                    { "locationCode": "WH1-STOR-AMB-A07-03-02-A", "position": "A", "locationType": "PalletRack", "status": "Active" },
                    { "locationCode": "WH1-STOR-AMB-A07-03-02-B", "position": "B", "locationType": "PalletRack", "status": "Active" }
                ] }
            ]
        }
    ]
}
```

`rows[i].cells` is **index-aligned with `columns`**, and a cell is literal
JSON `null` where the rack has a gap. Iterate rows × columns and paint.

### 9 — GET the SVG floor plan

```sh
curl "localhost:8080/sites/WH1/layout?format=svg" -o wh1.svg
open wh1.svg
```

```
200 image/svg+xml
```

```xml
<svg xmlns="http://www.w3.org/2000/svg" width="640" height="240" viewBox="0 0 640 240">
  <title>WH1 facility layout</title>
  <rect x="0" y="0" width="640" height="240" fill="#ffffff"/>
  <text x="24" y="26" font-family="monospace" font-size="16" font-weight="bold" fill="#1f2933">WH1 — Fulfilment Centre One</text>
  ...
</svg>
```

One colored band per zone (hue = temperature class, amber = hazmat), one row
of rects per aisle, one rect per slot with a `<title>` tooltip. It is a thin
adapter-only concern: the render function lives in the HTTP layer and
consumes the same read model the JSON endpoint does.

---

## Bulk import

`POST /locations/import` takes a JSON array of fully-specified rows and
bootstraps a whole building in one call, creating the Site/Zone/Aisle
structure on first sight. Validation is **atomic per row**: every row is
processed, and the report says exactly which rows failed and why. A 500-row
export with 3 bad rows still creates the other 497.

```sh
curl -X POST localhost:8080/locations/import -H 'Content-Type: application/json' -d '[
 {"siteCode":"WH1","siteName":"Fulfilment Centre One","areaCode":"STOR","zoneCode":"FRZ","temperatureClass":"Frozen","hazmat":false,"aisleCode":"A02","sequenceHint":2,"direction":"TwoWay","bay":"01","level":"01","position":"A","locationType":"PalletRack"},
 {"siteCode":"WH1","areaCode":"STOR","zoneCode":"FRZ","temperatureClass":"Frozen","hazmat":false,"aisleCode":"A02","sequenceHint":2,"direction":"TwoWay","bay":"01","level":"01","position":"b","locationType":"PalletRack"}
]'
```

```json
{
    "rowsSubmitted": 2,
    "slotsImported": 0,
    "rowsRejected": 2,
    "results": [
        {
            "index": 0,
            "locationCode": "WH1-STOR-FRZ-A02-01-01-A",
            "succeeded": false,
            "error": "location type violates a placement rule for this zone: PalletRack is denied in zone WH1-STOR-FRZ by rule [RULE-FRZ-NO-SHELF: Deny PalletRack where temperatureClass=Frozen]"
        },
        {
            "index": 1,
            "locationCode": "",
            "succeeded": false,
            "error": "location code segment must contain only uppercase letters and digits: position segment \"b\""
        }
    ]
}
```

Row 0 was rejected by the rule defined in step 5 — this is the
"ambient product in the frozen zone" guard doing its job, at registration
time, naming the exact rule violated. Row 1 was rejected by the
`LocationCode` value object before it ever reached the domain.

The endpoint answers `200 OK`, not `201`: a bulk import is a partial-success
report over many rows, not the creation of one addressable resource, so there
is no single `Location` to hand back.

---

## Errors (RFC 7807)

```sh
curl -i localhost:8080/sites/NOPE
```

```http
HTTP/1.1 404 Not Found
Content-Type: application/problem+json

{"type":"https://errors.facility-layout.warehouse-systems.dev/site-not-found","title":"Site not found","status":404,"detail":"site not found","instance":"/sites/NOPE"}
```

```sh
curl -i -X POST localhost:8080/locations -H 'Content-Type: application/json' \
  -d '{"locationCode":"WH1-STOR-AMB-A07-03-02-B","locationType":"PalletRack"}'
```

```http
HTTP/1.1 409 Conflict
Content-Type: application/problem+json

{"type":"https://errors.facility-layout.warehouse-systems.dev/duplicate-location-code","title":"A location slot with this code already exists","status":409,"detail":"a location slot with this code already exists","instance":"/locations"}
```

---

## API specification

`apis/openapi.yaml` (OpenAPI 3.0.3) documents every route with full
request/response schemas, a shared `Problem` component, and real
domain-grounded examples. It is linted in CI:

```sh
spectral lint apis/openapi.yaml --ruleset .spectral.yaml --fail-severity=warn
```

---

## Tests & quality gates

```sh
go build ./...
go vet ./...
go test ./... -race
gofmt -l .
golangci-lint run ./...

# combined domain + application coverage (CI gate: >= 90%)
go test ./internal/domain/... ./internal/application/... -race \
  -coverprofile=coverage.out \
  -coverpkg=./internal/domain/...,./internal/application/...
go tool cover -func=coverage.out | tail -1

# Postgres integration tests (build-tagged; skipped without DATABASE_URL)
docker compose up -d postgres
DATABASE_URL="postgres://facility:facility@localhost:5432/facility?sslmode=disable" \
  go test -tags=integration ./... -race -count=1

# BDD acceptance suite (godog/Gherkin, over the real HTTP API)
go test ./... -run TestFeatures -v

# architecture fitness tests (arch-go)
go test ./internal/architecture/... -v

# mutation testing (exploratory, never blocking — see MUTATION.md)
gremlins unleash ./internal/domain --workers 1 --timeout-coefficient 30
```

CI (`.github/workflows/ci.yml`) runs `lint`, `test` (with the 90% coverage
gate), `bdd`, `integration`, `api-lint`, `helm-lint` and `arch-test` on every
push and PR; `mutation` runs only on schedule/manual dispatch; and
`docker-publish` pushes to Docker Hub on `main` once every other job is green.

## Helm chart

```sh
helm install facility-layout ./charts/facility-layout \
  --set image.repository=claudioed/facility-layout \
  --set database.url="postgres://facility:facility@postgres:5432/facility?sslmode=disable"
```

Linted in CI with `ct lint --charts charts/facility-layout`.
