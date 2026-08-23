---
id: 0003-placement-rules-at-registration-time
title: 0003 — PlacementRules enforced once at registration time
sidebar_label: 0003 · Rules at registration time
description: Why placement legality is decided when a slot is created, rather than re-checked by every caller on every read.
---

# 0003 — PlacementRules enforced once at registration time

**Status:** Accepted

## Context

Some kinds of storage are not legal in some kinds of zone. Pallet racking may
not be rated for sub-zero temperatures; a hazmat zone may accept only
specific containment types; a frozen zone must not receive ambient shelving.
This is a real operational constraint with safety and compliance
consequences, and this context is the only place in the platform that knows
both what kind of slot something is and what kind of zone it sits in.

The question is not *whether* to enforce it. It is **where and when**.

The forces:

- **Multiple consumers will eventually ask the same question.**
  `inventory-storage` wants to know if a stow target is legal;
  `wes-work-planning` and `fulfillment-execution` want to know if a zone is
  usable. Three consumers evaluating the same rules independently is three
  implementations that will disagree the first time a rule changes.
- **Reads vastly outnumber writes.** A slot is registered once and read
  millions of times. Anything paid per-read is paid at the worst possible
  ratio.
- **Invalid state that merely gets detected is still invalid state.** If an
  illegal slot can exist in the database — even briefly, even flagged —
  something has to find it, something has to fix it, and something has to
  decide what happened to whatever was stored there meanwhile.
- **An aggregate cannot query a repository.** The rule set is
  cross-aggregate data. Whatever the answer is, it must not put persistence
  inside the domain.
- **"Rejected" is not an actionable error.** During a 500-row bulk import,
  an operator needs to know *which rule* refused *which row*.

Four options were considered:

1. Each **caller** checks before creating a slot.
2. Check **on read** — return legality alongside each slot.
3. A **nightly reconciliation job** flags illegal slots.
4. Check **at registration time**, inside the domain, once.

## Decision

We will enforce PlacementRules **once, at registration time, inside the
domain**, so that every slot in the database is legal by construction.

A `PlacementRule` is `(LocationType, Effect, ZonePredicate)`:

- `Effect` is `Allow` or `Deny`.
- The predicate matches on any of `zoneCode`, `temperatureClass`, `hazmat`.
  Every set field must match (AND); unset fields are wildcards; **at least
  one must be set** — a predicate that constrains nothing would match every
  zone in the building and is almost always a typo, so it is rejected at
  construction.

`RegisterLocationSlot` (and each row of `ImportFacilityLayout`) resolves the
zone, loads the applicable `RuleSet` from the repository, and passes both
into the aggregate constructor. The aggregate evaluates them:

```go
//  1. Any matching Deny rule naming this LocationType rejects it. Deny wins.
//  2. If any matching Allow rule exists for the zone at all, the zone is an
//     allow-list: the LocationType must be named by one of them.
//  3. Otherwise the zone is unconstrained and the placement is permitted.
func (rs RuleSet) Check(locationType string, attrs ZoneAttributes) error
```

The rejection error **always names the rule that was violated**:

```
location type violates a placement rule for this zone: PalletRack is denied in
zone WH1-STOR-FRZ by rule [RULE-FRZ-NO-SHELF: Deny PalletRack where
temperatureClass=Frozen]
```

It maps to HTTP `422 Unprocessable Entity` with problem type
`placement-rule-violated`.

### Why not the alternatives

| Option | Rejected because |
|---|---|
| **Each caller checks** | Reimplemented per consumer, drifts on the first rule change, and defeats the entire reason for extracting this Generic Subdomain — *"extract generic logic instead of duplicating it."* |
| **Check on read** | Pays the cost at the worst ratio, and still permits illegal slots to exist. Turns every consumer into a rules evaluator. |
| **Nightly job** | The invalid state exists in the meantime, and something must be built to remediate it and to reason about what was stored there. |

## Consequences

### Easier

- **Every Active slot in the database is legal.** A consumer that reads one
  back can treat it as already-validated: no temperature check, no hazmat
  check, no rule engine. That property is the single strongest argument for
  this being a service rather than a shared library.
- Rules are declared once, in one place, and are themselves an addressable
  REST resource (`/placement-rules`) that can be listed and inspected.
- Failures are actionable. Naming `RULE-FRZ-NO-SHELF` lets an operator go and
  look at the rule; "rejected" would not.
- The check is fully unit-testable with no infrastructure, because the rule
  set is passed in rather than fetched.
- Bulk import gets the same enforcement for free, per row, with the same
  named-rule error in the report.

### Harder

- **Changing a rule does not retroactively invalidate existing slots.** This
  is the real cost. If `RULE-FRZ-NO-SHELF` is added *after* a `PalletRack`
  already exists in a frozen zone, that slot stays. Enforcement is
  point-in-time by design.
  - Mitigation available today: `GET /sites/{siteCode}/layout` returns every
    slot with its zone and type, so an audit against the current rule set is
    a read plus a comparison. A first-class "revalidate" use case is
    deliberately not in v1.
- Rule ordering semantics have to be learned. "Deny wins, and any Allow turns
  the zone into an allow-list" is powerful but is not obvious from a single
  rule in isolation — which is why every rule carries a `Describe()` string
  and the API returns it.
- Adding a rule can make previously-valid registrations start failing, with
  no deploy and no code change. That is the intent, but it means rule
  definition is an operationally significant action.
- The rule set for a zone is loaded on every slot registration. For bulk
  imports of hundreds of rows this is repeated work — acceptable at the
  volumes involved, and the obvious optimisation (cache per zone within one
  import) is available if it ever matters.
