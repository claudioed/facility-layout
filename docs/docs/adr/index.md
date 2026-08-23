---
id: index
title: Architecture Decision Records
sidebar_label: About ADRs
description: Why these records exist, the template they follow, and how to propose a new one.
---

# Architecture Decision Records

An **Architecture Decision Record** captures one architecturally significant
decision: what was decided, what was going on at the time that made the
decision necessary, and what the team now has to live with as a result.

The records here are not aspirational design documents. Each one reconstructs
a decision that is **actually visible in this repository** — in the code, in
`CLAUDE.md`, in `TASKS.md`, or in the shape of the API. If you can't point at
the consequence in the tree, it isn't an ADR.

## Why keep them

The alternative is that the reasoning lives in somebody's head, or in a
review thread nobody can find. Six months later the code says *what* the
system does and nothing says *why*, so the next person either preserves a
constraint that stopped mattering or removes one that still does.

For this service specifically the stakes are higher than usual: it is a
[Generic Subdomain](../ddd/subdomain-classification.md) that four other
contexts are meant to conform to. Decisions about its coded address format
and its enforcement points are expensive to reverse once anyone depends on
them, so the reasoning behind them is worth writing down.

## The template

These follow **Michael Nygard's** lightweight format — the de facto standard
— one markdown file per decision, numbered `0001-`, `0002-`, and so on:

| Section | Contains |
|---|---|
| **Title** | A short noun phrase naming the decision |
| **Status** | Proposed / Accepted / Deprecated / Superseded by ADR-NNNN |
| **Context** | The forces at play — technical, organisational, domain. Written in value-neutral language: what was true, not what we wanted. |
| **Decision** | The response to those forces, stated actively: *"We will…"* |
| **Consequences** | What becomes easier **and** what becomes harder. Both. A consequences section with no downsides is a sales pitch. |

Records are **immutable once accepted**. A decision that changes gets a new
record that supersedes the old one; the old one stays, with its status
updated. The history of what was believed and when is the point.

## The records

| # | Title | Status |
|---|---|---|
| [0001](./0001-hexagonal-ports-and-adapters.md) | Hexagonal (ports and adapters) architecture | Accepted |
| [0002](./0002-hierarchical-location-code.md) | Industry-standard hierarchical location code over a flat identifier | Accepted |
| [0003](./0003-placement-rules-at-registration-time.md) | PlacementRules enforced once at registration time | Accepted |
| [0004](./0004-rfc-7807-from-day-one.md) | RFC 7807 problem details from the first commit | Accepted |
| [0005](./0005-one-way-decommission.md) | One-way decommission, no reactivation in v1 | Accepted |
| [0006](./0006-partial-success-bulk-import.md) | Bulk import reports partial success per row | Accepted |

## One record that is deliberately absent

The four sibling services each have an ADR covering **Kafka plus the
CloudEvents envelope convention** for integration events. This repository
does not, because **this service does not publish integration events**. It
has an in-process log publisher, a buffered test publisher and a Postgres
`events` table — no broker, no topic, no `apis/asyncapi.yaml`.

Writing that ADR here would document a decision nobody made and a mechanism
nobody built. The CloudEvents `type` *naming convention* is adopted (see
[Domain events](../ddd/domain-events.md)), but adopting a naming convention
for consistency with four sibling repos is not an architecturally
significant decision on its own. When a broker adapter is actually added,
that will be the moment for ADR 0007. See
[Context map](../ecosystem/context-map.md).

## Proposing a new one

1. Copy the most recent record and renumber it. Numbers are never reused,
   even if a record is later withdrawn.
2. Write the **Context** first, and write it neutrally. If the context
   section already argues for the outcome, the decision was not really open.
3. Write **Consequences** honestly, including the ones you dislike. That
   section is what makes the record useful to whoever has to revisit it.
4. Open it as `Proposed`. Flip it to `Accepted` when it is agreed, or leave
   it `Proposed` and supersede it later if the discussion goes elsewhere.
5. Add it to the table above and to `sidebars.ts`.
