## Why

`q` asks for confirmation while updates are in flight, but `ctrl+c` quits instantly — and even the confirmed quit does nothing to stop the work: update tasks run on `context.Background()`, so pulls and compose recreates continue blindly while the process exits. There is no cancellation path anywhere in the update pipeline.

## What Changes

- `ctrl+c` behaves like `q`: while updates are in flight it opens the same quit-confirmation dialog instead of quitting immediately.
- Each apply run gets a cancellable context owned by the TUI model; confirming quit cancels in-flight pulls, verifications, and provider invocations before exit.
- Cancelled tasks end in a defined terminal state (`failed` with a cancellation reason) rather than hanging the worker pool or corrupting row state.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `update-execution`: new requirement — update runs are cancellable; cancellation stops in-flight work and terminates tasks with a cancellation state.
- `tui-layout`: the "Quit confirmation during updates" requirement changes — `ctrl+c` is covered by the same confirmation as `q`, and confirming cancels in-flight work.

## Impact

- **Code**: `internal/tui/updates.go` (`applySelectedUpdates` creates/owns the context), `internal/tui/model.go` (key handling, quit confirmation, cancel on quit), `internal/updater/updater.go` (context already threaded through; ensure `runTask` aborts between phases and classifies `context.Canceled`).
- **Dependencies**: none new.
- **Behavior**: quitting mid-update now actively stops engine pulls and compose provider processes instead of abandoning them.
