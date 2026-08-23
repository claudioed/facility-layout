---
id: domain-events
title: Domain events
sidebar_label: Domain events
description: The eight past-tense facts this context publishes, their CloudEvents type convention, and their payload shapes.
---

# Domain events

This bounded context emits eight past-tense domain events. Together they are
its **Published Language** — the vocabulary downstream Conformists will key
off once integration is wired.

:::info[No AsyncAPI specification, and no broker — yet]
This service has `apis/openapi.yaml` but **no** `apis/asyncapi.yaml`, because
it does not publish integration events to a broker. The four sibling services
publish over Kafka and each ship an AsyncAPI 2.6.0 document; this one does
not, and this documentation site therefore has **no Events page under API
Reference** — this page documents the in-process domain events instead.

Today, events are handed to an `EventPublisher` port with two
implementations: a **log publisher** and a **buffered publisher** (used by
tests). When running against Postgres they are also appended to an `events`
table. The port's signature is deliberately the shape a Kafka producer would
satisfy, so adding an outbound Kafka adapter later is additive. See
[Context map](../ecosystem/context-map.md).
:::

## The type convention

Identical to the other four warehouse-systems services: reverse-DNS,
lowercase except the final PascalCase event name, and the entity segment
carries no hyphen even for multi-word aggregate names.

```
com.warehouse.<subdomain>.<bounded-context>.<entity>.<EventName>
```

This service's **subdomain segment is `wms`**. "Bin-accurate location" is
classified WMS-tier in the Amazon-fulfillment reference — the "Inventory &
Slotting" Core subdomain references it as WMS's Open Host Service — and this
service is the generalized, multi-consumer version of that same concern.

```
com.warehouse.wms.facility-layout.site.SiteRegistered
com.warehouse.wms.facility-layout.zone.ZoneRegistered
com.warehouse.wms.facility-layout.aisle.AisleRegistered
com.warehouse.wms.facility-layout.locationtype.LocationTypeRegistered
com.warehouse.wms.facility-layout.placementrule.PlacementRuleDefined
com.warehouse.wms.facility-layout.locationslot.LocationSlotRegistered
com.warehouse.wms.facility-layout.locationslot.LocationSlotDecommissioned
com.warehouse.wms.facility-layout.locationslot.FacilityLayoutImported
```

Entity segments in use: `site`, `zone`, `aisle`, `locationtype`,
`placementrule`, `locationslot`.

## The DomainEvent interface

```go
// DomainEvent is a past-tense fact published by an aggregate. Adapters
// (outbound/events) serialize and publish these; the domain never depends
// on the publishing mechanism. EventType is this context's Published
// Language: downstream Conformists key off it.
type DomainEvent interface {
	EventName() string
	EventType() string
	OccurredAt() time.Time
}
```

Every event embeds the same base, serialized as:

```json
{
  "eventName": "LocationSlotRegistered",
  "eventType": "com.warehouse.wms.facility-layout.locationslot.LocationSlotRegistered",
  "occurredAt": "2026-08-22T09:00:00Z"
}
```

`OccurredAt` comes from the injected `Clock` port, never from
`time.Now()` inside the domain — which is what makes event timestamps
deterministic in tests.

## The eight events

### SiteRegistered

A physical facility was added to the warehouse map.

| Field | Type |
|---|---|
| `siteCode` | string |
| `siteName` | string |

### ZoneRegistered

A behavioral zone was added inside a Site's area.

| Field | Type |
|---|---|
| `zoneId` | string (`WH1-STOR-AMB`) |
| `siteCode` | string |
| `areaCode` | string |
| `zoneCode` | string |
| `temperatureClass` | `Ambient` \| `Chilled` \| `Frozen` |
| `hazmat` | boolean |

### AisleRegistered

A physical corridor was added inside a Zone.

| Field | Type |
|---|---|
| `aisleId` | string (`WH1-STOR-AMB-A07`) |
| `zoneId` | string |
| `aisleCode` | string |
| `sequenceHint` | integer — walk-order position |
| `direction` | `OneWay` \| `TwoWay` |

This is the event a travel-path consumer cares about most: `sequenceHint`
and `direction` are the two structural inputs to walk-order reasoning.

### LocationTypeRegistered

A reusable slot shape/kind was defined.

| Field | Type |
|---|---|
| `locationType` | string (`PalletRack`) |
| `maxWeightKg` | number |
| `maxVolumeM3` | number |

### PlacementRuleDefined

A rule constraining which LocationTypes are legal in which Zones was
declared.

| Field | Type |
|---|---|
| `ruleId` | string |
| `locationType` | string |
| `effect` | `Allow` \| `Deny` |
| `predicate` | string — the human-readable predicate, e.g. `temperatureClass=Frozen` |

### LocationSlotRegistered

A coded leaf slot now exists on the warehouse map. This is the event
`inventory-storage` would consume to learn that a new `Bin` location is legal
to stow into.

| Field | Type |
|---|---|
| `locationCode` | string (`WH1-STOR-AMB-A07-03-02-B`) |
| `aisleId` | string |
| `zoneId` | string |
| `locationType` | string |
| `maxWeightKg` | number |
| `maxVolumeM3` | number |

`aisleId` and `zoneId` are denormalised into the payload deliberately: a
consumer should not have to know how to parse this context's code format to
route on zone. That is what makes it a *Published Language* rather than a
leaked internal representation.

### LocationSlotDecommissioned

A coded slot was permanently retired. The signal for a downstream context to
stop offering that location for new work.

| Field | Type |
|---|---|
| `locationCode` | string |

### FacilityLayoutImported

A bulk layout import completed. Emitted **once per import call**; the
individual `LocationSlotRegistered` events still fire per-slot within the
same import.

| Field | Type |
|---|---|
| `rowsSubmitted` | integer |
| `slotsImported` | integer |
| `rowsRejected` | integer |

The summary event exists so that a consumer can distinguish "the building was
loaded" from a burst of unrelated single registrations, without having to
infer it from event volume.

## Which use case emits what

| Use case | Events emitted |
|---|---|
| `RegisterSite` | `SiteRegistered` |
| `RegisterZone` | `ZoneRegistered` |
| `RegisterAisle` | `AisleRegistered` |
| `RegisterLocationType` | `LocationTypeRegistered` |
| `DefinePlacementRule` | `PlacementRuleDefined` |
| `RegisterLocationSlot` | `LocationSlotRegistered` |
| `DecommissionLocationSlot` | `LocationSlotDecommissioned` |
| `ImportFacilityLayout` | `LocationSlotRegistered` per successful row, plus one `FacilityLayoutImported` |
| `GetSiteLayout`, `GetZoneGrid` | **none** — read models never write and never publish |

## Publishers

| Adapter | Use |
|---|---|
| `outbound/events` — log publisher | Default. Writes each event to the service log. |
| `outbound/events` — buffered publisher | Tests. Collects events in memory for assertion. |
| `outbound/postgres` — event publisher | Appends to the `events` table (`event_name`, `event_type`, `occurred_at`, `payload JSONB`) when running against Postgres. |

There is no Kafka adapter in this repository. The `EventPublisher` port is
one method —
`Publish(ctx context.Context, event shared.DomainEvent) error` — chosen so
that adding one later is a new adapter and nothing else.
