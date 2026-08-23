---
id: ubiquitous-language
title: Ubiquitous language
sidebar_label: Ubiquitous language
description: The exact vocabulary of the Facility Layout bounded context, and the words it deliberately does not use.
---

# Ubiquitous language

These are the exact names used in the code, the API, the events and this
documentation. Where a word is shared with another bounded context, the
overlap is called out explicitly — same word, different model is allowed in
DDD and expected here.

## Core terms

| Term | Meaning |
|---|---|
| **Site** | A physical facility/building. The root of the hierarchy. Has a `SiteCode` (non-empty, uppercase alphanumeric, unique) and a human name. |
| **Zone** | A behavioral classification scoped to a Site, bundling the Area and Zone code segments into one aggregate. Carries a `TemperatureClass` (Ambient/Chilled/Frozen) and a `Hazmat` flag. Zones are not cosmetic — every `PlacementRule` is keyed by one. |
| **Aisle** | A physical corridor scoped to a Zone. Carries a `SequenceHint` (its walk-order position — the concrete travel-distance input the WES tier needs) and a `Direction` (`OneWay`/`TwoWay`). |
| **LocationType** | A reusable classification of physical slot shape/kind — `PalletRack`, `Shelf`, `ToteWall`, `BulkFloor`, `Staging`, `Amnesty` — each carrying a default capacity envelope (max weight, max volume). |
| **LocationSlot** | The leaf aggregate: one coded physical slot. Its identity **is** its `LocationCode`. Has a LocationType, a capacity envelope (which may override the type's default), and a `Status`. |
| **PlacementRule** | A declaration of which LocationTypes are legal in which Zones. The mechanism that prevents "ambient product in the frozen zone" — enforced once, at registration time, not re-checked by every caller. |
| **LocationCode** | The coded address of a slot: seven typed, hyphen-joined segments, coarsest to finest. A value object, never free text. |
| **Facility layout** | The readable, drawable projection of the whole structure: a Site's Zones, each Zone's Aisles, each Aisle's LocationSlots, assembled into a shape a UI can render as a floor plan or grid. |

## Value objects and enumerations

| Term | Values / shape | Notes |
|---|---|---|
| **LocationCode** | `Site-Area-Zone-Aisle-Bay-Level-Position` | Each segment non-empty and `[A-Z0-9]` only. Always round-trips through `String()` / `ParseLocationCode()`. |
| **Capacity** | `maxWeightKg`, `maxVolumeM3` | Both must be strictly positive. |
| **TemperatureClass** | `Ambient`, `Chilled`, `Frozen` | A Zone attribute; a PlacementRule predicate can match on it. |
| **Direction** | `OneWay`, `TwoWay` | An Aisle attribute; an input to travel-path planning. |
| **Status** | `Active`, `UnderMaintenance`, `Decommissioned` | Shared by every structural aggregate. |
| **Effect** | `Allow`, `Deny` | What a PlacementRule does when its predicate matches. |
| **ZonePredicate** | any of `zoneCode`, `temperatureClass`, `hazmat` | Every set field must match (AND); unset fields are wildcards; at least one must be set. |
| **SequenceHint** | non-negative integer | An Aisle's walk-order position. |

## The code segments

| Segment | Meaning | Real examples |
|---|---|---|
| Site | the physical facility/building | `WH1` |
| Area | coarse functional area | `STOR` (storage), `RCV` (receiving), `PACK`, `STAGE` |
| Zone | behavioral class *within* an area — drives rules | `AMB` (ambient), `CHL` (chilled), `FRZ` (frozen), `HAZ` (hazmat), `FWD` (forward-pick), `RSV` (reserve) |
| Aisle | physical corridor | `A07` |
| Bay | a bay/section along the aisle | `03` |
| Level | vertical level/shelf | `02` |
| Position | left-to-right slot on that level | `B` |

## Lifecycle vocabulary

| Term | Meaning |
|---|---|
| **Active** | The structure exists and is legal for storage/traversal. |
| **UnderMaintenance** | The structure exists but is temporarily out of service. A legal persisted state — e.g. loaded from an external facility-management system — that the read models render. v1 exposes no use case that *sets* it, but a slot in it can still be decommissioned. |
| **Decommissioned** | Permanently retired. **One-way** in v1: there is no reactivation use case, and re-registering a decommissioned LocationCode is rejected as a duplicate rather than quietly resurrecting the slot. See [ADR 0005](../adr/0005-one-way-decommission.md). |
| **Chain of custody** | The Site → Zone → Aisle resolution performed before a LocationSlot is allowed to exist. Registering a slot is a chain-of-custody check, not a bare insert. |
| **Amnesty** | A real term from the domain reference: where a damaged or mismatched item is set aside during stow. Modelled here as a `LocationType`, because an amnesty position is a physical slot with a shape and a capacity like any other. |

## Words this context shares with others

| Word | Here it means | Elsewhere it means |
|---|---|---|
| **Zone** | A behavioral classification of physical space (temperature class + hazmat) scoped to a Site. This service is its **source of truth**. | In `wes-work-planning` and the WES ubiquitous language generally, `Zone` is a unit of congestion and travel-path reasoning. Same physical thing, consumed as a read-only fact. |
| **Location** | A coded slot's *structural identity and legality*. Nothing about contents. | In `inventory-storage`, a `Bin`/`Location` is about capacity **occupancy** and what stock sits there. |
| **Capacity** | The static envelope of a slot's shape: max weight, max volume. | In `inventory-storage`, capacity is dynamic and consumed — how much room is left right now. |

## Words this context deliberately does **not** use

`StockUnit`, `Reservation`, `Usable inventory`, `Task`, `Assignment`, `Wave`,
`Pick`, `Pack`, `SLAM`, `Associate`, `Shift`.

None of those are physical structure. If a term from that list ever appears
in this service's domain layer, the boundary has leaked and the service has
started to duplicate a neighbour.
