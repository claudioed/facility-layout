---
id: 0016-functional-location-roles
title: 16. Functional location roles beyond storage
sidebar_label: 16. Functional location roles
sidebar_position: 16
description: "Every LocationType in v1 implicitly means storage. Real WMS/WES estates need the map to also say a coded location is a dock door, a yard slot, a work center, a drop zone, a QC bay, a consolidation point, or a shipping location — not just a place stock sits. This adds a LocationRole to LocationType, with role-specific attributes on LocationSlot, additive to the existing event and API contracts."
---

# 16. Functional location roles beyond storage

## Status

**Proposed.**

## Context

Every aggregate this service exposes today — `Site`, `Zone`, `Aisle`,
`LocationSlot` — models a place where stock is stored or a structure that
contains such places. `LocationType` (`PalletRack`, `Shelf`, `ToteWall`,
`BulkFloor`, `Staging`, `Amnesty`) is an open tag set, but every existing
value answers the same question: *what shape of storage is this?* There is
no way to say a coded location is not storage at all.

This gap is already visible at the fleet's boundary. `fulfillment-execution`'s
`Station` aggregate is `{id, capabilities, occupant}` — it has no physical
location, so nothing connects "pack station 3" to a place on this service's
map. `process-path-management`'s catalogue can route to a path id like
`pick` or `pack`, but not to a coded destination. `inventory-storage`
receives stock at an implicit "somewhere" with no dock concept. This
service's own `AGENTS.md` says `wes-work-planning` and `fulfillment-execution`
need this map for travel-path and congestion reasoning, but the map has
nothing for them to reason about outside of storage aisles.

Three real WMS/WES products were checked for how they handle this, because
this is a well-understood, Generic-subdomain concern that should not be
reinvented from scratch:

- **Oracle WMS Cloud**'s Location Master documents these Location Types:
  `Active`, `Reserve`, `Dock` ("used to assign to loads during
  receiving/shipping"), `Yard` ("used to locate trailer"), `Packing
  Station`, `Pack and Hold`, `Drop` ("used to hold both inbound and
  outbound LPNs while in between warehouse processes... you can configure
  the picking task to target a specific drop zone"), `QC`, `Receiving
  Station`, `VAS`, `Consolidation`, `Staging Location`, `Shipping Location`.
- **SAP EWM** assigns every storage type a **Storage Type Role**: `Standard
  Storage Type`, `Identification Point`, `Pick Point`, `Staging Area Group`,
  `Work Center` ("a physical area within the warehouse where processes such
  as deconsolidation, inspection, packing or value added service processing
  take place"), `Doors`, `Yard`, `Automatic Storage Retrieval`, `Production
  Supply`.
- **Blue Yonder** and **Manhattan Active WM**'s marketing both describe
  "yard" and "dock" as first-class parts of the same unified inventory view
  as reserve/active storage ("Unify inventory, labor, automation, and
  yard"; a single view "from inbound to DC, on yard, reserve storage,
  active pick sites, automated storage").

Both Oracle and SAP attach the functional distinction to the location's
*type*, not to the individual location, and both keep it inside the same
structural hierarchy (Oracle: Facility/Area/Aisle/Bay/Level/Position for
every type; SAP: storage type + storage bin for every role including
Doors and Yard). Neither introduces a parallel identity scheme for
non-storage locations.

## Decision

Add a `LocationRole` to `LocationType`, mirroring the shape of the
functional taxonomy above without importing either vendor's exact term
set:

```
Storage (default) | Dock | Yard | WorkCenter | Drop | Staging | QC |
Consolidation | Shipping
```

`Storage` is the default for every existing and newly-registered
`LocationType` that doesn't specify otherwise — this is a pure addition,
not a reclassification; every row registered before this ADR keeps working
unchanged.

`LocationSlot` gains typed, role-conditional attributes rather than a
generic key-value bag:

- `DockFlow` (`Inbound | Outbound | Both`) is **required** when the slot's
  role is `Dock`, and rejected (`ErrFunctionalAttributesNotAllowed`) on any
  other role.
- `WorkCenterActivities` (a non-empty set of `Pack | Sort | QC | VAS |
  Deconsolidate | Receive | Kit`) is **required** when the slot's role is
  `WorkCenter`, and rejected on any other role.

Capacity (`shared.Capacity`, today unconditionally required and strictly
positive) becomes **role-conditional**: `Dock`, `Yard`, `WorkCenter`, `QC`,
and `Shipping` locations may register with no capacity envelope, because a
door or a work center does not hold a weight/volume of stock the way a
pallet rack slot does. `Storage`, `Staging`, `Drop`, and `Consolidation`
keep the existing mandatory, strictly-positive capacity — those roles hold
inventory in transit or at rest, exactly like storage does today.

The existing 7-segment `LocationCode` (`Site-Area-Zone-Aisle-Bay-Level-
Position`, ADR-0002) is reused for functional locations too, e.g. an
outbound dock door might be `WH1-DOCK-OB-D-07-00-A`. This is deliberately
the less elegant of two options considered (see Consequences) but it keeps
every existing consumer's `LocationCode` parser, and this service's own
chain-of-custody invariant (Site → Zone → Aisle → Slot, ADR's core
invariant), completely untouched.

Placement of functional `LocationType`s into zones needs no new mechanism:
the existing `PlacementRule` (Allow/Deny a `LocationType` by zone
predicate, enforced once at registration time per ADR-0003) already
expresses "only `Dock` types may register in a `DOCK`-area zone" exactly as
it expresses "no `PalletRack` in a `HAZ` zone with a matching Deny rule".

New reads: `GET /sites/{siteCode}/locations?role={role}` and an MCP
`list_functional_locations(siteCode, role)` tool, for a caller that wants
"where are this site's dock doors" without walking the full site layout.
`GetSiteLayout`/`GetZoneGrid` responses gain a `role` field per slot, and
the SVG renderer gains a distinct glyph per role.

Event contract: `LocationTypeRegistered` gains `role`; `LocationSlotRegistered`
gains `role`, and `dockFlow`/`activities` when applicable (omitted
otherwise). Both are additive fields on existing, already-live event
types — ADR-0013 already makes this service's Published Language a binding
compatibility contract for `inventory-storage`'s `facilitycache` consumer,
which matches on the trailing CloudEvents type name (`...ZoneRegistered`,
etc.) and does not read capacity, so it is unaffected by either the new
fields or the now-optional capacity.

## Consequences

### Easier

- Fleet services finally have a real location to point at: a `Station` in
  `fulfillment-execution`, a receiving dock in `inventory-storage`, a path
  destination in `process-path-management` can all reference a
  `LocationCode` whose role this service now states authoritatively,
  instead of an implicit or free-text "somewhere".
- `PlacementRule` needed no new mechanism — the existing zone-predicate
  rule engine already covers functional-location placement, which is a
  sign the ADR-0003 design generalized further than it was originally
  asked to.
- The console (`facility-mfe`) gets a real reason to render more than a
  storage grid: a dock, a work center, a yard slot are now distinguishable
  facts, not lookalike `PalletRack` rows.

### Harder / accepted

- **The 7-segment code is awkward for non-storage locations.** A dock door
  numbered `D-07` does not have a meaningful "level" or "position" the way
  a pallet slot does; those segments are populated with placeholder values
  (`00`, a letter suffix) purely to satisfy the existing parser. The
  alternative — a second, shorter code family for functional locations —
  was rejected because it would fork `LocationCode` (ADR-0002) into two
  incompatible identity schemes and break every existing consumer's
  parsing assumption that a `LocationCode` always has seven segments. If
  the console UX proves this genuinely unworkable, that tradeoff should be
  revisited as its own ADR, not patched around silently.
- **Capacity's invariant is now conditional, not universal.** Every
  caller and every future contributor to `internal/domain/slot` must
  remember that a zero `Capacity` is legal for five of nine roles and
  illegal for the other four, rather than "capacity is always required".
  This is enforced in the domain layer (`LocationRole.RequiresCapacity()`)
  precisely so it cannot be forgotten at a single call site.
- **Downstream Conformists must decide whether they care about the new
  fields.** They are additive and safely ignorable, but a consumer that
  wants to act on "is this a dock" (e.g. `inventory-storage` deciding
  receiving logic) has new integration work to do; this ADR does not do
  that work, it only makes the fact available.
- Bulk import (`ImportFacilityLayout`, ADR-0006) grows two more optional
  columns (`dockFlow`, `activities`); existing import files with neither
  column keep working (both default to empty/Storage).

## Related

- ADR-0002 — Industry-standard hierarchical location code (the code shape
  this ADR reuses rather than forking).
- ADR-0003 — PlacementRules enforced once at registration time (the
  mechanism this ADR relies on unchanged for functional-location zoning).
- ADR-0006 — Bulk import reports partial success per row (the import shape
  this ADR extends with two optional columns).
- ADR-0013 — inventory-storage is the first real consumer of the Published
  Language (the compatibility contract this ADR's new event fields must
  remain additive to).
- ADR-0017 — Geometry and the travel graph (the companion ADR; functional
  locations are where geometry and travel distance become useful to
  `wes-work-planning`).
