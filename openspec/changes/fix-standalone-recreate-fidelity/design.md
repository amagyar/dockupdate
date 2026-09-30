## Context

`RecreateContainer` (`internal/engine/engine.go`) currently:
- builds `networking.NetworkingConfig` with **empty** `EndpointSettings` per network and reconnects extra networks with empty settings — dropping static IPs (`IPAMConfig`), aliases, and links;
- removes the renamed `<name>-old-<ts>` container immediately after `ContainerStart` succeeds, despite its own comment claiming removal happens "once the replacement is healthy enough to keep running" — no such check exists.

## Goals / Non-Goals

**Goals:**
- Recreated standalone containers keep their network identity (static IP, aliases, links).
- The old container survives until the replacement is demonstrably viable.
- A failing replacement triggers best-effort automatic rollback.

**Non-Goals:**
- Health-gating *compose* service restarts (the provider owns recreation semantics there).
- Preserving runtime-assigned fields (endpoint ID, MAC, dynamic IPv6) — the engine must regenerate these.
- Making the health-gate timeout user-configurable (a sensible constant; flag can follow if asked).

## Decisions

**Sanitize, don't blank, endpoint settings.** Copy each network's `EndpointSettings` from inspect, then nil the runtime-only fields (`EndpointID`, `IPAddress`/`GlobalIPv6Address` runtime values as assigned — keep `IPAMConfig` which carries the *requested* static IPs, `Aliases`, `Links`, `DriverOpts`, `NetworkID`). The first network still goes into `ContainerCreate`, the rest via `NetworkConnect` — but now with the sanitized settings. Alternative considered: leaving everything empty and documenting the loss — rejected; silently changing a container's network identity is a behavioral break for users relying on static IPs (common for DNS-registered services).

**Health gate inside `RecreateContainer`, after start + network connects.** Poll `ContainerInspect` on an interval: if `Config.Healthcheck` exists → wait for `State.Health.Status == healthy`, fail fast on `unhealthy`; else → one grace delay (e.g. 2s) then require `State.Running`. Bounded by an overall timeout (e.g. 60s for the healthcheck case). The updater interface (`Engine.RecreateContainer`) does not change.

**Rollback = remove new, rename old back, start old.** On gate failure: `ContainerRemove(new)`, `ContainerRename(old, name)`, `ContainerStart(old)`. All best-effort: if rollback itself fails, report the original gate error plus the rollback error and leave both containers for manual recovery (old one still exists under its backup name). Alternative considered: keep the broken replacement for debugging — rejected; a broken running-config container holding the original name blocks the user's own retry.

**Why not fix it in `updater`:** the recreate/rollback sequence is one engine-level transaction; splitting it across the updater would leak engine details (backup naming, inspect polling) into the pipeline for no reuse benefit.

## Risks / Trade-offs

- [Healthchecks that take minutes] → Bounded wait (60s), then treated as gate failure with rollback; documented in the spec via the timeout scenario. Slow-starting legacy apps without healthchecks only pay the 2s grace.
- [Rollback start fails because the old image was pruned] → Old container still exists (stopped) under its backup name; error message points at manual recovery.
- [Static IP conflict during the overlap] → None: the old container is stopped before the new one is created, freeing its addresses.
- [2s grace is arbitrary] → It only needs to catch crash-loop-on-start images; deeper validation belongs to healthchecks, which users can add.
