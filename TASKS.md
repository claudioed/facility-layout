# Build Tasks — Facility Layout

Build the full bounded context described in CLAUDE.md, in order. Keep
`go build ./...` and `go test ./...` green throughout. Read
/Users/claudioed/docs/amazon-fulfillment-ddd.md and
/Users/claudioed/warehouse-systems-ddd.md for domain context first, then
CLAUDE.md in this repo for the concrete model.

## Task 0 — Skeleton
- `go mod init github.com/claudioed/facility-layout`; create the layout from
  CLAUDE.md's Architecture section; `.gitignore` (bin/, .env); add chi + pgx
  + golang-migrate deps (match versions from `../inventory-storage/go.mod`
  where there's overlap, to stay consistent across the fleet).

## Task 1 — Domain (pure Go)
- shared: `LocationCode` value object (7 typed segments, `String()` +
  `ParseLocationCode()`, validation per CLAUDE.md's table), `Capacity`
  value object (weight + volume, both positive), DomainEvent + the 8 events.
- site: `Site` aggregate (SiteCode uniqueness enforced at the
  application/repo layer, not in the aggregate itself — a single aggregate
  can't see its siblings; but reject empty/malformed SiteCode here).
- zone: `Zone` aggregate, scoped to a Site, carries TemperatureClass +
  Hazmat flag.
- aisle: `Aisle` aggregate, scoped to a Zone, carries SequenceHint +
  Direction.
- placement: `LocationType` + `PlacementRule` — a rule references a
  LocationType and a Zone-matching predicate (by ZoneCode or by
  TemperatureClass/Hazmat).
- slot: `LocationSlot` aggregate — identity is its LocationCode; validates
  its LocationType against every applicable PlacementRule at construction
  (the rule set is passed in by the use case, since a single aggregate
  can't query a repository — see CLAUDE.md's invariants and Vernon's
  "aggregates don't reach outside themselves" discipline). Status lifecycle
  Active -> Decommissioned (one-way, per CLAUDE.md).
- Unit tests for EVERY invariant listed in CLAUDE.md's "Definition of done",
  including every failing path.

## Task 2 — Application
- ports: SiteRepo, ZoneRepo, AisleRepo, SlotRepo, LocationTypeRepo,
  PlacementRuleRepo, EventPublisher, Clock.
- usecases: all 10 use cases from CLAUDE.md. `RegisterLocationSlot` and
  `ImportFacilityLayout` are the two with real orchestration: they resolve
  the Site->Zone->Aisle chain via the repos (rejecting on any
  missing/Decommissioned link) and fetch applicable PlacementRules before
  constructing the `LocationSlot` domain object. `GetSiteLayout` and
  `GetZoneGrid` are pure read-model assemblers over the repos — no writes,
  no events.
- `ImportFacilityLayout` reports partial success: process every row, collect
  per-row success/failure, return the full report rather than
  all-or-nothing aborting on the first bad row (a 500-row CSV with 3 bad
  rows should still create the other 497 and tell you exactly which 3 and
  why).
- Unit-test against in-memory adapters.

## Task 3 — Outbound adapters
- memory: thread-safe impls of every port.
- postgres: pgxpool repos + migrations (sites, zones, aisles, location_types,
  placement_rules, location_slots tables — model the LocationCode segments
  as real columns, not just a serialized string, so `GetZoneGrid` can query
  by aisle/bay/level directly); build-tagged integration test (skip w/o
  DATABASE_URL) per repo.
- events: log/buffered publisher (same shape as the other four services'
  `internal/adapters/outbound/events` package — copy the pattern).

## Task 4 — Inbound HTTP
- chi router + handlers for every endpoint in CLAUDE.md's REST API section.
- RFC 7807 (`application/problem+json`) error responses from the start —
  do NOT build a bespoke error shape first (see CLAUDE.md's Tech &
  Standards section: this is the one place this repo deliberately does NOT
  replicate the other four services' history, it starts where they ended
  up). `Location` header on every 201. Correct status codes (201 create,
  200 read, 204 no-content actions, 404 not found, 409 conflict e.g.
  duplicate LocationCode, 422 for a PlacementRule violation, 400 malformed
  input).
- `GET /sites/{siteCode}/layout` and `GET /zones/{zoneId}/grid` are the
  headline "draw the warehouse" endpoints — get their JSON shape right and
  cover them with real tests asserting the nested/grid structure, not just
  a 200 status.
- SVG rendering (`?format=svg` on the layout endpoint) is a stretch goal —
  only attempt after everything else in this task list is complete and
  green; skip it under time pressure per CLAUDE.md.
- httptest per endpoint against in-memory repos, `/healthz`.

## Task 5 — Composition & ops
- `cmd/facility/main.go` wires env -> adapters -> use cases -> router.
- `docker-compose.yml` (pg16, same shape as
  `../inventory-storage/docker-compose.yml` — copy and adapt service/db
  names to `facility`).
- `Dockerfile` — copy `../inventory-storage/Dockerfile` verbatim, changing
  only the binary name (`facility`) and `cmd/facility` path.
- README.md with run steps + curl examples per CLAUDE.md's Definition of
  Done (the full worked example: site -> zone -> aisle -> location type ->
  placement rule -> slot -> GET layout -> GET grid, with real JSON shown).

## Task 6 — Quality engineering (mirrors Task 10 in the other four repos)

Same five-stage bar the other four services already meet, applied here from
the start rather than retrofitted:

1. **Linting**: copy `../inventory-storage/.golangci.yml` verbatim into this
   repo's root. `golangci-lint run ./...` exits 0. `gofmt -l .` empty.
2. **Coverage**: `internal/domain/...` + `internal/application/...` combined
   >= 90% statement coverage. Use the same measurement command as the other
   repos:
   ```sh
   go test ./internal/domain/... ./internal/application/... -race \
     -coverprofile=coverage.out \
     -coverpkg=./internal/domain/...,./internal/application/...
   go tool cover -func=coverage.out | tail -1
   ```
3. **Integration tests**: every outbound Postgres adapter gets a
   build-tagged (`//go:build integration`) round-trip test against a live
   Postgres (docker-compose up first).
4. **Mutation testing**: gremlins against `internal/domain/...` only,
   exploratory/triaged not gated (same posture as the other four —
   scheduled/manual-dispatch CI job, never blocking a push/PR).
5. **CI**: `.github/workflows/ci.yml` — copy the STRUCTURE from
   `../inventory-storage/.github/workflows/ci.yml` (top-level
   permissions/concurrency/defaults, `lint`, `test` w/ coverage gate,
   `integration`, `mutation` gated to workflow_dispatch/schedule only) and
   adapt names/paths/DB creds to this service. Do NOT include the `bdd`,
   `api-lint`, `helm-lint`, or `arch-test` jobs yet — those are separate
   later tasks (see Task 7-10 below), this task only needs the four core
   jobs plus `docker-publish` gated on
   `needs: [lint, test, integration]`.

**Definition of done for Task 6**: all five stages' gates pass, report the
actual coverage number achieved.

## Task 7 — REST API hardening + OpenAPI 3.0.3 docs + Spectral CI gate

Same shape as the other four repos' Task 11 (see
`../inventory-storage/REST_API_TASK.md` for the exact bar — read it, this
task mirrors it), but scoped down since this service starts already on RFC
7807 (Task 4 already did that): audit REST maturity level 2 compliance,
then write a very detailed `apis/openapi.yaml` (3.0.3) covering every route
with full request/response schemas, a shared `Problem` schema component,
and REAL domain-grounded examples using this service's own ubiquitous
language (real LocationCodes like `WH1-STOR-AMB-A07-03-02-B`, real Zone/
Aisle names — not foo/bar). Put it under `apis/openapi.yaml` (this repo
starts directly in the `apis/` folder convention the other four repos moved
to later — see their `apis/` folders for the pattern). Add `.spectral.yaml`
(copy from `../inventory-storage/.spectral.yaml`) and a new `api-lint` CI
job (copy from `../inventory-storage/.github/workflows/ci.yml`'s `api-lint`
job, adjusted to lint only `apis/openapi.yaml` — no AsyncAPI in this repo
yet, this service doesn't publish integration events to Kafka in this task,
only the in-process log publisher). Add `api-lint` to `docker-publish`'s
`needs` list.

**Definition of done**: `spectral lint apis/openapi.yaml --ruleset
.spectral.yaml --fail-severity=warn` exits 0, every route in the router has
a corresponding `paths` entry (report N/N), CI job green.

## Task 8 — Architecture fitness tests (arch-go)

Same as the other four repos' Task 14 (see
`../inventory-storage/ARCH_TEST_TASK.md` for the exact rule set to encode —
read it, apply the identical hexagonal dependency rule to this repo's
package layout). Add `internal/architecture/architecture_test.go` using
`github.com/arch-go/arch-go`. New `arch-test` CI job, added to
`docker-publish`'s `needs` list.

## Task 9 — Helm chart

Same as the other four repos' Task 15: a Helm chart at
`charts/facility-layout/`, copied and adapted from
`../inventory-storage/charts/inventory-storage/` (same template files:
deployment, service, ingress, hpa, serviceaccount, configmap, secret,
_helpers.tpl, NOTES.txt, values.yaml). Adjust `values.yaml`'s
`image.repository`, `service`, `config`, `database` sections to this
service's actual env vars and port. Add a `helm-lint` CI job (copy from
`../inventory-storage/.github/workflows/ci.yml`'s `helm-lint` job, point at
`charts/facility-layout`). Verify locally with `ct lint --charts
charts/facility-layout --validate-maintainers=false
--check-version-increment=false` before pushing. Add `helm-lint` to
`docker-publish`'s `needs` list.

## Task 10 — BDD (godog/Gherkin) acceptance tests

Same as the other four repos' BDD addition: godog
(`github.com/cucumber/godog`), `features/*.feature` files exercising the
REAL REST API end-to-end via `httptest.NewServer` over the in-memory
adapters (same wiring pattern as e.g.
`../inventory-storage/features_test.go` — read it for the exact structure).
Cover at minimum:
- Registering a full chain (site -> zone -> aisle -> slot) succeeds
- Registering a slot against an unknown/decommissioned parent is rejected
- Registering a slot that violates a PlacementRule is rejected
- Duplicate LocationCode registration is rejected
- Importing a layout reports partial success (mix of good/bad rows)
- GET layout and GET grid return the expected nested/matrix shape

Add a `bdd` CI job (copy from any of the other four repos' `bdd` job,
identical shape: `go test ./... -run TestFeatures -v`). Add `bdd` to
`docker-publish`'s `needs` list.

## Task 11 — Docker Hub publish job

Same as the other four repos' Task 13: `docker-publish` job in
`.github/workflows/ci.yml`, gated on `needs: [lint, test, bdd, integration,
api-lint, helm-lint, arch-test]` (the full list, once Tasks 6-10 are all
done) and `if: github.event_name == 'push' && github.ref ==
'refs/heads/main'`. Builds and pushes this repo's Dockerfile to Docker Hub
under the `claudioed` namespace, tagged `latest` + short git SHA,
linux/amd64, GitHub Actions cache. Requires `DOCKERHUB_USERNAME` /
`DOCKERHUB_TOKEN` repo secrets (same secrets already configured on the
other four repos under this account — if they're already set at the org/
account level this may just work; if not, note it explicitly rather than
silently skip).

## Task 12 — Verify, commit, push

- Full suite green: `go build ./...`, `go vet ./...`, `go test ./...
  -race`, `golangci-lint run ./...`, `gofmt -l .` clean, coverage >= 90%,
  `spectral lint` clean, `ct lint` clean, BDD scenarios all passing,
  arch-test passing.
- `git init`, initial commit, `git remote add origin
  git@github.com:claudioed/facility-layout.git` (repo already created and
  cloned empty — check `git remote -v`, it may already be set from the
  clone), push to `main`.
- Push and confirm the real GitHub Actions run goes green via `gh run
  watch` — this is the strongest verification, do it.
- Final report: every file created, every stage's Definition of Done
  confirmed with real command output (not paraphrased), the real `gh run
  view --json jobs` output showing every job green, and the actual `curl`
  output for the worked example in the README (site -> zone -> aisle ->
  slot -> layout -> grid).
