## Why

Every engine pull event becomes one `updater.Event`, one `tea.Msg`, and one full TUI render. A multi-layer image emits hundreds of progress events per second per task; the 64-slot channel buffer fills, `emit` blocks the worker, and render latency starts throttling the actual download bookkeeping. The UI can neither display nor the user perceive more than a handful of updates per second.

## What Changes

- Pull-progress events are coalesced per task with latest-wins semantics and emitted at a bounded rate (e.g. at most one per task per ~100ms).
- Phase changes, completion, and failure events are never throttled or dropped — only byte/layer progress samples.
- Terminal progress state is guaranteed visible: a task's final progress sample is emitted before its phase change out of `pulling`.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `update-execution`: the "Pull progress reporting" requirement changes — progress samples are rate-bounded with latest-wins coalescing, with terminal state preserved.

## Impact

- **Code**: `internal/updater/updater.go` (per-task progress throttle inside `runTask`'s progress callback).
- **Behavior**: progress bars update at human-perceivable cadence; worker pool no longer couples download speed to TUI render speed.
- **Tests**: event-stream tests need to expect fewer, coalesced progress events.
