---
id: 0012-standard-metrics-convention
title: 12. Standard metrics convention across the fleet
sidebar_label: 12. Standard metrics convention
sidebar_position: 12
description: "The fleet-wide OTel metrics convention: a mandatory Tier 1 baseline (telemetry.Setup, otelchi RED instrumentation, /healthz) identical across every service, and a Tier 2 naming/attribute convention for business metrics. facility-layout is already compliant on both tiers as written -- no code changes required by this record."
---

# 12. Standard metrics convention across the fleet

## Status

Accepted.

## Context

Every service in the fleet installs the same OTel pipeline (service pods
--OTLP/gRPC--> otel-collector --OTLP/gRPC--> Jaeger, with a `prometheus`
exporter re-publishing metrics for Prometheus/Grafana — see
warehouse-infra/terraform/observability.tf). That pipeline is a real,
enforced contract: the Collector's DNS name and OTLP port are hard-coded in
every repo's Helm values, and Grafana ships one dashboard
(`go-runtime.json`) built from the Go-runtime metrics
`go.opentelemetry.io/contrib/instrumentation/runtime` emits — the one thing
every instrumented service already reports identically.

Past that shared plumbing, nothing was standard. An audit across all eight
services found:

- Two services (`order-management`, `labor-performance`) had no
  `telemetry.Setup` at all — no MeterProvider, no TracerProvider, not even
  the Go-runtime metrics every other service gets for free. `order-management`
  additionally had no HTTP RED instrumentation (`otelchi`).
- Of the six services that do call `telemetry.Setup`, HTTP request
  rate/error/duration (RED) instrumentation via `otelchi` was present in five
  and absent in one (`labor-performance`).
- Every service that emits a business metric invented its own instrument
  name, unit, and attribute keys independently:
  `inventory.reservations` (attr `outcome`), `workforce.labor_assignments`
  (attrs `outcome`, `reason`, `path.id`... — actually the assignment attr key
  was `workforce.path.id`, not namespaced under `path_id` the way
  `wes-work-planning` names its own `path_id` attribute),
  `fulfillment.tasks.claimed` / `fulfillment.tasks.completed` (attr
  `task.type`), `facility.location_slot.registrations`. No two services
  agree on whether the bounded-context prefix is `<context>.<noun>` or
  `<context>.<aggregate>.<verb>`, whether an attribute key uses a dot or an
  underscore, or whether an outcome is spelled as an attribute on one
  instrument or as two separate instruments.

A metric no dashboard-builder can predict the shape of in advance is a
metric nobody actually queries across services. The Go-runtime dashboard
being the fleet's only dashboard, eleven weeks into having Prometheus
running, is the visible symptom.

## Decision

Every service's metrics fall into two tiers. Tier 1 is mechanical and the
same for all eight services. Tier 2 is business-specific per bounded
context but follows one shared naming/attribute convention so a human or a
dashboard can find any service's metrics without reading that service's
source first.

### Tier 1 — mandatory baseline, identical across every service

1. **Runtime metrics.** `telemetry.Setup` installs the MeterProvider and
   calls `runtime.Start(runtime.WithMeterProvider(...))`
   (`go.opentelemetry.io/contrib/instrumentation/runtime`). Already the
   fleet's de facto standard; this ADR makes it load-bearing — a service
   without it is non-compliant, not merely incomplete.
2. **HTTP RED metrics.** Every service that serves HTTP wires
   `otelchi.Middleware` for tracing AND
   `otelchimetric.NewServerRequestDuration` for the duration histogram, in
   that order, ahead of the request logger:
   ```go
   r.Use(middleware.RequestID)
   r.Use(otelchi.Middleware(serviceName, otelchi.WithChiRoutes(r)))
   r.Use(otelchimetric.NewServerRequestDuration(otelchimetric.NewBaseConfig(serviceName)))
   r.Use(RequestLogger(logger))
   r.Use(middleware.Recoverer)
   ```
   This is the ONLY sanctioned way to get request rate/error/duration — no
   service hand-rolls its own HTTP counter.
3. **Health endpoint.** `GET /healthz` returning `{"status":"ok"}`. Already
   universal; noted here because Prometheus's own liveness assumptions and
   the Kong route depend on it existing, so it is now part of the same
   contract as the metrics above, not a separate one.

### Tier 2 — business metrics: naming convention, not a fixed metric set

A service picks whichever business events actually matter to it (this ADR
does not mandate what to measure — that stays a domain decision per
bounded context), but every business instrument follows this shape:

- **Instrument name:** `<bounded_context>.<aggregate_or_noun>.<verb_or_state>`,
  all lowercase, dot-separated, no abbreviations
  (`inventory.reservations.created`, not `inv.rsv.create`). Prefer ONE
  counter with an `outcome` attribute over two separately-named counters
  for the success/failure split of the same event — `inventory.reservations`
  with `outcome=created|revoked` is the fleet's best existing example of
  this and is the pattern to copy, not to rename away from.
- **Attribute keys:** dot-separated, matching OTel semantic-convention style
  (`task.type`, not `taskType` or `task_type`). An attribute that identifies
  *why* something happened is always named `reason`; an attribute that
  identifies *what happened to it* is always named `outcome`. Don't invent a
  third word for either concept.
- **Unit:** every counter declares `metric.WithUnit(...)` using UCUM curly-
  brace instrument syntax (`{reservation}`, `{task}`, `{assignment}`) — this
  is what already differs least across the fleet and should stay that way.
- **Description:** every instrument's `metric.WithDescription(...)` states
  what business condition the metric is a proxy for (see
  `inventory-storage`'s `reservationCounterName` comment for the standard
  this ADR expects: not just what the number counts, but what a human
  should conclude when it moves).
- **Placement:** the instrument lives in the SAME adapter tier as
  `telemetry.Setup` (an outbound `telemetry` adapter implementing an
  application-layer port, e.g. `ports.ReservationMetrics`), never called
  directly from `internal/domain`. A use case takes the port as a
  dependency and calls it once per outcome, exactly the shape
  `inventory-storage`'s `ReservationMetrics` already establishes.

This ADR does NOT require renaming metrics that already fit the shape
(`inventory.reservations`, `fulfillment.tasks.claimed`,
`facility.location_slot.registrations` are compliant as written). It DOES
require:
- `order-management` and `labor-performance` to gain a `telemetry.Setup`
  (Tier 1, item 1) and `otelchi` HTTP instrumentation (Tier 1, item 2), and
  at least one Tier-2 business counter each, following the shape above.
- `labor-performance` (which already has `telemetry.Setup` but no `otelchi`)
  to gain Tier 1, item 2.
- Any NEW business metric added anywhere in the fleet from this point on to
  follow the Tier 2 convention, reviewed the same way the RFC 7807 error
  shape or the Kafka envelope format are reviewed — a metric name is a
  public contract with every dashboard and alert built on it, not
  internal detail a PR reviewer can wave through unchecked.

### `facility-layout`'s compliance status

This service is already compliant with the full convention as written, on
both tiers, with no changes required by this record:

- **Tier 1.** `internal/adapters/outbound/telemetry` calls `telemetry.Setup`
  and installs the Go-runtime meter; `internal/adapters/inbound/http/server.go`
  wires `otelchi.Middleware` and `otelchimetric.NewServerRequestDuration`
  ahead of the request logger; `GET /healthz` already exists.
- **Tier 2.** `facility.location_slot.registrations` follows the
  `<bounded_context>.<aggregate>.<verb_or_state>` shape with an `outcome`
  attribute, a UCUM `{registration}` unit, and a description stating the
  business condition it proxies — the exact pattern this ADR generalizes.

No code changes accompany this record; it documents an already-satisfied
constraint so the convention itself, and this service's conformance to it,
are discoverable without reading source.

## Consequences

- A new service added to the fleet has an unambiguous starting checklist:
  copy `telemetry.Setup` from any compliant repo verbatim, wire the three
  `otelchi` middleware lines, and name its first business counter
  `<context>.<noun>.<verb>` with an `outcome`/`reason` attribute pair where
  applicable.
- Grafana can eventually get a second, genuinely cross-service dashboard
  (e.g. "fleet throughput") because instrument names are predictable rather
  than requiring per-service tribal knowledge to query.
- This is additive: no existing instrument is renamed, no existing
  dashboard breaks, and both gap-closing services keep failing OPEN on a
  missing Collector exactly like every other service (`Setup`'s exporters
  are non-blocking; see `inventory-storage`'s `telemetry.go` header comment
  on why a service must not fail to start over absent telemetry).
- The convention is documentation and review discipline, not a CI gate.
  Nothing currently enforces instrument-name shape mechanically (unlike the
  hexagonal dependency rule, which arch-go does enforce) — a future ADR
  could add a lint if drift recurs, but is not scoped here.
- For `facility-layout` specifically: no consequence beyond making an
  already-true property explicit and reviewable — this record is the
  citation future PRs adding a second business metric here should be held
  to.
