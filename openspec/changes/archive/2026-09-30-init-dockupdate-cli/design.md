## Context

`dockupdate` is a greenfield Go CLI. The development machine runs Podman 6.0.1 via a libkrun machine, whose Docker-compatible API socket is exposed at `$TMPDIR/podman/podman-machine-default-api.sock` (discoverable via `podman machine inspect`). No Docker daemon is present; `podman-compose` is the available compose provider. The tool must still work identically against Docker (Linux daemon, Docker Desktop) and rootless Podman on Linux.

Target environment constraints:

- Engine access must not depend on the `docker` or `podman` CLI for container operations — only the compose provider CLI is shelled out to.
- Update detection must not download image data (HEAD manifest requests only) and must authenticate to private registries using the user's existing `~/.docker/config.json`.
- The output artifact is a single static binary per platform, distributed via GitHub Releases, Homebrew tap, and npm.

## Goals / Non-Goals

**Goals:**

- One Go code path that works against any Docker-compatible Engine API socket (Docker, Docker Desktop, Podman rootful/rootless, Podman machine).
- Responsive TUI: engine listing, registry checks, and updates all run asynchronously; the UI never blocks on I/O.
- Per-item concurrent update pipeline with independent one-line status progression: `pulling (progress)` → `verifying checksum` → `restarting service` → `success/failure`.
- Safe standalone-container recreation preserving inspect config (env, ports, volumes, networks, restart policy).
- Testable core: all engine/registry/compose interactions behind Go interfaces; unit tests run without any daemon.

**Non-Goals:**

- Kubernetes, Docker Swarm, and Podman pod management (such containers are detected and excluded from updates).
- Building/pulling images from Dockerfiles, image pruning UI, log viewing, exec into containers.
- Homebrew-core submission (own tap only) and per-platform npm optional-dependency packages (single wrapper with postinstall).
- Windows-native engine support (Windows binaries are built, but engine connectivity targets unix sockets / `npipe` is out of scope for v1).

## Decisions

### D1: Engine access via Docker Engine API SDK, not CLIs

Use `github.com/docker/docker/client` over a unix socket for all container/network/image/pull operations. Podman ≥4 exposes a Docker-compatible API, so one code path serves both engines.

- **Alternatives considered:** Podman Go bindings (`containers/podman/v5/pkg/bindings`) — Podman-only, loses Docker users. Shelling out to `docker`/`podman` CLI — brittle parsing, requires CLI installed, slower. containerd API — too low-level, and bypasses the user's engine of choice.
- Engine identity is read from `GET /info` (`ServerVersion`, `Components[].Name` contains "Podman Engine") and shown in the header.

### D2: Socket auto-detection chain

Probe in order, first socket that answers `Ping` wins:

1. `--socket` flag, then `DOCKUPDATE_HOST`, then `DOCKER_HOST` env vars
2. `/var/run/docker.sock` (Linux Docker / rootful Podman)
3. `~/.docker/run/docker.sock` (Docker Desktop on macOS)
4. Podman machine socket: run `podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}'` when a `podman` binary exists; glob fallback `$TMPDIR/podman/*-api.sock`
5. Linux rootless Podman: `$XDG_RUNTIME_DIR/podman/podman.sock`, `/run/user/$UID/podman/podman.sock`

- **Alternative considered:** only honoring `DOCKER_HOST` — fails the default Podman-machine-on-macOS case (nothing sets it there).

### D3: Update detection via registry manifest digest comparison

For each running container's image reference, compare the local image's `RepoDigests` entry against the remote manifest digest (`remote.Head`) for that tag using `google/go-containerregistry` with the Docker keychain (reads `~/.docker/config.json`; covers Docker Hub, GHCR, ECR, GAR, private registries).

- **Alternatives considered:** `docker manifest inspect` CLI — requires experimental CLI, Docker-only. Always pulling to compare — heavy network cost; rejected.
- Checks run in a background worker at startup and on `r`, with per-image results streamed into the TUI as messages. Locally built images (no `RepoDigests`) are reported as not updatable.

### D4: Compose operations shell out to a detected provider; everything else uses the API

Compose service recreation requires compose-file semantics (dependency order, networks, env interpolation). Detect the first available provider: `docker compose` → `podman compose` → `docker-compose` → `podman-compose`. Recreate with `compose -p <project> [-f <config_files>] up -d --force-recreate <service>` run from the project directory (as the process working directory — the common denominator across providers; `--project-directory` is unsupported by podman-compose), using the `com.docker.compose.*` container labels with `io.podman.compose.*` fallbacks.

- Images are always pulled via the Engine API (D1) before `up -d` so progress is uniform; compose's default `pull_policy: missing` means `up -d` then only recreates.
- **Alternative considered:** reimplementing recreate with `compose-go` — large scope, subtle drift from real compose behavior; rejected for v1.
- Standalone containers are recreated purely through the API: inspect → pull → stop (10s timeout) → remove → create with the stored config (temporary name swap to free the name) → start → reconnect additional networks.

### D5: Per-item state machine with bounded-concurrency worker pool

Each selected update is a task executed by a worker goroutine: `Pulling(progress) → Verifying → Restarting → Done | Failed(err)`. Pull progress aggregates per-layer JSON events (`jsonmessage`) into a single percent/bytes value. Workers emit `tea.Msg` over a channel; the Bubble Tea model is the only state mutator. Default concurrency 3 (`--concurrency`).

- **Alternative considered:** unbounded goroutine per item — risks saturating the registry connection and engine on large selections.

### D6: Bubble Tea architecture

Single root `tea.Model` with a tab enum (Services / Networks / Updates) and per-tab sub-models. Long-running operations (engine queries, registry checks, updates) return `tea.Cmd`s that deliver result messages. Bubbles `progress` renders pull bars; Lipgloss provides the style system. Testable via `tea.NewProgram(..., tea.WithoutRenderer())` plus direct `Update(msg)` unit tests.

### D7: Distribution via goreleaser + npm postinstall wrapper

goreleaser (`CGO_ENABLED=0`, `-s -w` ldflags, version injected via `main.version`) builds linux/darwin/windows × amd64/arm64 archives + checksums, and pushes a Homebrew formula to a `homebrew-tap` repo. The npm package `dockupdate` ships only `package.json`, `bin/dockupdate` shim, and `install.js`, which detects platform/arch and downloads the matching archive from the GitHub release matching the package version. A GitHub Actions release workflow runs goreleaser and then `npm publish` on tag push.

- **Alternative considered:** per-platform npm optionalDependencies — 8+ packages per release; rejected in favor of one small package.

## Risks / Trade-offs

- [Podman compat API gaps vs Docker] → Integration smoke test against the local Podman machine; feature-detect at startup via `/info` version; degrade gracefully per missing endpoint.
- [Compose provider flags differ slightly across the 4 providers] → Restrict to the common flag subset (`-p`, `--project-directory`, `-f`, `up -d --force-recreate`); surface provider stderr verbatim in the failure state.
- [Standalone recreate can lose exotic runtime config (devices, caps, custom seccomp)] → Recreate from the full inspect `Config`/`HostConfig`; mark standalone updates with a caution indicator; on recreate failure, report the error and leave the old container stopped-but-present for manual recovery.
- [Registry rate limits (Docker Hub anonymous pulls/HEADs)] → Reuse keychain auth when present; cache check results for the session; manual refresh only re-checks stale entries.
- [macOS Podman socket lives under `$TMPDIR` (per-session)] → Never hardcode; always resolve via `podman machine inspect` or glob at startup.
- [First release needs user-created repos/secrets (GitHub, homebrew-tap, npm token)] → Document exact setup steps in README; workflow fails fast with a clear message if secrets are missing.

## Open Questions

- Default behavior for old images after a successful update: keep (current plan; `--prune` opt-in) vs prompt per update. Revisit after first dogfood.
- Whether update checks should re-run on a timer (current: startup + manual `r` only).
