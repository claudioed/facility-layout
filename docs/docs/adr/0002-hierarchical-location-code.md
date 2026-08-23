---
id: 0002-hierarchical-location-code
title: 0002 — Industry-standard hierarchical location code over a flat identifier
sidebar_label: 0002 · Hierarchical location code
description: Why a slot's identity is Site-Area-Zone-Aisle-Bay-Level-Position rather than a UUID or a free-text bin string.
---

# 0002 — Industry-standard hierarchical location code over a flat identifier

**Status:** Accepted

## Context

This service must give every physical storage slot in a building an identity.
That identity is the thing four other bounded contexts will reference, that
appears on the label stuck to the rack, and that an associate reads out loud
during a stow. It is the single most consequential decision in the service.

The forces:

- **Physical and digital identity must match.** The Amazon-fulfillment
  reference is explicit that a stow requires both an item scan and a
  *location* scan, and that placing an item without recording the location is
  precisely how inventory becomes "lost." Whatever identifies a slot in the
  database has to be the same thing that is physically readable on the rack.
- **Consumers need the hierarchy, constantly.** "Which zone is this in?"
  gates PlacementRule evaluation. "Which aisle?" gates travel-path reasoning.
  "What is the walk order?" gates pick-path optimisation. These questions are
  asked far more often than slots are created.
- **There is an established industry answer.** WMS products converge on
  **Site → Area → Zone → Aisle → Bay → Level → Position**, hyphen-joined and
  human-parsable. It is not a novel problem.
- **This is a Generic Subdomain.** There is no competitive advantage in
  identifying locations differently from the rest of the industry, and real
  cost in doing so: operators, labels, printers and imported CSVs all already
  speak this format.
- **Humans are in the loop.** An operator reads the code off a rack, says it
  over a radio, and types it into a handheld. A format that is hostile to
  that is hostile to the actual job.

Three options were on the table:

1. **Flat UUID** as the slot identity, with site/zone/aisle as foreign keys.
2. **Free-text bin string**, whatever the customer's existing labels say.
3. **Structured hierarchical code** as the identity.

## Decision

We will make a slot's identity an **industry-standard, seven-segment
hierarchical location code**, modelled as a **value object**:

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

Specifically:

- `LocationCode` is a struct of seven unexported string fields, constructed
  only through `NewLocationCode(...)` or `ParseLocationCode(...)`, both of
  which reject any segment that is empty or contains a character outside
  `[A-Z0-9]`. It always round-trips through `String()`.
- **The code IS the LocationSlot's identity** — not a natural key alongside a
  surrogate one. It is the primary key in `location_slots`.
- Parent identities are **derived** from the code with no lookup:
  `ZoneID()` = `Site-Area-Zone`, `AisleID()` = `ZoneID-Aisle`.
- Lowercase input is **rejected, not normalised**. Two spellings of one
  physical slot must not both be acceptable when the code is an identity.
- The database stores **both** representations: `code` as the primary key,
  and each of the seven segments as its own column, so the zone-grid read
  model can filter and order by aisle/bay/level in SQL rather than parsing
  strings at read time.

### Why not the alternatives

| Option | Rejected because |
|---|---|
| **Flat UUID** | Every question the hierarchy answers for free becomes a join. Not speakable over a radio, not readable on a label, not sortable into a pick path, and impossible to sanity-check by eye during a bulk import. Requires a parallel human-readable label anyway — at which point there are two identities that can disagree. |
| **Free-text bin string** | No validation surface at all. Guarantees eventual drift between the label on the rack and the string in the database, and gives PlacementRule evaluation nothing structured to match on. |
| **Composite key, no single code** | Works in SQL, but every API path, log line, error message and operator conversation then needs seven fields instead of one token. |

## Consequences

### Easier

- A bare location string is **self-describing**. `WH1-STOR-AMB-A07-03-02-B`
  tells you the site, the area, the behavioural zone, the corridor, and the
  exact position, with no lookup and no context.
- The chain-of-custody check is cheap and cannot disagree with itself: the
  code states which Site, Zone and Aisle must exist and be Active. Nothing is
  passed alongside it that could contradict it.
- Both read models are assembled without a join table — `GetZoneGrid` groups
  by `(aisle, bay)` and `level` straight out of the code's own segments.
- Sorting by code approximates walk order, and lexicographic ordering within
  an aisle gives a usable rack elevation for free.
- Bulk import from a real WMS export needs no identifier mapping: the codes
  in the file *are* the identities.
- Validation errors are specific and actionable —
  `position segment "b"` rather than "invalid id."

### Harder

- **A code changes when the building is re-organised.** Move a zone and every
  slot beneath it gets a new identity. This is the real cost, and it is
  accepted: re-slotting is a rare, planned, physically-signposted event,
  whereas reading and reasoning about locations happens constantly. Optimise
  for the common case.
- **The identity carries meaning, which is normally an anti-pattern.** It is
  accepted here because the meaning is physical and externally visible — the
  code is printed on the rack — so the coupling exists in the real world
  whether or not the database acknowledges it.
- Segment vocabularies (`STOR`, `AMB`, `A07`) are conventions, not enums. The
  service validates the character set and the arity, not the meaning. A typo
  that is still `[A-Z0-9]` will be accepted as a new zone — which is why
  registration goes through the chain-of-custody check rather than trusting
  the code alone.
- Every consumer will be tempted to `split("-")` the code and own a copy of
  this format forever. Mitigated by denormalising `zoneId` and `aisleId` into
  the `LocationSlotRegistered` event payload and into every slot response, so
  consumers never need to parse. See
  [Consuming this service](../ecosystem/consuming-this-service.md).
- Storing both the code and its seven segments is deliberate duplication in
  the schema, kept consistent by the fact that only one constructor can
  produce either.
