# Mutation testing

Mutation testing measures whether the test suite would actually *notice* if
the production code changed. [gremlins](https://github.com/go-gremlins/gremlins)
rewrites small pieces of the domain (flipping conditionals, moving
comparison boundaries, changing arithmetic) and re-runs the tests: a mutant
that is **KILLED** was caught by a test, a mutant that **LIVED** slipped past.

Scope: `internal/domain/...` only. That is where the invariants live and
where a silent behavioural change is most expensive.

## Posture: exploratory, never blocking

Same as the other four warehouse-systems services. The `mutation` job in
`.github/workflows/ci.yml` is gated to `workflow_dispatch` and the weekly
schedule — it never runs on a push or a PR, and it never blocks a merge. It
is a periodic quality signal to triage, not a gate. Coverage (>= 90%) and
`golangci-lint` are the blocking gates.

## Running it

```sh
go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
gremlins unleash ./internal/domain --workers 1 --timeout-coefficient 30
```

## Baseline result

```
Mutation testing completed in 51 seconds 661 milliseconds
Killed: 89, Lived: 10, Not covered: 0
Timed out: 0, Not viable: 0, Skipped: 0
Test efficacy: 89.90%
Mutator coverage: 100.00%
```

**Mutator coverage 100% / Not covered 0** is the number that matters most:
every mutable statement in the domain is reached by at least one test. There
is no untested corner of the domain hiding behind an aggregate coverage
percentage.

## Triage of the 10 survivors

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
coverage.
