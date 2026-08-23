---
id: consuming-this-service
title: Consuming this service
sidebar_label: Consuming this service
description: How a downstream context would integrate today (synchronous REST) and how it is designed to integrate later (events) — with the current status stated plainly.
---

# Consuming this service

:::warning[Current status]
No service consumes `facility-layout` today. There is no Kafka adapter, no
published topic, and no AsyncAPI document in this repository. This page
describes how consumption is **designed** to work, so that the intended shape
is written down before anybody builds it — not a description of running
integration.
:::

## Two integration styles, by need

| Need | Style | Available today |
|---|---|---|
| "Is this exact location valid, right now, before I accept this stow?" | **Synchronous REST** — `GET /locations/{locationCode}` | ✅ The endpoint exists and works |
| "Keep a local read model of the building's structure in sync" | **Event subscription** | ❌ Requires a broker adapter that does not exist |

The synchronous path is usable immediately by anything that can make an HTTP
call. The event path is the designed steady state and is unbuilt.

## Synchronous validation

The narrowest useful integration, and the one that needs nothing new:

```bash
curl -i localhost:8080/locations/WH1-STOR-AMB-A07-03-02-B
```

- `200` with `"status": "Active"` → the location is real and legal to use.
- `200` with `"status": "Decommissioned"` or `"UnderMaintenance"` → it exists
  but must not be used for new work.
- `404` → it is not on the map. Whatever produced that code is wrong.
- `400` → the code is not seven valid `[A-Z0-9]` segments, so it was never a
  location at all.

The important property: because
[PlacementRules are enforced at registration time](../adr/0003-placement-rules-at-registration-time.md),
a consumer that reads back an `Active` slot can treat it as
**already-validated**. It does not need to re-check temperature class,
hazmat, or capacity legality — that check already happened, once, before the
slot was allowed to exist. The consumer's job is a single existence-and-status
read, not a rules evaluation.

For structure rather than a single slot:

| Consumer question | Endpoint |
|---|---|
| What zones does this site have? | `GET /sites/{siteCode}/zones` |
| What aisles, in walk order? | `GET /zones/{zoneId}/aisles` — ordered by `sequenceHint` |
| The whole building at once | `GET /sites/{siteCode}/layout` |

## Event subscription (designed, not built)

The intended steady state is that consumers keep their own read model in
sync from this context's [domain events](../ddd/domain-events.md), rather
than making a synchronous call per validation.

The events a consumer would care about:

| Event | Consumer reaction |
|---|---|
| `LocationSlotRegistered` | Add the location to the local map as usable. Payload carries `zoneId` and `aisleId` already denormalised, so no code parsing is needed. |
| `LocationSlotDecommissioned` | Stop offering that location for new work. |
| `ZoneRegistered` | Learn a new zone, its temperature class and hazmat flag. |
| `AisleRegistered` | Learn a corridor, its `sequenceHint` walk-order position and its `direction`. This is the travel-path input. |
| `FacilityLayoutImported` | Recognise a bulk load — useful for distinguishing a building bootstrap from ordinary drift. |

Building it would mean, in this repository: an
`internal/adapters/outbound/kafka` package implementing
`ports.EventPublisher`, an `EVENT_PUBLISHER=kafka|log` switch, a
`warehouse.facility-layout.events` topic, and an `apis/asyncapi.yaml` — all
matching the pattern the four sibling services already use. None of it
exists.

## Rules for a consumer

These apply to either style, and follow from this context being an Open Host
Service its consumers are Conformist to.

**Never write.** No other context gets write access to a Site, Zone, Aisle,
LocationType, PlacementRule or LocationSlot. The map has exactly one author.

**Never re-derive the hierarchy by string surgery.** A `LocationCode` looks
easy to `split("-")`, and every consumer that does it owns a copy of this
context's format forever. Use the `zoneId` and `aisleId` the API and the
events already give you.

**Never re-implement PlacementRules.** Duplicating the rule engine in a
consumer recreates the exact problem the rule engine exists to prevent: two
implementations that disagree the first time a rule changes.

**Translate at your edge.** Conformist does not mean "use `LocationSlot` as
your internal type." `inventory-storage` should keep its own `Bin` aggregate
and populate it from this context's facts; the two models have different
lifecycles and different reasons to change.

**Expect one-way decommission.** A retired code never comes back. A consumer
never has to handle a `Decommissioned → Active` transition, and should not
build one. See [ADR 0005](../adr/0005-one-way-decommission.md).

## What this service will never do

So that no consumer waits for it:

- It will not track occupancy, stock, reservations or usable inventory.
- It will not know about Tasks, Assignments, Waves, Shifts or Associates.
- It will not compute a travel path. It supplies the structural inputs —
  `sequenceHint`, `direction`, zone adjacency — and the WES tier does the
  optimisation, because that optimisation *is* the WES tier's core domain.
- It will not call another warehouse-systems service. Ever.
