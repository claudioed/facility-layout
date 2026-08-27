# Project: Facility Layout (Generic Subdomain — physical warehouse structure)

The system of record for **where things physically are in the building**:
the site's structural hierarchy (site, area, zone, aisle) and the coded
storage slots inside it. This context does NOT own occupancy or stock — that
stays in `inventory-storage`'s `Bin`/`StockUnit` aggregates. It owns whether a
coded location **exists, is active, and is legal for a given kind of storage
unit** — the "warehouse map" that other contexts read but never write.

Source of truth for the domain model: `/Users/claudioed/docs/amazon-fulfillment-ddd.md`
and `/Users/claudioed/warehouse-systems-ddd.md`. Honor that ubiquitous language.

## Strategic classification (read this before writing any code)

**Generic Subdomain**, same bucket as Cartonization and WCS in
`warehouse-systems-ddd.md` — well-understood, not a competitive
differentiator, and explicitly the kind of concern that doc says to
**extract rather than duplicate** ("model this as a separate Generic
Subdomain both contexts call into" — the same reasoning applied there to
Cartonization applies here to physical location). `inventory-storage` (WMS
tier) needs location validity to accept a stow; `wes-work-planning` /
`fulfillment-execution` (WES tier) need zone/aisle adjacency for travel-path
and congestion reasoning. Neither owns it; both consume it. This is why it
is its own bounded context and its own service, not a package bolted onto
`inventory-storage`.

**Relationship to the rest of the system**: this service is an **Open Host
Service** with a **Published Language** (its domain events). It has NO
inbound dependency on any of the other four services and never will —
`inventory-storage`/`wes-work-planning`/`workforce-management`/
`fulfillment-execution` are all downstream **Conformists** to whatever this
service publishes. This service does not reach into any of their
aggregates, and none of them get write access to this one's. (Actually
wiring that consumption is a separate, later, additive task in those
repos — out of scope here. This task builds `facility-layout` as a complete,
independently correct, standalone service.)

## Architecture (NON-NEGOTIABLE — identical to the other four services)

Hexagonal / Ports & Adapters. Strict dependency rule: **domain depends on
nothing; application depends on domain; adapters depend on
application/domain.** No framework or SQL types in the domain layer.

```
cmd/facility/                 main.go — composition root
internal/
  domain/
    site/                     Site aggregate
    zone/                     Zone aggregate (area + behavioral zone, scoped to a Site)
    aisle/                    Aisle aggregate (scoped to a Zone; travel metadata)
    slot/                     LocationSlot aggregate (the coded leaf location)
    placement/                LocationType + PlacementRule (which unit types are legal where)
    shared/                   value objects: LocationCode, Capacity, events
  application/
    ports/                    OUT interfaces: SiteRepo, ZoneRepo, AisleRepo, SlotRepo,
                               PlacementRuleRepo, EventPublisher, Clock
    usecases/                 one struct per use case
  adapters/
    inbound/http/             chi handlers, DTOs, error mapping
    outbound/postgres/        pgxpool repos + migrations
    outbound/memory/          in-memory repos for tests/local
    outbound/events/          log/buffered publisher (kafka-ready iface)
migrations/                   golang-migrate SQL files
```

## Integration publishing & analytics data product (ADR-0009, ADR-0010)

This is an **Open Host Service**: its domain events are its Published Language.

- **Integration (ADR-0009):** a `outbound/kafka` publisher emits every domain
  event to `warehouse.facility.events`, selected by `EVENT_PUBLISHER=kafka`.
  (The `outbound/events` log/outbox publisher remains the default when
  `EVENT_PUBLISHER` is unset.) This service has no OTel package, so the
  publisher is trace-free by design.
- **Analytics (ADR-0010):** an additive read side built from this service's OWN
  events. OLTP domain/application are NOT modified and must NOT import the
  analytics store (arch-test enforces); `internal/analytics/report/` depends on
  nothing. A SECOND kafka adapter publishes to `warehouse.facility.analytics`
  (fanned alongside the integration publisher). Separate analytical Postgres
  (`ANALYTICS_DATABASE_URL`), `migrations/analytics/`, read-only reader role.
  Three processes: `cmd/facility` (OLTP), `cmd/facility-projector` (only writer;
  consumes from FirstOffset, idempotent on event_id), `cmd/facility-reports`
  (read-only reader, `GET /reports/...`); MCP report tool too.
- **Report:** **Layout Catalog Growth & Change**, keyed per site/zone × DAY
  bucket (slow-changing catalog): slots registered/decommissioned, zones/aisles/
  types/rules added, bulk imports. `GET /reports/.../freshness` reports lag.

## The location-code hierarchy (INDUSTRY STANDARD — use this exact shape)

This is not a made-up scheme. It is the widely-used WMS industry pattern:
**Site → Area → Zone → Aisle → Bay → Level → Position**, human-parsable and
hyphen-joined, e.g. `WH1-STOR-AMB-A07-03-02-B`
(Site=`WH1`, Area=`STOR`, Zone=`AMB`, Aisle=`A07`, Bay=`03`, Level=`02`,
Position=`B`). Segments read left→right, coarsest to finest:

| Segment  | Meaning                                              | Example |
|----------|-------------------------------------------------------|---------|
| Site     | the physical facility/building                        | `WH1`   |
| Area     | coarse functional area                                 | `STOR` (storage), `RCV` (receiving), `PACK`, `STAGE` |
| Zone     | behavioral class *within* an area — drives rules       | `AMB` (ambient), `CHL` (chilled), `FRZ` (frozen), `HAZ` (hazmat), `FWD` (forward-pick), `RSV` (reserve) |
| Aisle    | physical corridor                                      | `A07`   |
| Bay      | a bay/section along the aisle                          | `03`    |
| Level    | vertical level/shelf                                   | `02`    |
| Position | left-to-right slot on that level                       | `B`     |

`LocationCode` is a **value object**, not a free-text string: it is built
from these seven typed segments and always round-trips through
`String()`/`ParseLocationCode()`. Reject construction if any segment is
empty or contains characters other than `[A-Z0-9]`.

## Ubiquitous Language (use these exact names)

- **Site** — a physical facility/building. Root of the hierarchy. Has a
  `SiteCode` and a human name.
- **Zone** — a behavioral classification scoped to a Site (bundles the
  Area+Zone segments above into one aggregate — see "Aggregates" below for
  why they are combined). Zones are not cosmetic: they carry
  `TemperatureClass` (Ambient/Chilled/Frozen) and a `Hazmat` flag, and every
  `PlacementRule` is keyed by Zone. This is the SAME `Zone` word
  `warehouse-systems-ddd.md` already uses for WES's congestion/travel-path
  reasoning — this service becomes its source of truth.
- **Aisle** — a physical corridor scoped to a Zone. Carries a
  `SequenceHint` (its walk-order position for travel-path optimization,
  the concrete answer to WES's currently-missing travel-distance input) and
  a `Direction` (one-way or two-way).
- **LocationType** — a reusable classification of physical slot shape/kind
  (`PalletRack`, `Shelf`, `ToteWall`, `BulkFloor`, `Staging`, `Amnesty` —
  `amnesty` is a real term from the domain reference doc: where a
  damaged/mismatched item is set aside during stow). Carries a default
  capacity envelope (max weight, max volume).
- **LocationSlot** — the leaf aggregate: one coded physical slot. Its
  identity IS its `LocationCode`. Has a `LocationType`, a capacity envelope
  (can override the type's default), and a `Status`
  (`Active`/`Decommissioned`/`UnderMaintenance`). This is the thing
  `inventory-storage`'s `Bin` aggregate will eventually be validated
  against (that wiring is a separate future task — see the Strategic
  Classification section above).
- **PlacementRule** — declares which `LocationType`s are legal in which
  `Zone` (e.g. only a `PalletRack` LocationType may be placed in a `HAZ`
  zone if the rule set says so; a `Frozen`-temperature-class zone rejects a
  LocationType not rated for cold). This is the actual mechanism that
  prevents "ambient product in the frozen zone" — enforced once, at
  registration time, not re-checked by every caller.
- **Facility layout** — the readable, "drawable" projection of the whole
  structure: a Site's Zones, each Zone's Aisles, each Aisle's LocationSlots,
  assembled into a shape a UI/operator can literally render as a floor plan
  or grid. This is a first-class capability of this service (see REST API).
- **Location classification (read)** — `GET /locations/{locationCode}/classification`
  resolves a slot to its Zone and returns the Zone's `Hazmat`/`TemperatureClass`
  attributes as a cheap, denormalized read. This is the concrete Published
  Language realization of this context's Open Host Service role: `inventory-storage`
  consumes it synchronously at stow time to enforce hazmat/temperature
  placement rules on classified SKUs, without duplicating Zone data (ADR-0008).

## Aggregates & invariants (enforce in domain, unit-tested)

- **Site**: `SiteCode` must be non-empty, uppercase alphanumeric, unique.
  Cannot register a `Zone` against an unknown or `Decommissioned` Site.
- **Zone**: scoped to exactly one Site; `AreaCode` + `ZoneCode` together
  form the LocationCode's second and third segments. Cannot register an
  `Aisle` against an unknown or `Decommissioned` Zone.
- **Aisle**: scoped to exactly one Zone. Cannot register a `LocationSlot`
  whose LocationCode's Site/Area/Zone/Aisle segments don't resolve to an
  existing, `Active` Aisle (and transitively Zone and Site) — **this is the
  core invariant of the whole service**: no orphan slots, ever. Registering
  a slot is a chain-of-custody check, not a bare insert.
- **LocationSlot**: `LocationCode` is globally unique (it IS the identity).
  A slot's `LocationType` must satisfy every `PlacementRule` that applies to
  its Zone, checked at registration time — reject with a clear domain error
  naming the violated rule if not. Capacity envelope (weight/volume) must be
  positive. A `Decommissioned` slot cannot be re-activated by re-registering
  the same code — it must go through an explicit reactivation use case (or,
  for v1, simply stays a one-way transition — decide and document which,
  keep it simple: one-way decommission is fine for v1).
- **PlacementRule**: references an existing `LocationType` and either a
  specific `Zone` or a `TemperatureClass`/`Hazmat` predicate; cannot
  reference a `LocationType` that does not exist.
- Read models (a Site's full layout, a Zone's grid) are PROJECTIONS built by
  querying across these aggregates via the repositories — not separately
  stored state.

## Domain events (past tense)

SiteRegistered, ZoneRegistered, AisleRegistered, LocationTypeRegistered,
PlacementRuleDefined, LocationSlotRegistered, LocationSlotDecommissioned,
FacilityLayoutImported (emitted once per bulk import call, carrying a count
of slots imported — the individual `LocationSlotRegistered` events also
fire per-slot within that same import).

CloudEvents `type` convention (IDENTICAL to the other four services —
reverse-DNS, lowercase except the final PascalCase event name, entity
segment has NO hyphen even for multi-word aggregate names, matching e.g.
workforce-management's `shiftplan`):

```
com.warehouse.<subdomain>.facility-layout.<entity>.<EventName>
```

This service's **subdomain segment is `wms`** — "bin-accurate location" is
already classified WMS-tier in `amazon-fulfillment-ddd.md` ("Inventory &
Slotting" Core subdomain references bin-accurate location as WMS's Open
Host Service), and this service is the generalized, multi-consumer version
of that same concern. Entity segments: `site`, `zone`, `aisle`,
`locationtype`, `placementrule`, `locationslot`. Examples:

```
com.warehouse.wms.facility-layout.locationslot.LocationSlotRegistered
com.warehouse.wms.facility-layout.zone.ZoneRegistered
com.warehouse.wms.facility-layout.placementrule.PlacementRuleDefined
```

## Use cases (application layer)

1. RegisterSite(siteCode, name) -> Site
2. RegisterZone(siteCode, areaCode, zoneCode, temperatureClass, hazmat) -> Zone
3. RegisterAisle(zoneRef, aisleCode, sequenceHint, direction) -> Aisle
4. RegisterLocationType(name, defaultCapacity) -> LocationType
5. DefinePlacementRule(locationType, zonePredicate) -> PlacementRule
6. RegisterLocationSlot(locationCode, locationType, capacityOverride?) ->
   LocationSlot (validates the full chain-of-custody + placement rules)
7. DecommissionLocationSlot(locationCode) -> marks Decommissioned
8. ImportFacilityLayout(rows[]) -> bulk-registers sites/zones/aisles/slots
   from a structured list in one call, atomic per-row validation, partial
   success reported per row (this is the bootstrap mechanism for loading a
   real building's layout from a CSV/JSON export)
9. GetSiteLayout(siteCode) -> the full nested, drawable structure (read model)
10. GetZoneGrid(zoneRef) -> a 2D grid (aisle x bay x level) of that zone's
    slots, shaped for direct UI rendering (read model)

## REST API (inbound adapter) — this is the "draw the warehouse" capability

Structural / write side (builds up the model):

- POST   /sites                                    -> RegisterSite
- GET    /sites                                    -> list sites
- GET    /sites/{siteCode}                         -> get one site
- POST   /sites/{siteCode}/zones                   -> RegisterZone
- GET    /sites/{siteCode}/zones                   -> list a site's zones
- POST   /zones/{zoneId}/aisles                    -> RegisterAisle
- GET    /zones/{zoneId}/aisles                    -> list a zone's aisles
- POST   /location-types                           -> RegisterLocationType
- GET    /location-types                           -> list location types
- POST   /placement-rules                          -> DefinePlacementRule
- GET    /placement-rules                           -> list placement rules
- POST   /locations                                -> RegisterLocationSlot
- GET    /locations/{locationCode}                 -> get one slot
- GET    /locations/{locationCode}/classification  -> Zone Hazmat/TemperatureClass
  (denormalized read; Published Language for placement-rule consumers like
  inventory-storage — see ADR-0008)
- POST   /locations/{locationCode}/decommission    -> DecommissionLocationSlot
- POST   /locations/import                         -> ImportFacilityLayout
  (bulk; request body is a JSON array of rows, each row fully specifying a
  site/area/zone/aisle/bay/level/position + locationType — the WHOLE point
  of this endpoint is that a real warehouse's layout gets loaded here once,
  reproducibly, not typed in by hand slot-by-slot)

Read / "draw" side (this is the deliverable the user explicitly asked for —
REST APIs whose PURPOSE is to render the warehouse, not just CRUD):

- GET  /sites/{siteCode}/layout
  Returns the FULL nested structure for one site: zones -> aisles ->
  location slots, with every slot's coordinates, type, status. Shaped so a
  frontend can directly render a floor plan (group by zone, then aisle,
  then bay/level/position) without further client-side joining. This is
  the primary "draw the warehouse" endpoint.
- GET  /zones/{zoneId}/grid
  Returns ONE zone's slots as an explicit 2D grid: rows = Level, columns =
  (Aisle, Bay) pairs in aisle SequenceHint order, cell = slot status/type
  (or null for a gap). This is literally a matrix a UI can iterate over and
  paint as a warehouse map — no client-side layout math required.
- GET  /sites/{siteCode}/layout?format=svg
  STRETCH GOAL, do this if time allows after everything else is solid:
  same data as the JSON layout endpoint, rendered server-side as a minimal
  SVG floor plan (one colored rect per zone, one row of rects per aisle,
  color = zone temperature class / hazmat flag). Must be valid, viewable
  SVG (`<svg>...</svg>`, `Content-Type: image/svg+xml`) — a human should be
  able to `curl` this straight into a `.svg` file and open it in a browser.
  If you build this, keep it a thin adapter-only concern (render function in
  the HTTP layer only, no new domain code) — do NOT let SVG rendering leak
  into the domain or application layers. If you run low on turns, SKIP this
  and ship the JSON layout + grid endpoints solidly instead — they are the
  actual requirement; SVG is a nice-to-have.
- GET  /healthz

JSON DTOs live in the http adapter; never leak domain structs. Follow the
SAME REST maturity level (Richardson Level 2) and RFC 7807
(`application/problem+json`) error format the other four services already
use — do not reinvent it, replicate it exactly (resource nouns, correct
verbs/status codes, `Location` header on every 201, RFC 7807 problem
details on every error).

## Tech & standards (IDENTICAL stack and quality bar to the other four services)

- Go 1.26, modules. Module path: `github.com/claudioed/facility-layout`.
- chi (github.com/go-chi/chi/v5), pgx/v5 + pgxpool, golang-migrate SQL
  migrations.
- Config via env (`DATABASE_URL`, `HTTP_ADDR` defaulting to `:8080`).
  `docker-compose.yml` for Postgres 16, matching the other repos' exact
  shape (service name `postgres`, healthcheck, named volume).
- Typed domain errors mapped to HTTP status in the adapter; RFC 7807
  `application/problem+json` for every error response from day one (do NOT
  build the old bespoke `{"error":...}` shape and migrate later like the
  other four repos did historically — go straight to RFC 7807).
- Table-driven tests: domain + application (in-memory adapter); one
  httptest per endpoint; build-tagged Postgres integration test (skipped
  w/o `DATABASE_URL`).
- gofmt/go vet clean; every package has a doc comment.
- `.golangci.yml` — copy VERBATIM from `../inventory-storage/.golangci.yml`
  (same linters: errcheck, govet, staticcheck, unused, ineffassign,
  bodyclose, misspell, unconvert, gocritic; same test-file errcheck
  exclusion; gofmt+goimports formatters).

## Definition of done

- `go build ./...`, `go vet ./...`, `go test ./...` (and `-race`) all green.
- gofmt clean; `golangci-lint run ./...` zero issues (config copied from
  inventory-storage, see above).
- Unit test coverage >= 90% combined across `internal/domain/...` and
  `internal/application/...` (identical gate to the other four services).
- README.md: run steps (compose/migrate/go run), every endpoint with curl
  examples (including a full worked example: register a site, a zone, an
  aisle, a location type, a placement rule, a slot, then GET the layout and
  the grid and show the actual JSON), a layering note.
- These invariants each have a failing-path test: slot registration
  rejected when the Site/Zone/Aisle chain doesn't resolve or isn't Active;
  slot registration rejected when it violates a PlacementRule; duplicate
  LocationCode rejected; Zone/Aisle registration rejected against an
  unknown or Decommissioned parent.

## Local quality gate (run before every commit)

- **After making changes, run `make check`.** That is the fast
  self-correction loop: `fmt-check`, `vet`, `build`, `lint`, `test` — the same
  sensors CI runs, in well under a minute, with no database needed. Fix
  whatever it reports and re-run until it is green *before* you commit.
- **Before pushing, run `make check-all`** — `check` plus the 90% `coverage`
  gate, the arch-go `arch-test` fitness tests, and the `bdd` acceptance suite.
- `make vuln` runs `govulncheck ./...` (the supply-chain sensor, blocking in
  CI as the `vuln` job). Run it after touching `go.mod`/`go.sum`.
- `make mutation` runs gremlins over `./internal/domain` with the thresholds in
  `.gremlins.yaml` — the same command the blocking `mutation-fast` CI job runs.
  It takes a couple of minutes, so it is not part of `check`; run it when you
  change domain behaviour or domain tests.
- `make integration` and the slow scheduled `mutation` job are deliberately
  NOT in either bundle: the first needs `DATABASE_URL` and a live Postgres,
  the second takes too long for an edit loop.
- The lefthook git hooks enforce this automatically once someone has run
  `lefthook install` locally (pre-commit: `fmt-check` + `vet` + `lint`;
  pre-push: `make check`) — but run `make check` proactively rather than
  relying on the hook to catch you.

**Why this exists:** it keeps quality *left* (harness engineering) — every
sensor that used to fire only in CI, post-push, now fires locally so an agent
or a human can self-correct before the problem ever reaches a reviewer or the
pipeline.
