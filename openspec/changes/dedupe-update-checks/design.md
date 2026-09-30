## Context

`checkCmd` (`internal/tui/model.go`) builds a fresh `registry.Checker` per container and calls `Checker.Check`, which always performs `remote.Head` (and `remote.Image` for indexes). The remote answer for a ref cannot differ between containers in the same pass; only the *local* side (repo digests per image ID) can.

## Goals / Non-Goals

**Goals:**
- One remote lookup per image ref per pass, including dedup of concurrent in-flight lookups.
- Per-container classification unchanged (local digests still compared individually).
- Zero persistence: cache lives and dies with the pass.

**Non-Goals:**
- Cross-pass or cross-session caching (ETag/If-Modified-Since handling) — complexity not justified for a TUI refresh action.
- Rate-limit *backoff*/retry policy (dedup removes the self-inflicted pressure; reacting to a 429 is a separate concern).

## Decisions

**Split `Checker.Check` into remote and pure local parts.** Keep the public `Check(ctx, ref, localDigests)` signature for compatibility, but factor internals: `remoteDigests(ctx, ref)` (already exists) for the network half and a pure `classify(ref, localDigests, candidates, primary)` for the comparison. The dedup wrapper only memoizes the network half.

**Singleflight-style memoizer owned by the model's check pass.** A small `lookupCache` struct: `map[string]*call` where `call` has `done chan struct{}`, result, and error. First caller for a ref marks it and does the lookup; others block on `done` (with ctx) and share the outcome. This is the standard `golang.org/x/sync/singleflight` shape implemented in ~20 lines — chosen over adding the dependency for one call site.

**Cache lifetime = one `startChecksCmd` batch.** The model creates the cache when it launches a pass and drops the reference afterward; the next pass builds a fresh one. No invalidation logic needed.

**Errors are shared, not retried per container.** A failed shared lookup yields the same `KindFailed` result for every container on that ref — matches the spec scenario and avoids thundering-herd retries against an already-struggling registry.

## Risks / Trade-offs

- [A slow shared lookup delays all rows on that ref] → Same total latency as today (those requests would have queued on the registry anyway); other refs still resolve in parallel.
- [Cache keyed by ref string ignores per-container auth differences] → Keyed auth comes from the user's keychain per registry, not per container; identical for a shared ref. Safe.
- [Memory: blocked goroutines on a never-closing `done`] → The owner always closes `done` (defer), even on panic-ish paths; waiters also select on their own ctx timeout (60s check timeout already exists).
