#!/usr/bin/env python3
"""scripts/coverage-quality.py — the "green tests that assert nothing"
detector (harness-coverage-expansion plan, Phase 5 Task 5.1).

Cross-references `go tool cover -func` per-function line coverage against
gremlins' LIVED-mutant report for the same package tree. A function at or
above the coverage threshold (default 90%, this repo's own gate) that
still has a LIVED mutant on one of its lines is exactly the failure mode
mutation testing exists to catch: the line executed, but nothing asserted
on the result. Flags those functions; does not touch anything else
gremlins already reports (NOT_COVERED functions are gremlins' own job,
not this script's).

This is a WEEKLY, ADVISORY sensor (see ci.yml's `drift` job) — it never
blocks a PR. Known accepted survivors from MUTATION.md (tie-break /
near-equivalent mutants, see the how-to-test skill) are expected to keep
showing up here; the point is visibility over time, not a new gate to
satisfy on every commit.

Usage:
    go test ./internal/domain/... -coverprofile=/tmp/cov.out -covermode=atomic
    gremlins unleash ./internal/domain --workers 1 --timeout-coefficient 30 > /tmp/gremlins.log 2>&1 || true
    python3 scripts/coverage-quality.py /tmp/cov.out /tmp/gremlins.log
"""
from __future__ import annotations

import subprocess
import re
import sys
from dataclasses import dataclass, field

COVERAGE_THRESHOLD = 90.0

LIVED_RE = re.compile(r"^\s*LIVED\s+\S+\s+at\s+(\S+):(\d+):(\d+)\s*$")
FUNC_RE = re.compile(
    r"^(?P<path>\S+):(?P<line>\d+):\s+(?P<func>\S+)\s+(?P<pct>[\d.]+)%\s*$"
)


@dataclass
class FuncCoverage:
    file_suffix: str
    start_line: int
    name: str
    pct: float
    lived_lines: list[int] = field(default_factory=list)


def parse_cover_func(cover_out: str) -> list[FuncCoverage]:
    funcs: list[FuncCoverage] = []
    for line in cover_out.splitlines():
        m = FUNC_RE.match(line.strip())
        if not m or m.group("func") == "total:":
            continue
        funcs.append(
            FuncCoverage(
                file_suffix=m.group("path"),
                start_line=int(m.group("line")),
                name=m.group("func"),
                pct=float(m.group("pct")),
            )
        )
    return funcs


def parse_lived(gremlins_out: str) -> list[tuple[str, int]]:
    hits: list[tuple[str, int]] = []
    for line in gremlins_out.splitlines():
        m = LIVED_RE.match(line)
        if not m:
            continue
        path, lineno, _col = m.groups()
        hits.append((path, int(lineno)))
    return hits


def _match_key(path: str) -> str:
    """Reduce a file path to '<parent-dir>/<filename>' for matching across
    tools that report paths relative to different working directories
    (gremlins reports relative to the package root it was invoked on;
    `go tool cover -func` reports full module import paths)."""
    parts = path.replace("\\", "/").split("/")
    return "/".join(parts[-2:]) if len(parts) >= 2 else path


def nearest_func_for_line(funcs: list[FuncCoverage], file_suffix: str, line: int) -> FuncCoverage | None:
    """A mutant's line belongs to whichever function in the same file
    starts at the highest start_line that is still <= the mutant's line.
    go tool cover -func only gives us each func's START line, not its
    span, so this is an approximation — correct as long as functions in
    a file are listed in source order, which `go tool cover -func` does."""
    key = _match_key(file_suffix)
    same_file = [f for f in funcs if _match_key(f.file_suffix) == key]
    same_file = [f for f in same_file if f.start_line <= line]
    if not same_file:
        return None
    return max(same_file, key=lambda f: f.start_line)


def main() -> int:
    if len(sys.argv) != 3:
        print(f"usage: {sys.argv[0]} <cover.out> <gremlins.log>", file=sys.stderr)
        return 2

    cover_path, gremlins_path = sys.argv[1], sys.argv[2]

    cover_func = subprocess.run(
        ["go", "tool", "cover", "-func=" + cover_path],
        capture_output=True, text=True, check=True,
    ).stdout
    funcs = parse_cover_func(cover_func)

    with open(gremlins_path, encoding="utf-8", errors="replace") as fh:
        gremlins_out = fh.read()
    lived = parse_lived(gremlins_out)

    flagged: dict[str, FuncCoverage] = {}
    for file_suffix, line in lived:
        f = nearest_func_for_line(funcs, file_suffix, line)
        if f is None:
            continue
        if f.pct >= COVERAGE_THRESHOLD:
            key = f"{f.file_suffix}:{f.name}"
            flagged.setdefault(key, f).lived_lines.append(line)

    if not flagged:
        print("coverage-quality: no high-coverage functions with surviving mutants found. clean.")
        return 0

    print(f"coverage-quality: {len(flagged)} function(s) at >={COVERAGE_THRESHOLD:.0f}% line "
          f"coverage still have a LIVED (surviving) mutant — tests execute the line but "
          f"do not assert on it:\n")
    for key, f in sorted(flagged.items()):
        lines = ", ".join(str(l) for l in sorted(set(f.lived_lines)))
        print(f"  {f.file_suffix}:{f.name} ({f.pct:.1f}% covered) — surviving mutant(s) at line(s) {lines}")
    print(
        "\nCross-check each against MUTATION.md's accepted-survivor list before treating "
        "as a real gap — tie-break/near-equivalent mutants are expected to appear here "
        "(see the how-to-test skill). This job is advisory: it does not fail the build."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
