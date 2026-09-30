## Why

Every container gets its own registry check, and each check issues a HEAD plus (for multi-arch tags) a manifest GET. Ten containers running `nginx:1.25` produce ten identical remote round-trips. Docker Hub counts manifest reads against its pull rate limit (100/6h anonymous, 200/6h authenticated), so a busy host can turn "everything works" into `check failed: toomanyrequests` for no benefit — the remote digest for a given ref is identical for every container in the same check pass.

## What Changes

- Within one check pass (startup or manual refresh), remote registry lookups are deduplicated by image reference: the first check for a ref performs the HEAD/manifest requests; concurrent and subsequent checks for the same ref share the result.
- Per-container comparison stays local and per-image: containers on the same ref but with different local image IDs still get individually correct outcomes.
- The cache is scoped to a single pass and discarded on the next refresh — no stale results across presses of `r`.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `update-checking`: new requirement — remote digest lookups are shared per image reference within a check pass, without changing per-container classification.

## Impact

- **Code**: `internal/registry/registry.go` (split remote lookup from comparison; add a per-pass memoizing wrapper), `internal/tui/model.go` (`checkCmd` shares one checker/cache per pass).
- **Dependencies**: none new.
- **Ops**: materially fewer registry requests on hosts with many same-image containers; less exposure to Docker Hub rate limiting.
