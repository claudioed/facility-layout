# Quality engineering — facility-layout

The same five-stage bar the other four warehouse-systems services meet,
applied here from the first commit rather than retrofitted.

## 1 — Linting

`.golangci.yml` is copied **verbatim** from `../inventory-storage`: same
linters (`errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`,
`bodyclose`, `misspell`, `unconvert`, `gocritic`), same `_test.go` errcheck
exclusion, same `gofmt` + `goimports` formatters.

```sh
golangci-lint run ./...   # 0 issues
gofmt -l .                # empty
```

## 2 — Coverage

Gate: **>= 90%** statement coverage combined across `internal/domain/...`
and `internal/application/...`.

```sh
go test ./internal/domain/... ./internal/application/... -race \
  -coverprofile=coverage.out \
  -coverpkg=./internal/domain/...,./internal/application/...
go tool cover -func=coverage.out | tail -1
```

Achieved: **99.3%**.

The last few percent come from a fault-injection suite
(`internal/application/usecases/faults_test.go`) that wraps each in-memory
adapter in a decorator failing exactly one operation, so every
`if err != nil { return err }` branch in the application layer — including
"the write landed but the event could not be published" — is exercised for
real rather than assumed.

The invariants CLAUDE.md calls out each have an explicit failing-path test:

- slot registration rejected when the Site/Zone/Aisle chain does not resolve
  (unknown site, unknown zone, unknown aisle)
- slot registration rejected when a link exists but is not Active
  (decommissioned site, zone, aisle)
- slot registration rejected when it violates a PlacementRule, with the
  error naming the violated rule
- duplicate LocationCode rejected — including re-registering a
  decommissioned code
- Zone registration rejected against an unknown or decommissioned Site
- Aisle registration rejected against an unknown or decommissioned Zone

## 3 — Integration tests

Every outbound Postgres adapter has a build-tagged
(`//go:build integration`) round-trip test that skips silently without
`DATABASE_URL`:

```sh
docker compose up -d postgres
DATABASE_URL="postgres://facility:facility@localhost:5432/facility?sslmode=disable" \
  go test -tags=integration ./... -race -count=1
```

Five tests cover the Site repo, the Zone+Aisle repos, the LocationType and
PlacementRule repos (including that a partially-unconstrained
`ZonePredicate` round-trips through SQL `NULL`s), the Slot repo (including
that the seven LocationCode segments are stored as real columns and that
ordering by them works), and the Postgres outbox publisher (including that
the CloudEvents `type` is persisted verbatim). They self-migrate: each test
runs `RunMigrations` and truncates, so execution order does not matter.

## 4 — Mutation testing

gremlins against `internal/domain/...` only, exploratory and **never
blocking** — the `mutation` CI job is gated to `workflow_dispatch` and the
weekly schedule. Baseline: **89.90% test efficacy, 100% mutator coverage, 0
uncovered mutants**. See [MUTATION.md](MUTATION.md) for the full triage of
the 10 survivors (nine are ASCII-boundary noise in the `[A-Z0-9]`
validators; one is a near-equivalent mutant in an error-message builder).

## 5 — Architecture fitness tests

`internal/architecture/architecture_test.go` encodes the hexagonal
dependency rule as executable tests using
[arch-go](https://github.com/arch-go/arch-go) — the Go analogue of ArchUnit
— one subtest per rule so a failure names which rule broke:

1. `internal/domain/**` depends on nothing internal except `internal/domain/**`
2. `internal/application/**` depends only on domain and itself
3. inbound adapters never depend on outbound adapters
4. outbound adapters never depend on inbound adapters
5. nothing under `internal/**` imports `cmd/**` (the composition root stays a leaf)
6. `internal/application/ports` contains interfaces only

```sh
go test ./internal/architecture/... -v
```

## CI

`.github/workflows/ci.yml` — blocking on every push and PR:

| Job | Gate |
|---|---|
| `lint` | `golangci-lint` v2.13.1, zero issues |
| `test` | build, vet, `go test -race`, coverage >= 90% |
| `bdd` | godog/Gherkin acceptance suite over the real HTTP API |
| `integration` | build-tagged Postgres tests against a `postgres:16` service |
| `api-lint` | Spectral against `apis/openapi.yaml`, `--fail-severity=warn` |
| `helm-lint` | `ct lint --charts charts/facility-layout` |
| `arch-test` | arch-go hexagonal dependency rules |

Non-blocking: `mutation` (schedule / manual dispatch only).

Release: `docker-publish` runs only on a push to `main`, only after every
blocking job is green, and pushes `claudioed/facility-layout:latest` plus a
short-SHA tag to Docker Hub.
