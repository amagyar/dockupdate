## Context

`runTask`'s progress callback emits one `EventPullProgress` per `jsonstream.Message`. Docker emits several messages per layer per second; the TUI consumes one event per `Update` cycle and re-renders each time. The event channel (64 buffer) and the render loop become a backpressure path onto the worker goroutine.

## Goals / Non-Goals

**Goals:**
- Bound progress-event frequency per task; keep every non-progress event lossless.
- Worker never blocks on progress delivery (progress is droppable-by-design data).
- Minimal complexity inside `runTask` — no extra goroutines per task.

**Non-Goals:**
- Changing the byte/layer aggregation itself (`LayerAggregator` stays).
- Adaptive/debounce-based UI refresh (fixed-rate sampling is sufficient and predictable).

## Decisions

**Throttle at the emission point, synchronously.** In `runTask`, wrap the progress callback with a per-task gate: remember the last emitted sample and the last emit time; emit immediately if ≥100ms since the previous sample, otherwise store the sample as pending. A trailing flush happens right after `PullImage` returns (emit the pending sample if any) before the `PhaseVerifying` phase-change event — satisfying the "final sample precedes phase change" requirement with no timers.

**Why not a ticker goroutine per task:** a time-based flusher adds a goroutine and a mutex per task to save nothing — the synchronous gate plus trailing flush is ~15 lines, race-free (the callback runs on the pull-reading goroutine only), and trivially testable with a fake clock.

**Phase/done/fail events bypass the gate entirely** through the existing `emit` helper; only `EventPullProgress` passes through the sampler.

**Clock injection for tests:** the gate takes a `now func() time.Time` (defaults to `time.Now`) so tests can advance time deterministically without sleeping.

## Risks / Trade-offs

- [Progress appears steppier at 10Hz] → Imperceptible in practice; spinners already tick at a similar cadence.
- [A pull shorter than 100ms shows only the final sample] → Acceptable: row still lands on the correct final state; such pulls are cache-hits anyway.
- [Future per-task goroutine assumptions] → The gate's state is stack-local to `runTask`; no shared mutable state introduced.
