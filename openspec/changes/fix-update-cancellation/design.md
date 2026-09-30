## Context

Two gaps around quitting mid-update:

1. `handleKey` (`internal/tui/model.go`) routes `q` through the confirmation dialog when `m.inFlight > 0`, but `ctrl+c` returns `tea.Quit` unconditionally.
2. `applySelectedUpdates` (`internal/tui/updates.go`) starts the runner with `context.Background()`. Nothing cancels pulls (`ImagePull` HTTP stream), engine calls, or `exec.CommandContext` provider runs on quit — the process just exits and abandons them.

## Goals / Non-Goals

**Goals:**
- One cancellable context per apply run, owned by the model.
- Confirm-quit cancels the run; the program may exit without waiting for workers to finish unwinding.
- `ctrl+c` and `q` share one quit path.

**Non-Goals:**
- A pause/cancel button that stops the run but keeps the TUI open (possible follow-up; the context plumbing added here makes it trivial later).
- Rollback of half-applied restarts on cancel (a cancelled recreate may leave a stopped container, same as a failed restart today — manual recovery path already exists).

## Decisions

**Model owns `updateCtx context.Context` + `updateCancel context.CancelFunc`.** `applySelectedUpdates` creates them via `context.WithCancel(context.Background())` and passes the context to `runner.Run`. Confirmed quit calls `updateCancel()` then `tea.Quit`. A new run overwrites the pair (old runs are always finished by then, since `updRunning` gates `enter`).

**Unify the quit paths.** Extract the in-flight check: both `q` and `ctrl+c` go to the confirmation dialog when `m.inFlight > 0`, otherwise quit directly. The dialog's `y` handler cancels and quits.

**Cancellation surfaces as a classified error, not a hang.** The engine client and compose provider already accept contexts, so cancellation propagates naturally as `context.Canceled`. In `runTask`, wrap phase errors: when `ctx.Err() != nil` after a phase error, emit `EventFailed` with an `update cancelled` error. Queued tasks: `runTask` checks `ctx.Err()` at entry and short-circuits to cancelled. This keeps the events channel closing normally (`wg.Wait()` returns because workers never block on a dead context: `emit` already selects on `ctx.Done()`).

**Trade-off accepted: exit doesn't wait for unwind.** After `updateCancel()`, `tea.Quit` exits immediately; workers may still be closing the pull stream for a few ms. The alternative (await `events` channel close before quitting) delays exit and can hang if an engine call ignores cancellation; not worth it — the daemon side of a cancelled pull is harmless.

## Risks / Trade-offs

- [Killing `docker compose up` mid-recreate leaves a stopped container] → Same recovery story as a failed restart today (documented in README); acceptable.
- [Cancelled task rows show `failed: update cancelled` after the TUI is gone] → Irrelevant at exit, but correct if a future "cancel run, stay open" feature reuses the plumbing.
- [Context leak if a run never starts] → `applySelectedUpdates` returns early without creating the context when no rows are checked; cancel is idempotent.
