---
id: getting-started
title: Getting started
sidebar_label: Getting started
description: Run facility-layout locally, with or without Postgres, and walk the full site → zone → aisle → slot → layout example.
---

# Getting started

`facility-layout` is a single Go binary. It runs with no database at all
(in-memory adapters, log-only event publishing) or against Postgres 16.

## Run it with no database

```bash
go run ./cmd/facility
# facility-layout 2026/08/22 09:00:00 DATABASE_URL not set; using in-memory adapters
# facility-layout 2026/08/22 09:00:00 listening on :8080
```

This is the fastest way to explore the API. Everything is in process; the
event publisher writes to the log.

## Run it with Postgres

```bash
docker compose up -d postgres
export DATABASE_URL="postgres://facility:facility@localhost:5432/facility?sslmode=disable"
go run ./cmd/facility          # migrations run automatically at startup
```

Migrations can also be applied standalone with `golang-migrate`:

```bash
migrate -source file://migrations -database "$DATABASE_URL" up
```

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Listen address |
| `DATABASE_URL` | *(unset)* | Postgres DSN. Unset ⇒ in-memory adapters + log-only event publishing |
| `MIGRATIONS_PATH` | `migrations` | Directory of golang-migrate SQL files |

## Container

```bash
docker build -t claudioed/facility-layout .
docker run --rm -p 8080:8080 claudioed/facility-layout
```

## Build a building in six calls

Each step below is a real request against a running instance. The full
responses are shown in [Drawing the warehouse](../api-reference/drawing-the-warehouse.md)
and in the repository README.

```bash
# 1 — the site
curl -X POST localhost:8080/sites -H 'Content-Type: application/json' \
  -d '{"siteCode":"WH1","name":"Fulfilment Centre One"}'

# 2 — a behavioral zone inside its area
curl -X POST localhost:8080/sites/WH1/zones -H 'Content-Type: application/json' \
  -d '{"areaCode":"STOR","zoneCode":"AMB","temperatureClass":"Ambient","hazmat":false}'

# 3 — an aisle, with its walk-order position
curl -X POST localhost:8080/zones/WH1-STOR-AMB/aisles -H 'Content-Type: application/json' \
  -d '{"aisleCode":"A07","sequenceHint":7,"direction":"TwoWay"}'

# 4 — a reusable slot kind with its default capacity envelope
curl -X POST localhost:8080/location-types -H 'Content-Type: application/json' \
  -d '{"name":"PalletRack","defaultCapacity":{"maxWeightKg":1200,"maxVolumeM3":2.4}}'

# 5 — a rule: pallet racking is not rated for the cold
curl -X POST localhost:8080/placement-rules -H 'Content-Type: application/json' \
  -d '{"ruleId":"RULE-FRZ-NO-SHELF","locationType":"PalletRack","effect":"Deny","zone":{"temperatureClass":"Frozen"}}'

# 6 — the coded leaf slot itself
curl -X POST localhost:8080/locations -H 'Content-Type: application/json' \
  -d '{"locationCode":"WH1-STOR-AMB-A07-03-02-B","locationType":"PalletRack"}'
```

Then draw it:

```bash
curl localhost:8080/sites/WH1/layout                    # nested, renderable JSON
curl localhost:8080/zones/WH1-STOR-AMB/grid             # 2D matrix
curl "localhost:8080/sites/WH1/layout?format=svg" -o wh1.svg   # server-rendered floor plan
```

## Quality gates

The repository holds itself to the same five-stage bar as the four sibling
services:

```bash
go build ./...
go vet ./...
go test ./... -race
gofmt -l .
golangci-lint run ./...

# combined domain + application coverage (CI gate: >= 90%)
go test ./internal/domain/... ./internal/application/... -race \
  -coverprofile=coverage.out \
  -coverpkg=./internal/domain/...,./internal/application/...
go tool cover -func=coverage.out | tail -1

# BDD acceptance suite (godog/Gherkin, over the real HTTP API)
go test ./... -run TestFeatures -v

# architecture fitness tests (arch-go)
go test ./internal/architecture/... -v
```

The architecture rule described in
[Hexagonal architecture](../ddd/hexagonal-architecture.md) is an executable
test, not a convention: `internal/architecture/architecture_test.go` fails
the build if the domain ever imports the application layer.
