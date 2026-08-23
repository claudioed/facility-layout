---
id: subdomain-classification
title: Subdomain classification
sidebar_label: Subdomain classification
description: Why Facility Layout is a Generic Subdomain, and what that buys the platform.
---

# Subdomain classification

**Facility Layout is a Generic Subdomain.**

DDD splits a domain into Core, Supporting and Generic subdomains by
*competitive differentiation*. Physical-location structure is well
understood, has an established industry pattern, and is not where a retailer
or 3PL wins. It sits in the same bucket the platform's DDD reference puts
Cartonization and WCS in.

| Subdomain | Type | Why |
|---|---|---|
| Fulfillment Orchestration & Optimization (`wes-work-planning`) | **Core** | Continuous re-planning to the fastest/cheapest path is the differentiator. |
| Inventory & Slotting (`inventory-storage`) | **Core** | Random stow + bin-accurate tracking is a genuine operational innovation and the backbone of pick-path efficiency. |
| Task lifecycle: Pick/Pack/SLAM (`fulfillment-execution`) | **Core** | Directly drives throughput and accuracy at scale. |
| Labor & Workforce Management (`workforce-management`) | **Supporting** | Allocates workforce to workload; important, industry-common. |
| **Physical facility layout (`facility-layout`)** | **Generic** | Well-understood, standardised coded-location structure. Not a differentiator; needed identically by contexts on both sides of the WMS/WES line. |
| Cartonization | **Generic** | Modelled separately, referenced by both WMS and WES rather than duplicated in either. |
| WCS / equipment control | **Generic** | Buy, don't build — device orchestration is rarely a competitive advantage. |

## The argument, from the reference docs

The platform reference lists, among the DDD disciplines to enforce:

> **Extract generic logic instead of duplicating it.** Cartonization is a
> good example: rather than implementing box-selection logic separately in
> both WMS (for planning/estimates) and WES (at point of pack), model it as
> its own Generic Subdomain both contexts call into.

Physical location has exactly the same shape as Cartonization:

- It is needed by the **WMS tier**: `inventory-storage` cannot accept a stow
  without knowing the scanned location is real and active. The
  Amazon-fulfillment reference is explicit that placing an item without a
  valid location scan is how inventory becomes "lost."
- It is needed by the **WES tier**: `wes-work-planning` and
  `fulfillment-execution` reason about `Zone`, `Travel Path` and
  `Congestion`. Those words are already in the WES ubiquitous language — but
  nothing in the platform was the source of truth for what a Zone *is*, or
  which aisle is next in walk order. An Aisle's `SequenceHint` is the
  concrete answer to that previously-missing travel-distance input.
- **Neither tier owns it.** Both consume it.

So it is extracted: its own bounded context, its own service, its own data
store — not a package bolted onto `inventory-storage`.

## Generic does not mean unimportant

"Generic" is a statement about *differentiation*, not about *criticality*.
The map being wrong is catastrophic; the map being clever is worthless. That
shapes how this service is built:

- **Correctness over cleverness.** Every invariant is enforced in the domain
  and unit-tested on both the passing and the failing path.
- **Stability over feature velocity.** Four other services will conform to
  this context's Published Language. Breaking it is expensive for everyone,
  so the surface is small and boring on purpose.
- **Buy-shaped, build-because-we-must.** The location-code hierarchy is
  copied from the industry rather than invented, precisely because there is
  no advantage in being different here. See
  [ADR 0002](../adr/0002-hierarchical-location-code.md).

## Relationship to the rest of the system

This service is an **Open Host Service** with a **Published Language** (its
domain events plus a stable REST surface). It has **no inbound dependency**
on any of the other four services and never will:

- It does not read `inventory-storage`'s stock.
- It does not know what a Task, an Assignment, a Wave or a Shift is.
- It never mutates another context's data, and none of them get write access
  to this one's aggregates.

`inventory-storage`, `wes-work-planning`, `workforce-management` and
`fulfillment-execution` are all downstream **Conformists** to whatever this
service publishes.

:::note[Honest status]
The Conformist relationship above is a **strategic decision that is not yet
technically wired**. This service currently has no Kafka integration and no
live consumer. See [Context map](../ecosystem/context-map.md) for exactly
what exists today versus what is planned.
:::

## Where it sits in the WMS/WES/WCS layering

The Amazon-fulfillment reference classifies "bin-accurate location" inside
the WMS-tier **Inventory & Slotting** core subdomain, and notes that context
exposes it as an Open Host Service. This service is the generalized,
multi-consumer version of that same concern — which is why its CloudEvents
`type` namespace carries the `wms` subdomain segment:

```
com.warehouse.wms.facility-layout.<entity>.<EventName>
```

The layer label is inherited from where the concern originates; the bounded
context is its own. That is the reference's own advice applied literally:
carve bounded contexts on capabilities, not on the three product labels.
