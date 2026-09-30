## 1. Task construction dedup

- [ ] 1.1 In `applySelectedUpdates` (`internal/tui/updates.go`), group checked compose rows by `(project, service)`; create one `updater.Task` per group (owner = first row in row order, its container ID as `Task.ID`) and assign the shared pointer to every replica row in the group
- [ ] 1.2 Keep per-container tasks for standalone rows and for compose groups of size 1 (no behavior change there)

## 2. In-flight accounting

- [ ] 2.1 In `applyUpdateEvent` (`internal/tui/model.go`), decrement `m.inFlight` only when the row's container ID equals the event's `TaskID`, so a shared task completes the counter exactly once

## 3. Tests

- [ ] 3.1 Unit test: 3 checked replicas of one service produce 1 task, all 3 rows share the pointer, and a `Done` event leaves `inFlight` at 0 (not negative)
- [ ] 3.2 Unit test: two different services in one project still produce two tasks; standalone containers are unaffected
- [ ] 3.3 Verify the fake-compose `updater` tests still pass unchanged (runner untouched)

## 4. Docs

- [ ] 4.1 README "Compose-aware restarts" line: note that applying any replica recreates the whole service once
