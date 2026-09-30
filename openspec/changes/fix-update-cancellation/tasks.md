## 1. Runner cancellation semantics

- [ ] 1.1 In `internal/updater/updater.go`, short-circuit `runTask` at entry when `ctx.Err() != nil`, emitting `EventFailed` with an `update cancelled` error
- [ ] 1.2 After each phase error in `runTask`, reclassify to `update cancelled` when `ctx.Err() != nil` (pull/verify/restart)

## 2. TUI context ownership

- [ ] 2.1 Add `updateCtx`/`updateCancel` fields to the model; create them in `applySelectedUpdates` and pass the context to `runner.Run` instead of `context.Background()`
- [ ] 2.2 Unify quit handling: `ctrl+c` takes the same path as `q` (confirmation when `m.inFlight > 0`, immediate quit otherwise)
- [ ] 2.3 On confirmed quit (`y`), call `updateCancel()` before returning `tea.Quit`

## 3. Tests

- [ ] 3.1 Updater test: cancelling the context mid-pull ends the task as failed with the cancellation reason and closes the events channel
- [ ] 3.2 Updater test: tasks queued behind the worker pool terminate as cancelled without invoking the engine
- [ ] 3.3 TUI model test: `ctrl+c` with `inFlight > 0` opens the confirmation dialog instead of quitting; with `inFlight == 0` it quits
- [ ] 3.4 TUI model test: confirming the dialog calls the cancel function
