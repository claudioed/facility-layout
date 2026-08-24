---
id: 0008-location-classification-read-endpoint
title: 8. Location classification read endpoint
sidebar_label: 8. Location classification read endpoint
sidebar_position: 8
description: "GET /locations/{locationCode}/classification — a cheap, denormalized read of a LocationSlot's resolved hazmat/temperature-class attributes, so inventory-storage (and future consumers) can validate placement of classified products without duplicating Zone data."
---

# 8. Location classification read endpoint

## Status

**Accepted.**

## Context

`inventory-storage` is adding a `ProductClassification` concept at the SKU
level (`Hazmat`, `Fragile`, `TemperatureSensitive`, `Oversized`,
`HighValue` tags). At stow time it needs to validate that a classified SKU
is going into a legal location: a `Hazmat` SKU may only land in a
hazmat-rated Zone, and a `TemperatureSensitive` SKU may only land in a Zone
whose `TemperatureClass` matches the product's requirement.

`facility-layout` already carries every fact that check needs. `Zone` (see
[ADR 0002](./0002-hierarchical-location-code.md) and the [Aggregates &
invariants](../ddd/aggregates.md) doc) has always had a `Hazmat` flag and a
`TemperatureClass`. This is not new domain knowledge — it is data this
service has held since its first commit, and per CLAUDE.md's Strategic
classification section, this service is the declared **Open Host Service**
and **source of truth** for Zone attributes; every other context is a
**Conformist** that reads but never mutates them.

The forces:

- **The existing `LocationSlot` DTO deliberately does not carry these
  fields.** `GET /locations/{locationCode}` (`locationSlotResponse`)
  returns the slot's own shape — code, coordinates, type, capacity,
  status — not its parent Zone's attributes. Hazmat and TemperatureClass
  live on `Zone`, one level up the chain of custody, and adding them to the
  slot DTO would either duplicate zone state on every slot read (paid at
  the worst ratio — see [ADR 0003](./0003-placement-rules-at-registration-time.md)'s
  identical argument about reads vastly outnumbering writes) or blur the
  slot resource's own contract. Neither is wanted; the existing endpoint's
  shape is intentionally staying put.
- **A consumer that only needs the classification answer should not have
  to fetch and parse the whole Zone resource**, nor issue two calls
  (`GET /locations/{locationCode}` to learn the `zoneId`, then
  `GET /zones/{zoneId}` to learn its attributes) for a check this
  frequent. `inventory-storage` will call this on every stow of a
  classified SKU — the same "reads vastly outnumber writes, and every
  consumer evaluating the join independently is drift waiting to happen"
  reasoning ADR-0003 already established for PlacementRules applies here
  to a simpler question: not "is this placement legal by rule," just "what
  are this slot's zone attributes."
- **No new domain concept is warranted.** `Zone` already owns `Hazmat` and
  `TemperatureClass`; `LocationSlot` already resolves to exactly one `Zone`
  via its `LocationCode`. This is a join across two existing aggregates,
  read-only, expressible entirely as an application-layer composition of
  existing repository queries (`SlotRepo.FindByCode` then
  `ZoneRepo.FindByID`) — precisely the same `FindByCode` →
  `FindByID(code.ZoneID())` resolution `RegisterLocationSlot`'s
  chain-of-custody check already performs. Adding a new domain aggregate,
  a new write path, or new persisted state to answer this question would
  be manufacturing complexity this context does not need.
- **The consumer is cross-context.** `inventory-storage` is explicitly
  named in CLAUDE.md as a downstream Conformist to this service's Published
  Language. The natural home for "give a cross-context caller a cheap,
  stable answer to a placement question" is a new REST resource in this
  service's Open Host Service, not a client-side join `inventory-storage`
  would have to build and keep correct against this service's internal
  shape.

## Decision

**We will add `GET /locations/{locationCode}/classification`, a new,
read-only, denormalized endpoint that resolves a LocationSlot's
LocationCode to its parent Zone and returns exactly `{hazmat: bool,
temperatureClass: "Ambient"|"Chilled"|"Frozen"}` — no new domain aggregate,
no change to the existing `LocationSlot` DTO or any other endpoint's
contract, and this context remains the sole source of truth for these
attributes.**

### The application layer: composition, not duplication

A new use case, `GetLocationClassification`, wraps exactly the two
repository calls already used elsewhere:

```go
type GetLocationClassification struct {
    Slots ports.SlotRepo
    Zones ports.ZoneRepo
}

func (uc *GetLocationClassification) Execute(ctx context.Context, code shared.LocationCode) (*zone.Zone, error) {
    s, err := uc.Slots.FindByCode(ctx, code)   // ErrLocationSlotNotFound if nil
    ...
    z, err := uc.Zones.FindByID(ctx, code.ZoneID()) // ErrZoneNotFound if nil
    ...
    return z, nil
}
```

It returns the resolved `*zone.Zone` aggregate itself — the HTTP adapter
projects only the two fields the response needs. No new use case logic
duplicates `RegisterLocationSlot`'s chain-of-custody resolution; it reuses
the same two ports, and (because every Active slot in the database already
has a resolvable, Active zone by the invariant ADR-0003 enforces at
registration time) it does not need to re-walk the full Site → Zone →
Aisle chain, just the one hop from slot to its zone.

### The HTTP layer: a fourth GET on the same resource, same conventions

```
GET /locations/{locationCode}/classification -> 200 {"hazmat": bool, "temperatureClass": "..."}
```

sits alongside the existing `GET /locations/{locationCode}` under the same
`r.Route("/locations", ...)` block, follows the identical
handler → DTO → `writeError`/RFC 7807 pattern every other read endpoint in
this service uses (see [ADR 0004](./0004-rfc-7807-from-day-one.md)):

- A LocationCode that does not parse (not seven `[A-Z0-9]` segments) is
  `400 malformed-location-code` — it could never identify a slot, so it is
  never a 404, exactly as `GET /locations/{locationCode}` already
  behaves.
- An unknown slot is `404 location-slot-not-found`.
- The (should-be-unreachable, given the chain-of-custody invariant) case of
  a slot whose zone cannot be resolved is `404 zone-not-found` — reusing
  the existing error, not inventing a new one, because it is the same
  category of failure `RegisterLocationSlot` already names.

A new `locationClassificationResponse` DTO is added to `dto.go`, and a
`toLocationClassificationResponse` mapper to `mappers.go`, following the
exact shape of every existing response/mapper pair in this file. No
existing DTO changes.

### Why not the alternatives

| Option | Rejected because |
|---|---|
| **Add `hazmat`/`temperatureClass` to `locationSlotResponse`** | Blurs what the slot resource is about (its own structural identity, not its zone's), and pays the join cost on every plain slot read even when the caller does not need it. |
| **Make `inventory-storage` call `GET /locations/{locationCode}` then `GET /zones/{zoneId}`** | Two round trips for one question, asked on the hot stow path; every consumer re-implementing the same two-call join is exactly the duplication ADR-0003 already argued against for PlacementRules. |
| **A new `ClassificationLookup` domain aggregate or persisted projection** | Nothing new is being modeled — `Zone` already owns these attributes. A stored projection would need to be kept in sync with `Zone` and would be state this service does not need, for a question a stateless read answers correctly today. |
| **Do nothing; let `inventory-storage` duplicate a copy of Zone attributes** | The one thing this entire service exists to prevent — CLAUDE.md's Strategic classification section names this exact failure mode ("extract generic logic instead of duplicating it") as the reason `facility-layout` is its own service. |

## Consequences

### Easier

- **`inventory-storage`'s stow-time validation is one cheap GET.** No
  client-side join, no risk of drifting from this service's actual Zone
  data, no new state for `inventory-storage` to own and keep in sync.
- **This context's Open Host Service / Published Language claim (CLAUDE.md,
  ADR-0007) now has a second concrete, purpose-built read surface** beyond
  the "draw the warehouse" endpoints — one shaped around a placement
  *decision* rather than a structural render, matching the same
  intent-level design principle ADR-0007 already applied to the MCP tool
  surface.
- **No domain or aggregate change, so no invariant, migration, or event
  changes.** This is additive: one new use case struct, one new HTTP
  handler, one new DTO, one new route.
- **Reuses proven error categories.** `location-slot-not-found` and
  `zone-not-found` already exist and are already documented; nothing new
  is added to the RFC 7807 problem table.
- **Fully unit-tested with the in-memory adapters**, exactly like every
  other read use case in this service — no infrastructure required.

### Harder

- **A fourth way to read overlapping data about the same LocationCode**
  (`GET /locations/{locationCode}`, `GET /zones/{zoneId}`, `GET
  /sites/{siteCode}/layout`, and now this) exists in the API surface.
  Each has a distinct, narrow purpose, but a caller unfamiliar with the
  service now has one more endpoint to choose among; the OpenAPI
  description on this endpoint and the Locations tag exist specifically
  to steer that choice.
- **Two repository reads per call** (`SlotRepo.FindByCode` then
  `ZoneRepo.FindByID`) instead of one, though both are already
  indexed-by-identity lookups this service performs on every slot
  registration, so this is not a new access pattern, just a new place it
  is invoked from.
- **If `Zone`'s attribute set ever grows** (a new classification dimension
  beyond hazmat/temperature), this endpoint's response shape needs a
  deliberate decision about whether to expose it here too — it does not
  automatically inherit new Zone fields, by design, since it is a curated
  projection and not the whole aggregate.
