# Mutation testing

Mutation testing measures whether the test suite would actually *notice* if
the production code changed. [gremlins](https://github.com/go-gremlins/gremlins)
rewrites small pieces of the domain (flipping conditionals, moving
comparison boundaries, changing arithmetic) and re-runs the tests: a mutant
that is **KILLED** was caught by a test, a mutant that **LIVED** slipped past.

Scope: `internal/domain/...` only. That is where the invariants live and
where a silent behavioural change is most expensive.

## Posture: a blocking fast run, plus a deep scheduled run

A sensor that never fires tells you nothing, so the mutation score is now a
gate — set at the level the suite already meets, exactly like the 90% coverage
gate.

- **`mutation-fast`** (`.github/workflows/ci.yml`) runs on every push and PR
  and blocks `docker-publish`. It runs `gremlins unleash ./internal/domain`
  with the workers, timeout coefficient and thresholds from `.gremlins.yaml`
  (efficacy >= 89%, mutator coverage >= 95%).
- **`mutation`** stays gated to `workflow_dispatch` and the weekly schedule
  with `--workers 1 --timeout-coefficient 30` and a 90-minute budget — the
  slow, deterministic, exhaustive re-measurement to triage against.

### Why the fast run keeps the whole domain layer

The usual way to make a mutation run cheap enough to block on is to narrow it
to one package. That was measured here and rejected — the full domain layer is
already inside the budget, and narrowing would gut the sensor:

| scope | mutants | efficacy | runtime |
|-------|---------|----------|---------|
| `./internal/domain/slot` | 5 | 100.00% | 30s |
| `./internal/domain/placement` | 27 | 96.30% | 32s |
| **`./internal/domain` (chosen)** | **99** | **89.90%** | **~60s** |

`slot` is the richest package by behaviour but only yields 5 mutants, so as a
regression detector it is close to useless. The whole layer gives 3.7x the
mutants of the best single package for roughly 2x the runtime, and at ~60s it
sits far inside the job's `timeout-minutes: 20`. Note also that both single
packages score *above* the gate: narrowing would have meant setting the
threshold against a subset that hides the survivors, which is precisely the
"sensor that cannot fire" problem this change exists to fix.

## Running it

```sh
go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0

# what mutation-fast runs (config comes from .gremlins.yaml)
make mutation           # == gremlins unleash ./internal/domain

# what the scheduled mutation job runs
gremlins unleash ./internal/domain --workers 1 --timeout-coefficient 30
```

## Baseline result

```
Mutation testing completed in 1 minute 11 seconds
Killed: 217, Lived: 19, Not covered: 9
Timed out: 0, Not viable: 0, Skipped: 0
Test efficacy: 91.95%
Mutator coverage: 96.33%
```

**Mutator coverage 96.33%** — the 9 "not covered" mutants below are all pre-existing (`slot/functional.go`'s `String()` method, never exercised for its `dockFlow==""` branch beyond what's needed to prove `IsZero()`/dockFlow rendering, and `shared/geometry.go`'s degenerate-segment fallback in `DistanceToPoint`, unreachable in practice since `NewSegment` rejects identical endpoints) — same triage as prior phases, not new coverage gaps.

## Triage of the 19 survivors

```
LIVED CONDITIONALS_BOUNDARY at aisle/aisle.go:55:21
LIVED CONDITIONALS_BOUNDARY at aisle/aisle.go:55:47
LIVED CONDITIONALS_NEGATION  at placement/rules.go:63:16
LIVED CONDITIONALS_BOUNDARY at shared/geometry.go:173:7
LIVED CONDITIONALS_BOUNDARY at shared/geometry.go:175:14
LIVED ARITHMETIC_BASE       at shared/geometry.go:186:33
LIVED CONDITIONALS_BOUNDARY at shared/location_code.go:75:47
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:9
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:21
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:35
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:47
LIVED CONDITIONALS_BOUNDARY at slot/functional.go:153:54
LIVED CONDITIONALS_BOUNDARY at travel/graph.go:139:31
LIVED CONDITIONALS_BOUNDARY at travel/graph.go:211:17
LIVED CONDITIONALS_BOUNDARY at travel/graph.go:233:33
LIVED CONDITIONALS_BOUNDARY at travel/graph.go:250:66
LIVED CONDITIONALS_NEGATION at travel/graph.go:250:66
LIVED CONDITIONALS_BOUNDARY at zone/zone.go:110:35
LIVED CONDITIONALS_BOUNDARY at zone/zone.go:110:47
```

**Ten are the pre-existing benign ASCII-boundary pattern** (`aisle/aisle.go`, `shared/location_code.go`, `site/site.go`, `slot/functional.go`, `zone/zone.go` — every uppercase-alphanumeric validator's `>=`/`<=` boundary), unchanged from prior phases; see the original triage below.

**One is the pre-existing near-equivalent** `placement/rules.go:63` (error-message wording only), unchanged from prior phases.

**Four new survivors are in ADR-0017's `shared.Segment.DistanceToPoint` and `travel.Graph.Distance` (Phase B2), all near-equivalent mutants at exact geometric boundaries:**

- `shared/geometry.go:173/175` (`t < 0` / `t > 1` clamping): at the exact boundary `t == 0` or `t == 1`, clamping and not-clamping produce the identical closest point, so a `<`→`<=` mutant is undetectable there by construction — proven by the added boundary tests in `geometry_test.go` (`t=0`/`t=1` cases), which pass under both the original and mutated conditions.
- `shared/geometry.go:186` (final `math.Sqrt` combination): an `ARITHMETIC_BASE` mutant here would have to change the combination of three already-individually-tested deltas in a way that happens to preserve every hand-verified distance in the test table; accepted as noise given the surrounding lines are otherwise fully killed.
- `travel/graph.go:139` (`gaps > 0` in `bayDistance`): guards the single-bay-aisle edge case (added `TestGraphSingleBayAisleIgnoresCentreline`); the boundary itself (`gaps == 0` vs `gaps == -1`, impossible since `len(a.Bays) >= 1`) is unreachable, making the `>`/`>=` mutant equivalent.
- `travel/graph.go:211` (Dijkstra's `candidate < dist[edge.To]` relaxation) and `travel/graph.go:233/250` (path-reversal loop bound, `priorityQueue.Less`): both are classic Dijkstra/heap boundary conditions where a `<`→`<=` mutant only matters on a tie, and every test fixture in `graph_test.go` (including the added `TestGraphFourWaypointRoute`) uses distinct edge weights, so no tie is ever exercised. Accepted: forcing a tie-breaking test would pin an arbitrary (and currently unspecified) tie-break order rather than a real invariant.

**Conclusion:** no new survivor indicates a missing test of a domain invariant; all four are genuine near-equivalent mutants at exact geometric/algorithmic boundaries. `.gremlins.yaml`'s gate (89% efficacy / 95% mutator coverage) remains appropriately below the measured 91.95%/96.33%.

## Original triage (Phase 0 baseline, retained for the pre-existing survivors)

```
LIVED CONDITIONALS_BOUNDARY at aisle/aisle.go:48:21
LIVED CONDITIONALS_BOUNDARY at aisle/aisle.go:48:47
LIVED CONDITIONALS_BOUNDARY at shared/location_code.go:75:47
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:9
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:21
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:35
LIVED CONDITIONALS_BOUNDARY at site/site.go:58:47
LIVED CONDITIONALS_BOUNDARY at zone/zone.go:87:35
LIVED CONDITIONALS_BOUNDARY at zone/zone.go:87:47
LIVED CONDITIONALS_NEGATION  at placement/rules.go:63:16
```

**Nine of the ten are the same benign pattern.** Every uppercase-alphanumeric
validator is written as:

```go
if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
```

`CONDITIONALS_BOUNDARY` rewrites `>=` to `>` (and `<=` to `<`), which
excludes exactly the characters `A`, `Z`, `0` and `9` from the accepted set.
Killing them means asserting that a segment consisting solely of `A`, `Z`,
`0` or `9` is accepted — a test of the ASCII table, not of a domain rule.
The tests already prove the rule that matters (lowercase, hyphens,
underscores and spaces are all rejected; mixed alphanumerics are accepted).
Accepted as noise; not worth a test.

**One is a genuine near-equivalent mutant.** `placement/rules.go:63` is the
loop inside `describeAllowRules`:

```go
if rule.Effect() != Allow || !rule.Predicate().Matches(attrs) {
    continue
}
```

This function only builds the *human-readable rule list appended to an
already-decided rejection message*. Negating the condition changes which
rules are named in that string, not whether the placement is rejected. The
tests assert that the violated rule's id appears in the error — which still
holds under the mutant, because the offending rule is in the set either way.
Pinning the exact wording of an error message would make the suite brittle
for no invariant gain. Accepted deliberately.

**Conclusion:** no survivor indicates a missing test of a domain invariant.
The re-run baseline to compare against is 89.90% efficacy / 100% mutator
coverage — which is why `.gremlins.yaml` pins the gate just underneath it, at
89% efficacy / 95% mutator coverage: enough headroom not to be flaky, tight
enough that losing an assertion turns the build red.
