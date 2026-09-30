## 1. Sampler

- [ ] 1.1 Add a per-task progress sampler in `internal/updater` (latest-wins, ≥100ms between samples, injectable clock), emitting through the existing `emit` helper
- [ ] 1.2 Wire it into `runTask`'s progress callback; flush the pending sample after `PullImage` returns, before emitting the `PhaseVerifying` phase change
- [ ] 1.3 Confirm phase/done/fail events bypass the sampler

## 2. Tests

- [ ] 2.1 Unit test: burst of progress callbacks within one interval yields one sample; advancing the fake clock yields the next latest sample
- [ ] 2.2 Unit test: trailing flush emits the final sample before the phase-change event in the recorded event stream
- [ ] 2.3 Update existing updater tests/fixtures that asserted per-event progress passthrough
- [ ] 2.4 `go test -race ./internal/updater/` green
