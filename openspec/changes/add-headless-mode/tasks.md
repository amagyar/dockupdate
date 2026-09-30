## 1. Headless package

- [ ] 1.1 Create `internal/headless` with `RunCheck(ctx, opts) (report, exitCode)`: connect via `engine.Connect`, list containers, filter to updatable running ones, run concurrent bounded digest checks with the existing 60s per-check timeout
- [ ] 1.2 Factor the update-eligibility predicate (managed-workload + running-state rules) into a shared helper used by both `internal/tui` and `internal/headless`
- [ ] 1.3 Render the report: one aligned line per container (`name  image  outcome`), sorted by name; errors to stderr; "no updatable containers" message when empty
- [ ] 1.4 Implement exit-code algebra: 2 on connect failure or any failed check, else 1 when updates are available, else 0

## 2. Entrypoint

- [ ] 2.1 Add the `--check` flag in `cmd/dockupdate/main.go` and dispatch to `headless.RunCheck`; document that `--concurrency`/`--prune` are TUI-only
- [ ] 2.2 Update `--help` output and the README usage table + a cron example

## 3. Tests

- [ ] 3.1 Unit tests for `RunCheck` against a fake engine + httptest registry: mixed outcomes, exit codes 0/1/2, stable sorted output
- [ ] 3.2 Unit test: engine-unreachable path exits 2 with probed locations on stderr
- [ ] 3.3 Verify `go vet ./... && go test ./...` and the race detector stay green
