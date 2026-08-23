---
id: context-map
title: Context map
sidebar_label: Context map
description: Where facility-layout sits among the five warehouse-systems services — what is actually wired today, and what is only strategically decided.
---

# Context map

:::warning[Read this before the diagrams]
`facility-layout` currently has **zero live integration** with any of the
other four warehouse-systems services. It publishes no Kafka events, consumes
no Kafka events, and has no consumer calling its REST API in production. Its
`EventPublisher` port is backed only by an in-process log publisher, a
buffered test publisher, and a Postgres `events` table.

Everything on this page marked *planned* is a **strategic design decision
that has not been built**. It is documented here because the decision is
real and shapes the service's API; it is flagged everywhere so nobody reads
an intention as an implementation.
:::

## The five services

| Service | Tier | Subdomain | This service's relationship to it |
|---|---|---|---|
| `inventory-storage` | WMS | Core | **Planned consumer** — location validity for `Bin`/`StockUnit` |
| `wes-work-planning` | WES | Core | **Planned consumer** — Zone/Aisle travel-path and congestion input |
| `fulfillment-execution` | — | Core | **Planned consumer** — Zone/Aisle input into task dispatch |
| `workforce-management` | — | Supporting | No planned relationship |
| **`facility-layout`** | WMS-tier concern, extracted | **Generic** | Open Host Service to all of the above |

## What is actually wired today

The four sibling services do integrate with each other over Kafka, on a
shared broker (`~/warehouse-systems/docker-compose.kafka.yml`). That live
topology looks like this — and `facility-layout` is deliberately absent from
it:

```mermaid
graph LR
    IS["inventory-storage<br/><i>WMS · Core</i>"]
    WM["workforce-management<br/><i>Supporting</i>"]
    WP["wes-work-planning<br/><i>WES · Core</i>"]
    FE["fulfillment-execution<br/><i>Core</i>"]
    FL["facility-layout<br/><i>Generic</i><br/><b>no broker integration</b>"]

    IS -->|"warehouse.inventory.events<br/>StockReserved"| WP
    WM -->|"warehouse.workforce.events<br/>ShiftPlanCommitted"| WP
    WP -->|"warehouse.work-planning.events<br/>WorkReleased"| FE
    FE -->|"warehouse.fulfillment.events<br/>TaskCompleted"| WP

    style FL fill:#fff3cd,stroke:#b45309,stroke-width:2px,stroke-dasharray: 6 4
```

`facility-layout` has **no edges** in that diagram. It has no
`internal/adapters/outbound/kafka` package, no `KAFKA_BROKERS` environment
variable, and no `apis/asyncapi.yaml`. That absence is the accurate current
state, not an omission from this page.

## What is planned, and not yet built

The strategic design is settled: this service becomes the **Open Host
Service** for physical-location truth, and the three consumers below become
**Conformists** to its Published Language.

```mermaid
graph TD
    FL["<b>facility-layout</b><br/><i>Generic Subdomain</i><br/>Open Host Service + Published Language<br/>Site · Zone · Aisle · LocationSlot · PlacementRule"]

    IS["<b>inventory-storage</b><br/><i>WMS · Core</i><br/>Bin / StockUnit"]
    WP["<b>wes-work-planning</b><br/><i>WES · Core</i><br/>release &amp; flow balancing"]
    FE["<b>fulfillment-execution</b><br/><i>Core</i><br/>Pick / Pack / SLAM tasks"]
    WM["<b>workforce-management</b><br/><i>Supporting</i>"]

    FL -. "PLANNED — not yet wired<br/>location validity for a stow<br/>(Conformist)" .-> IS
    FL -. "PLANNED — not yet wired<br/>Zone + Aisle SequenceHint<br/>as travel-path / congestion input<br/>(Conformist)" .-> WP
    FL -. "PLANNED — not yet wired<br/>Zone / Aisle context for dispatch<br/>(Conformist)" .-> FE
    FL -. "no planned relationship" .-x WM

    style FL fill:#e0f0ff,stroke:#0b69a3,stroke-width:3px
    style IS stroke-dasharray: 5 5
    style WP stroke-dasharray: 5 5
    style FE stroke-dasharray: 5 5
    style WM fill:#f5f5f5,stroke:#999,stroke-dasharray: 2 4
```

**Every edge in that diagram is dashed and labelled "PLANNED".** None of them
exist in code today, in this repository or in the other four.

### The three planned consumer relationships

| Consumer | What it would consume | Why it needs it |
|---|---|---|
| `inventory-storage` | `LocationSlotRegistered`, `LocationSlotDecommissioned`, or a synchronous `GET /locations/{code}` | A chaotic-storage stow is only valid against a location that exists and is Active. Today `inventory-storage` owns its own `Bin` identity with no external validation; validating it against this service would remove the possibility of stowing into a location that is not on the map. |
| `wes-work-planning` | `ZoneRegistered`, `AisleRegistered` | The WES ubiquitous language already contains `Zone`, `Travel Path` and `Congestion`, but nothing in the platform was the source of truth for the physical facts behind them. An Aisle's `SequenceHint` and `Direction` are the concrete travel-distance inputs that were previously missing. |
| `fulfillment-execution` | `ZoneRegistered`, `AisleRegistered` | Same physical facts, used at dispatch granularity rather than release granularity. |

`workforce-management` stops at the process-path boundary and never links an
associate to a specific location, so it has no reason to consume this
service.

## The strategic relationship, in context-mapping vocabulary

Using Evans/Vernon's patterns as the platform's DDD reference does:

### Open Host Service + Published Language

This service defines an **Open Host Service**: a stable, general-purpose
protocol any number of consumers may use, rather than a bilateral contract
negotiated per consumer. Its **Published Language** is the eight past-tense
[domain events](../ddd/domain-events.md) plus the REST surface, expressed in
its own vocabulary (`LocationCode`, `Zone`, `Aisle`, `LocationType`,
`PlacementRule`).

Two design choices exist to make that language *publishable*:

- `LocationSlotRegistered` denormalises `zoneId` and `aisleId` into the
  payload. A consumer routing on zone must not have to know how to parse this
  context's code format. A Published Language that requires the consumer to
  reimplement the producer's parsing is a leaked internal representation.
- The `EventPublisher` port is a single method,
  `Publish(ctx, event) error` — deliberately the shape a Kafka producer
  satisfies, so adding a broker adapter later is purely additive.

### Conformist, downstream

The consumers are **Conformists**: they accept this context's model rather
than negotiating a shared one, and translate it into their own vocabulary at
their edge. That is the right pattern here precisely *because* this is a
Generic Subdomain — there is nothing to differentiate by modelling location
differently, so conforming costs a consumer nothing and saves everyone a
translation layer.

### No inbound dependency, ever

The most important property of this context map is a non-edge: **nothing
points into `facility-layout` from the other four.** It does not read
`inventory-storage`'s stock. It does not know what a Task, an Assignment, a
Wave or a Shift is. None of them get write access to its aggregates.

A context that everyone depends on has to be cheap to depend on. No coupling
back means no startup ordering constraints, no circular deployment
dependencies, and no cascade beyond a plain read failure.

### Why this is a Generic Subdomain at all

The platform's DDD reference lists, among the disciplines to enforce:

> **Extract generic logic instead of duplicating it.** Cartonization is a
> good example: rather than implementing box-selection logic separately in
> both WMS (for planning/estimates) and WES (at point of pack), model it as
> its own Generic Subdomain both contexts call into.

Physical location is the same case in a different costume: needed by the WMS
tier for stow validity, needed by the WES tier for travel and congestion,
owned by neither. So it is referenced from both rather than duplicated in
either. See [Subdomain classification](../ddd/subdomain-classification.md).

## What would have to be built

For the planned edges to become real, in this repository:

1. An `internal/adapters/outbound/kafka` package implementing
   `ports.EventPublisher`, selected by an `EVENT_PUBLISHER=kafka|log`
   environment variable — the same pattern the four siblings use.
2. A topic — `warehouse.facility-layout.events` would follow the existing
   `warehouse.<service>.events` convention.
3. An `apis/asyncapi.yaml` (AsyncAPI 2.6.0, Spectral-linted) describing the
   envelope and every published event, matching the other four repos.

And, in the consumer repositories, an inbound consumer plus an
anti-corruption translation into their own models. All of that is future,
additive work, explicitly out of scope for this service as built. See
[Consuming this service](./consuming-this-service.md).
