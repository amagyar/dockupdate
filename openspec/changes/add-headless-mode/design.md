## Context

`main.go` dispatches straight into `tui.Run`. The check pipeline (`engine.Connect` → `ListContainers` → `registry.Checker.Check`) has no TUI dependency except living inside `tui` commands; `updater.Runner` already proves the internals run headless.

## Goals / Non-Goals

**Goals:**
- `dockupdate --check` runs the full check pass and exits with a scriptable code.
- Zero changes to TUI behavior; shared logic is extracted, not duplicated.
- Runtime stays snappy: checks run concurrently with the same per-check 60s timeout as the TUI.

**Non-Goals:**
- `--apply`/headless updates: restart semantics (compose detection, rollback, prune) deserve their own change once check mode is adopted.
- `--json`/machine-readable output: design the format when a consumer exists; exit codes already cover scripting.
- Notifications (ntfy/mail/webhook): deliberately left to wrapper scripts.

## Decisions

**New package `internal/headless` with one entrypoint: `RunCheck(ctx, opts) (report, code)`。** `main.go` becomes a thin dispatcher: `--check` → `headless.RunCheck`, otherwise TUI. The package depends on `engine` and `registry` only — testable with the existing fake-engine pattern, no Bubble Tea import.

**Concurrent checks with an errgroup-free fan-out.** One goroutine per container, bounded by a small fixed limit (not `--concurrency`, which is pull-bandwidth semantics), each with the 60s timeout, results collected and sorted by container name for stable output. Alternative considered: sequential checks — rejected; 30 containers × slow registry would serialize into minutes.

**Classification reuse, not reimplementation.** Managed/non-running exclusion mirrors `rebuildUpdateRows`'s rules; factor the eligibility predicate (`Managed()`, state check) so TUI and headless share one source of truth. Pinned/local-build/failed classification stays inside `registry.Checker`.

**Exit-code algebra:** `2` if connect fails or any check returns `KindFailed`; else `1` if any `KindUpdateAvailable`; else `0`. Documented in `--help` and README.

## Risks / Trade-offs

- [Cron runs hit Docker Hub rate limits daily] → Mitigated by the dedupe-update-checks change if both land; otherwise documented as a caveat for anonymous Hub usage.
- [Output format bikeshedding] → One-line-per-container `name  image  outcome`, aligned columns; simple to parse with awk, explicitly not a stable machine API yet.
- [Windows service/scheduled-task usage] → Works as-is once windows-engine-detection lands; until then `DOCKER_HOST` covers it.
