## Why

Release binaries are built for windows/amd64 and windows/arm64, but socket auto-detection only probes unix paths (`/var/run/docker.sock`, XDG/rootless Podman paths, `$TMPDIR` globs). On Windows — where Docker Desktop and Podman machine are the common setups — the tool fails out of the box unless the user manually sets `DOCKER_HOST` or passes `--socket npipe:////./pipe/docker_engine`. The most common Windows path should just work.

## What Changes

- On Windows, the probe list includes the Docker Desktop named pipe `npipe:////./pipe/docker_engine` and the Podman machine socket (resolved via `podman machine inspect`, as on unix) instead of the unix-only filesystem paths.
- Explicit configuration (`--socket`, `DOCKUPDATE_HOST`, `DOCKER_HOST`) keeps working unchanged and still wins over auto-detection; `npipe://` values are already passed through by `normalize`.
- Unix candidate ordering is untouched.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `engine-connectivity`: the "Automatic engine socket detection" requirement changes — the candidate list becomes platform-aware, adding the Windows named pipe and omitting unix-only paths on Windows.

## Impact

- **Code**: `internal/engine/detect.go` (platform-conditional candidate list; `os.Getuid` already returns -1 on Windows so the rootless path self-disables).
- **Distribution**: the existing windows release artifacts become usable out of the box; README documents the Windows default.
- **Tests**: candidate-resolution unit tests gain a GOOS-conditional case (the pure `Candidates(Environment)` function makes this testable without a Windows runner).
