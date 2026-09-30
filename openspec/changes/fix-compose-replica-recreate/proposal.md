## Why

Update tasks are built per container, but `docker compose up -d --force-recreate <service>` recreates the *whole service*. A service scaled to N replicas produces N tasks that each invoke the same recreate for the same project+service: N redundant service recreations per update, causing needless container churn and start-order races.

## What Changes

- Selected rows that belong to the same compose project+service collapse into a single update task; the service is recreated exactly once per apply run.
- Sibling replica rows mirror the shared task's status (same one-line pipeline state) instead of running independent pipelines.
- In-flight counting is corrected to count tasks, not rows sharing a task.
- Standalone containers are unaffected (one task per container, as today).

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `update-execution`: the "Compose service restart" requirement changes — one recreate per compose project+service per apply run, with replica rows sharing one task.

## Impact

- **Code**: `internal/tui/updates.go` (task construction in `applySelectedUpdates`, in-flight accounting in `applyUpdateEvent`), TUI model tests. `internal/updater` is untouched (it already runs whatever tasks it is given).
- **Behavior**: applying an update to any replica of a compose service recreates the whole service once — this is what compose does regardless, so no behavioral surprise, only fewer redundant invocations.
