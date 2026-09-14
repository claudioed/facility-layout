# How to add an integration event (publish and consume)

Use when asked to publish a new cross-context integration event from this
service, or to add a Kafka-fed local cache/consumer to this service
itself. This fleet's Kafka is ONE broker platform-wide — every design
decision below exists because that shared-broker reality has already
caused a real incident once (wes-work-planning#67, described below).

facility-layout is unusual in this fleet: it is an **Open Host Service**
whose domain events ARE its Published Language (ADR-0009). It currently
has no cross-service consumer of its own — every sibling service
(inventory-storage, wes-work-planning, workforce-management,
fulfillment-execution) is a Conformist reading FROM it, never the other
way around (see this repo's `AGENTS.md`: "This service has NO inbound
dependency on any of the other four fleet services and never will"). So
this guide's "publish" half is this repo's own lived pattern; its
"consume" half documents the rules a future consumer here would have to
follow, plus the one Kafka consumer this repo DOES already run — the
analytics projector, which is same-service, not cross-context.

## Publishing a new integration event

### 1. Is it actually new, or an addition to an existing event?

Unlike a service that forwards a single enriched event, this publisher
(`internal/adapters/outbound/kafka/publisher.go`) emits EVERY domain
event to the integration topic — the whole Published Language, by
design doc comment: "this publisher therefore emits EVERY domain event
to the integration topic". So there is rarely a decision to make about
WHETHER to publish; the real decision is whether your change is a new
event type or an additive field on an existing one, because that
distinction is what ADR-0013 makes into a compatibility contract. ADR-0016
added `role` to the already-live `LocationTypeRegistered`/
`LocationSlotRegistered` events (an additive field); ADR-0017 added four
brand-new event types (`LocationGeometryUpdated`, `AisleGeometryUpdated`,
`FixedStructureRegistered`, `CrossAisleRegistered`) specifically so
ADR-0013's existing field-level contract with inventory-storage's
`facilitycache` consumer stayed untouched. Check
`docs/docs/adr/0013-first-published-language-consumer.md`'s table of
fields a live downstream Conformist depends on before renaming or
removing anything on `ZoneRegistered`, `LocationSlotRegistered`, or
`LocationSlotDecommissioned` — those three are the fields actually
depended on today, not aspirational.

### 2. Envelope: CloudEvents-like, structured mode

Every message on `warehouse.facility.events` (see `Topic` constant,
`internal/adapters/outbound/kafka/publisher.go:31`) is an `Envelope`:

```go
type Envelope struct {
    EventId    string          `json:"event_id"`
    EventType  string          `json:"event_type"`
    OccurredAt time.Time       `json:"occurred_at"`
    Source     string          `json:"source"`
    Data       json.RawMessage `json:"data"`
}
```

`Data` carries the domain event's own JSON verbatim — the events already
serialize themselves to their wire shape (their struct tags ARE the
contract), so there is no per-event marshalling switch in the publisher.
The analytics topic (`warehouse.facility.analytics`,
`analytics_publisher.go`) uses the same shape plus a `schema_version`
int — a SEPARATE adapter (`AnalyticsPublisher`, not `Publisher`) so the
integration and analytics streams evolve independently; the composition
root fans out to both.

### 3. Partition key: the aggregate's identity, with a real fallback

`aggregateKey(event)` (`publisher.go:101`) type-switches on the event and
returns its aggregate id (`SiteCode`, `ZoneID`, `AisleID`, etc.) so all
events for one aggregate land on the same partition, preserving
per-aggregate order. Adding a new event type means adding a `case` here
too — `FacilityLayoutImported` and the `default` both fall back to
`event.EventType()` because a bulk-import event has no single natural
aggregate id; that fallback still gives a stable, non-empty key rather
than an empty string.

### 4. Contract + docs

- Add the message to `apis/asyncapi.yaml` under this service's channel.
- Regenerate the AsyncAPI HTML reference:
  ```bash
  cd docs && npm run gen-async-docs:all
  ```
  This repo's `docs-api-drift` CI job fails the PR if the generated
  `static/asyncapi/` output doesn't match a fresh regen.

### 5. Test

Unit test the marshal shape and the aggregate-key switch against a fake
`Writer` (see `publisher_test.go`/`analytics_publisher_test.go` — never a
real broker in a unit test). This repo's own architecture fitness test
`TestKafkaIntegrationTestsUseTestcontainers`
(`internal/architecture/fitness_test.go`) scans every
`*_integration_test.go` file that touches `segmentio/kafka-go` and fails
the build if it skip-gates on `os.Getenv("KAFKA_BROKERS")`, hardcodes
`localhost:9092`, or doesn't import
`testcontainers-go/modules/kafka` — enforced statically, not just by
convention, because this fleet's CI `integration` job
(`.github/workflows/ci.yml`) provisions Postgres only. This repo's
current integration tests (`postgres_integration_test.go`,
`travel_integration_test.go`, `telemetry_integration_test.go`,
`repos_integration_test.go`, `geometry_integration_test.go`) are all
Postgres-only, so the fitness test currently has nothing to flag — the
first Kafka-touching integration test added here is the one that has to
get this right.

## Consuming an integration event (this repo's own analytics projector, and any future cross-context consumer)

### 1. Never import a sibling's Go packages

If facility-layout ever consumes a SIBLING service's topic (it doesn't
today — see the intro above), the rule is the one inventory-storage's
`facilitycache/consumer.go` states in its own doc comment: know the
sibling's topic name and payload shape ONLY, never its Go types. Mirror
the payload struct locally rather than adding a module dependency.

### 2. Choose the right consumer-group pattern — this is the part that bites

Two DIFFERENT correct patterns exist. Picking the wrong one for the use
case is THE most common integration-event mistake in this fleet, learned
from a real incident (wes-work-planning#67): a fixed shared consumer
group id on a consumer meant to run as exactly one instance per
environment let a local dev/test harness process join the SAME broker's
SAME group as a live in-cluster Deployment, and Kafka's rebalance
protocol handed the partition to only ONE of the two group members — the
other silently starved.

**Pattern A — long-lived, single-instance consumer group (a named
constant).** Use when exactly ONE instance of this consumer ever runs at
a time. This repo's OWN analytics projector is the concrete example:
`AnalyticsConsumerGroup = "facility-analytics"`
(`internal/adapters/inbound/kafka/analytics_consumer.go:28`) is a plain
named constant, reused across restarts — correct because Kafka's
committed-offset resume semantics are exactly what a single, always-one-
instance projector wants: pick up where it left off. Note this constant
is a named symbol a reader can trace, not an inline literal buried in a
`kafkago.ReaderConfig{...}` call — see the fitness-test point below.

**Pattern B — per-process-unique consumer group (a generated id).** Use
when the consumer rebuilds a complete read model from a topic's FULL
history on every start (an event-sourced local cache, not a fixed
single-instance worker) — see inventory-storage's `facilitycache/
consumer.go` `consumerGroupPrefix` + `uniqueConsumerGroup()` for the
reference shape. The group id MUST be unique per process instance
(hostname+PID+timestamp), NEVER a fixed shared string, because a
brand-new process joining a group an EARLIER instance already consumed
resumes from that instance's committed offset — the new process gets
marked "ready" with an empty local cache having replayed nothing.

**This repo's own fitness test enforces the naming discipline, not the
pattern choice.** `TestKafkaConsumerGroupNeverHardcodedInline`
(`internal/architecture/fitness_test.go`) bans `GroupID: "literal-string"`
anywhere in the codebase — it does NOT ban long-lived named groups (the
projector's `AnalyticsConsumerGroup` constant is a deliberate, explicitly
whitelisted-by-comment exception); it bans an inline string literal a
reviewer cannot trace back to a definition and reasoning. This repo
currently has no cross-context (Pattern B) consumer, so the test has
nothing to flag on that front today — it exists for fleet consistency and
to catch a future one that reintroduces the incident's shape.

### 3. Readiness gate, if a future consumer here backs a local cache

If a consumer replays a topic's full history to build a cache other code
depends on, expose a `Ready()` gate the health check consults, and block
readiness (not process startup) until the initial replay finishes. A
readiness check that only re-evaluates on a NEW message arriving
deadlocks forever on an ordinary restart of a shared/already-caught-up
group. This repo's existing analytics projector sidesteps the whole
question by using Pattern A with `StartOffset: kafkago.FirstOffset`
(`analytics_consumer.go:95`, only affecting the FIRST join since a
committed offset takes precedence after that) plus idempotent
`ProcessedEvents.MarkProcessed` per `event_id` — no separate readiness
concept was needed because the projector's own read path already handles
"haven't caught up yet" by simply not having the data.

## Verify before opening the PR

```bash
make check-all    # includes arch-test — will catch a reintroduced
                    # inline GroupID literal or an auth-middleware regression
```

Prove any new fitness-test-adjacent behavior actually matters by running
the specific scenario against a real broker if this repo has
testcontainers-based integration tests for the consumer/publisher
touched.
