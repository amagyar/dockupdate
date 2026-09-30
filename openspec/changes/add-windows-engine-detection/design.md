## Context

`Candidates(Environment)` (`internal/engine/detect.go`) builds one unix-centric probe list. Windows support exists only incidentally: `normalize` passes `npipe://` through, and the moby client can dial npipe — but no candidate ever produces an npipe host, so a stock Windows run finds nothing.

## Goals / Non-Goals

**Goals:**
- Stock Docker Desktop and Podman machine setups on Windows connect with zero configuration.
- Unix ordering byte-for-byte unchanged.
- All logic stays unit-testable on any OS (no Windows runner required).

**Non-Goals:**
- TCP detection (Docker Desktop's `localhost:2375` is opt-in and insecure; users can set `DOCKER_HOST`).
- Windows *service* support or named-pipe ACL diagnostics.
- Changing the ping/connect machinery — the client already speaks npipe.

## Decisions

**Runtime GOOS switch, not build tags.** `Candidates` takes the platform from a new `Environment.GOOS` field (defaulted to `runtime.GOOS` in `DefaultEnvironment`). A test sets `GOOS: "windows"` and asserts the candidate list — no `//go:build windows` files, no dual test binaries. Alternative considered: build-tagged `candidates_windows.go`/`candidates_unix.go` — rejected; it splits the ordering across files and makes the Windows path untestable from the macOS/Linux dev machines this project is developed on.

**Windows list: npipe first, then Podman machine.** Docker Desktop is the dominant Windows engine; the npipe name is a fixed well-known path. `podman machine inspect` works on Windows (it shells out fine), so the existing `PodmanSocket` hook is reused as-is. Unix-only entries (`/var/run/...`, XDG, `$TMPDIR` glob, `/run/user/<uid>`) are skipped on Windows — they'd never exist and only add 5s ping timeouts.

**`os.Getuid` needs no guard:** it returns -1 on Windows, and the `UID >= 0` check already skips the rootless path. Verified against the Go standard library docs.

## Risks / Trade-offs

- [Podman machine inspect slow/failing on Windows adds startup latency] → Same behavior as unix today (10s probe budget overall, 5s per ping); acceptable.
- [Docker Desktop's npipe exists but the engine is still starting] → Ping fails, falls through to the next candidate, and the unreachable-engine view already lists what was probed with an `r` retry.
- [Future platforms (freebsd etc.)] → The switch defaults to the unix list for any non-windows GOOS, preserving current behavior.
