## Why

`RecreateContainer` silently loses user configuration and removes the rollback path too early. Per-network endpoint settings are replaced with empty structs, dropping static IPs, network aliases, and links; and the old container is deleted immediately after `ContainerStart` returns — before anyone knows whether the new container stays up. A container that crashes two seconds into its new image leaves the user with no working service and no rollback target (the code comment even claims the old one is kept "once the replacement is healthy", but no health check exists).

## What Changes

- Standalone recreation preserves per-network endpoint configuration: static IP assignments (IPAM config), aliases, and links from the old container's inspect record; only runtime-assigned fields (endpoint IDs, runtime MAC/IPv6 addresses) are dropped.
- After start, the replacement must prove itself before the old container is removed: containers with a healthcheck must reach `healthy`; others must still be running after a short grace period.
- If the replacement fails the gate, the update fails and the system attempts an automatic rollback: remove the broken replacement, restore the old container's name, and start it again.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `update-execution`: the "Standalone container recreation" requirement changes — endpoint settings are preserved, a post-start health gate guards old-container removal, and failure triggers best-effort auto-rollback.

## Impact

- **Code**: `internal/engine/engine.go` (`RecreateContainer`, new endpoint-settings sanitization and a health-wait helper), engine live tests, updater integration (no interface change: rollback happens inside `RecreateContainer`).
- **Behavior**: slower worst-case standalone update (health gate, bounded by a timeout) in exchange for not bricking the service.
- **Edge case**: static IPs can conflict if the old container is still attached — the old container is stopped first (as today), so its addresses are free.
