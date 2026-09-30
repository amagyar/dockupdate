## Context

`applySelectedUpdates` (`internal/tui/updates.go`) creates one `updater.Task` per checked row. For compose-managed containers, the restart step shells out to `<provider> up -d --force-recreate <service>`, which recreates **every replica** of the service. With N replicas checked, the same service is recreated N times: redundant work, image pulls shared by luck only (same ref, engine-side cache dedupes), and recreate races.

## Goals / Non-Goals

**Goals:**
- One compose recreate per project+service per apply run.
- All replica rows visibly track the update (shared one-line status).
- No changes to `internal/updater`'s runner/state machine.

**Non-Goals:**
- Grouping the *pull* phase across tasks (each task still pulls its own ref; identical refs are engine-cached).
- Changing standalone (non-compose) update behavior.
- Cross-run dedup (e.g. remembering a service was already recreated earlier in the session).

## Decisions

**Deduplicate at task construction (TUI layer), not in the runner.** `applySelectedUpdates` groups checked compose rows by `(project, service)` and creates one task per group; the runner stays a dumb executor. Alternative considered: a merge pass inside `updater.Runner` — rejected because the runner would need compose semantics and the TUI would still have to reconcile multiple rows with one surviving task.

**Replica rows share the group's `*updater.Task` pointer.** The first row (in row order) of a group is the *owner*; its container ID becomes `Task.ID`. Sibling rows get the same pointer, so `applyUpdateEvent` (which matches `r.task.ID == ev.TaskID`) updates every replica row at once, and `taskStatus` renders identically for all of them.

**Fix in-flight accounting to count owner rows only.** Today `m.inFlight--` fires per row whose task transitions to a terminal phase; with a shared pointer it would fire once per replica and drive the counter negative. Decrement only when the row's container ID equals the event's `TaskID` (the owner), restoring 1 decrement per task.

**Selection semantics follow compose reality.** Checking any subset of a service's replica rows recreates the whole service (that is what `up --force-recreate` does); unchecked replica rows simply don't display the pipeline state. Documenting this in the spec avoids implying per-replica granularity that compose cannot provide.

## Risks / Trade-offs

- [A sibling row outlives its container] → Compose recreates all replicas with new IDs; the post-run inventory refresh (`updatesFinishedMsg` → reload) already rebuilds rows, so stale sibling rows are replaced.
- [User confusion: unchecked replica also restarted by compose] → Accepted: this is inherent compose behavior, and the spec now states it explicitly.
- [Shared pointer mutated by event application from multiple rows] → `Task.Apply` is idempotent per event and applied sequentially on the single-threaded TUI update loop; the loop `break`s per row, and repeated application of the same event to the same pointer is a no-op state-wise (same fields assigned).
