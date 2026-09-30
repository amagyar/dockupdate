## Why

`dockupdate` is TUI-only, which excludes the most requested usage pattern for an update checker: running unattended from cron/systemd/CI to answer "do any of my containers have updates?" and act on the answer (notify, open the TUI, file a ticket). The check pipeline is already UI-independent — only the entrypoint couples it to Bubble Tea.

## What Changes

- New `--check` flag: connect to the engine, run registry digest checks for all updatable running containers, print a human-readable report to stdout, and exit — no TUI.
- Exit codes make the result scriptable: `0` = no updates available, `1` = updates available, `2` = error (engine unreachable, check failures).
- The report lists each container with its outcome (update available / up to date / local build / pinned / failed), one per line.

## Capabilities

### New Capabilities

- `headless-cli`: non-interactive execution mode that runs update checks and reports results via stdout and process exit codes, reusing the engine, registry, and inventory internals without starting the TUI.

### Modified Capabilities

<!-- None — the TUI flow is untouched; `update-checking` requirements already cover the check semantics this mode reuses. -->

## Impact

- **Code**: `cmd/dockupdate/main.go` (flag + mode dispatch), new `internal/headless` (or a small file in `cmd`) orchestrating connect → list → check → report. Reuses `internal/engine`, `internal/registry`; no changes to `internal/tui`.
- **Flags**: `--check` composes with `--socket`; `--concurrency`/`--prune` remain TUI-update concerns and are ignored in check mode.
- **Docs**: README usage section gains the new flag and a cron example.
- **Non-goal (explicit)**: headless *applying* of updates (`--apply`) and machine-readable output (`--json`) — both deliberate follow-ups once check-mode usage validates the shape.
