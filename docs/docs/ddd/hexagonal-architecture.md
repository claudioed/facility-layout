---
id: hexagonal-architecture
title: Hexagonal architecture
sidebar_label: Hexagonal architecture
description: Ports and adapters, the strict inward dependency rule, and the arch-go fitness test that enforces it.
---

# Hexagonal architecture

Ports and adapters, with one non-negotiable rule:

> **The domain depends on nothing. The application depends on the domain.
> Adapters depend on the application and the domain. Nothing points inward
> from the outside except through a port.**

No framework type, no SQL type, no `net/http` type ever appears in the domain
layer.

```mermaid
graph LR
    subgraph Inbound["Driving adapters"]
        HTTP["inbound/http<br/>chi handlers · DTOs<br/>RFC 7807 mapping · SVG"]
    end

    subgraph App["Application"]
        UC["usecases<br/>10 use cases"]
        P["ports<br/>OUT interfaces only"]
    end

    subgraph Dom["Domain — pure Go"]
        D["site · zone · aisle<br/>placement · slot · shared"]
    end

    subgraph Outbound["Driven adapters"]
        PG["outbound/postgres<br/>pgxpool + migrations"]
        MEM["outbound/memory<br/>thread-safe in-memory"]
        EV["outbound/events<br/>log · buffered"]
    end

    HTTP --> UC
    UC --> D
    UC --> P
    PG -.implements.-> P
    MEM -.implements.-> P
    EV -.implements.-> P
```

## Package layout

```
cmd/facility/                 main.go — composition root (the only place that knows all layers)
internal/
  domain/                     pure business logic; imports nothing but stdlib and itself
    shared/                   LocationCode, Capacity, TemperatureClass, Direction, Status, the 8 events
    site/  zone/  aisle/      the structural aggregates
    placement/                LocationType, PlacementRule, RuleSet evaluation
    slot/                     LocationSlot — the coded leaf aggregate
  application/
    ports/                    OUT interfaces only (repos, EventPublisher, Clock)
    usecases/                 one struct per use case + the two read-model assemblers
  adapters/
    inbound/http/             chi handlers, DTOs, RFC 7807 error mapping, SVG rendering
    outbound/postgres/        pgxpool repos + golang-migrate migrations
    outbound/memory/          thread-safe in-memory repos for tests and local runs
    outbound/events/          log + buffered publishers (broker-ready interface)
  architecture/               arch-go fitness tests
migrations/                   golang-migrate SQL
```

## The ports

All eight are **driven** (outbound) interfaces. There are no inbound port
interfaces: a use case struct *is* the inbound port, called directly by the
HTTP adapter.

```go
type SiteRepo interface {
	Save(ctx context.Context, s *site.Site) error
	FindByCode(ctx context.Context, code string) (*site.Site, error)
	List(ctx context.Context) ([]*site.Site, error)
}

type ZoneRepo interface {
	Save(ctx context.Context, z *zone.Zone) error
	FindByID(ctx context.Context, id string) (*zone.Zone, error)
	ListBySite(ctx context.Context, siteCode string) ([]*zone.Zone, error)
}

type AisleRepo interface {
	Save(ctx context.Context, a *aisle.Aisle) error
	FindByID(ctx context.Context, id string) (*aisle.Aisle, error)
	ListByZone(ctx context.Context, zoneID string) ([]*aisle.Aisle, error)
}

type SlotRepo interface {
	Save(ctx context.Context, s *slot.LocationSlot) error
	FindByCode(ctx context.Context, code shared.LocationCode) (*slot.LocationSlot, error)
	ListByAisle(ctx context.Context, aisleID string) ([]*slot.LocationSlot, error)
	ListByZone(ctx context.Context, zoneID string) ([]*slot.LocationSlot, error)
}

type LocationTypeRepo  interface { /* Save · FindByName · List */ }
type PlacementRuleRepo interface { /* Save · FindByID   · List */ }

type EventPublisher interface {
	Publish(ctx context.Context, event shared.DomainEvent) error
}

type Clock interface {
	Now() time.Time
}
```

Note what the port signatures traffic in: **domain types**. `SlotRepo` takes
a `shared.LocationCode`, not a `string`. An adapter cannot hand the
application an unvalidated code, because the type it must supply can only be
produced by the validating constructor.

`Clock` is a port for the same reason: no domain or application code calls
`time.Now()` directly, so every event timestamp is deterministic under test.

## The ten use cases

| # | Use case | Shape |
|---|---|---|
| 1 | `RegisterSite` | thin — validate, check uniqueness, save, publish |
| 2 | `RegisterZone` | thin + parent Site resolution |
| 3 | `RegisterAisle` | thin + parent Zone resolution |
| 4 | `RegisterLocationType` | thin |
| 5 | `DefinePlacementRule` | thin + LocationType existence check |
| 6 | **`RegisterLocationSlot`** | **real orchestration** — full chain of custody + rule set |
| 7 | `DecommissionLocationSlot` | load, transition, save, publish |
| 8 | **`ImportFacilityLayout`** | **real orchestration** — per-row, partial success |
| 9 | `GetSiteLayout` | read-model assembler — no writes, no events |
| 10 | `GetZoneGrid` | read-model assembler — no writes, no events |

Only two use cases carry real logic. That is intentional: the invariants live
in the domain, and the application layer's job is resolution and
orchestration, not rule-keeping.

## Aggregates do not reach outside themselves

The clearest expression of the discipline is
`slot.NewLocationSlot`'s signature:

```go
func NewLocationSlot(
	code shared.LocationCode,
	locationType placement.LocationType,
	capacityOverride shared.Capacity,
	attrs placement.ZoneAttributes,
	rules placement.RuleSet,
) (*LocationSlot, error)
```

The slot must satisfy every applicable `PlacementRule`, but it cannot query a
repository to find them — that would put persistence inside the domain. So
the use case loads the rule set and the zone's attributes and passes them in.
The aggregate stays pure and fully unit-testable with no test doubles at all.

## The rule is a test, not a convention

`internal/architecture/architecture_test.go` uses
[`arch-go`](https://github.com/arch-go/arch-go) to encode the dependency
rule as an executable fitness test. It fails the build if:

- the domain imports anything internal other than the domain,
- the application layer imports anything but the domain,
- an inbound adapter imports an outbound adapter, or vice versa,
- anything other than `cmd` wires every layer together,
- the `ports` package contains anything but interfaces,
- the domain grows a catch-all `utils`/`common` package.

It runs as its own blocking `arch-test` job in CI. A layering violation is
therefore a red build, not a review comment somebody might miss.

## Composition root

`cmd/facility/main.go` is the only file that knows about all the layers. It
reads the environment, picks the adapters, constructs the use cases and
mounts the router:

- `DATABASE_URL` set → Postgres repositories, migrations run at startup, the
  Postgres event publisher.
- `DATABASE_URL` unset → in-memory repositories and the log publisher.

Because the swap happens at exactly one place and every port is an interface
over domain types, the HTTP layer and the entire test suite are identical in
both modes. The BDD suite in `features/` runs the real router over the
in-memory adapters via `httptest.NewServer`, which is only possible because
nothing above the port knows which adapter it got.
