# dockupdate

[![CI](https://github.com/adev/dockupdate/actions/workflows/ci.yml/badge.svg)](https://github.com/adev/dockupdate/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/adev/dockupdate)](https://github.com/adev/dockupdate)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9999/badge)](https://www.bestpractices.dev/projects/9999)

A terminal UI for managing containers across **Docker and Podman**: compose-aware inventory, network browsing, and interactive updates with live progress — shipped as a single binary.

```
┌ dockupdate ─ Podman 6.0.1 · unix://…/podman-machine-default-api.sock ── 2 updates ┐
│ [1 Services]  [2 Networks]  [3 Updates]                                           │
│                                                                                    │
│ [x] web    nginx:1.25     ⠋ pulling  ████████░░░░░░ 45%  45MB/100MB               │
│ [x] cache  redis:7.2      ⠋ verifying checksum…                                   │
│ [ ] db     postgres:16    update available                                        │
│                                                                                    │
│ tab/1-3 switch · r refresh · space select · enter apply · q quit                   │
└────────────────────────────────────────────────────────────────────────────────────┘
```

## Features

- **Works with Docker and Podman** through the Docker-compatible Engine API — auto-detects the socket (`/var/run/docker.sock`, Docker Desktop, Podman machine, rootless Podman), no `docker`/`podman` CLI needed for container operations.
- **Services tab** — containers grouped by compose project and service, standalone containers in their own group.
- **Networks tab** — list networks, drill into one to see every connected container.
- **Updates tab** — background registry digest checks (no downloads) against Docker Hub, GHCR, ECR and private registries (uses your existing `~/.docker/config.json` and Podman `auth.json`).
- **Interactive updates** — check the containers to update; each one pulls with a live progress bar and independently advances `pulling → verifying checksum → restarting service → ✔/✖`. A 100MB update finishes while a 3GB one is still downloading.
- **Compose-aware restarts** via your installed provider (`docker compose`, `podman compose`, `docker-compose`, or `podman-compose`); standalone containers are recreated in place with their full config preserved.

## Install

### Homebrew

```sh
brew install adev/tap/dockupdate
```

### npm

```sh
npm install -g dockupdate
```

The npm package downloads the matching binary from GitHub Releases at install time (checksum-verified).

### Binary

Download an archive for your platform from [GitHub Releases](https://github.com/adev/dockupdate/releases), verify it against `checksums.txt`, and place `dockupdate` on your `PATH`.

## Usage

```sh
dockupdate                  # start the TUI
dockupdate --socket unix:///run/user/501/podman/podman.sock
dockupdate --concurrency 5  # up to 5 updates at once (default 3)
dockupdate --prune          # remove old images after successful updates
dockupdate --version
```

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--socket` | auto-detect | Engine socket; overrides `DOCKUPDATE_HOST`/`DOCKER_HOST` |
| `--concurrency` | `3` | Max concurrent updates |
| `--prune` | `false` | Remove old images after a successful update |
| `--version` | | Print version and exit |

### Keybindings

| Key | Action |
| --- | --- |
| `tab` / `shift+tab`, `1` `2` `3` | Switch tabs |
| `↑`/`↓` or `j`/`k` | Move |
| `enter` | Collapse/expand group · open network · apply selected updates |
| `space` | Toggle update checkbox |
| `a` | Select/deselect all updates |
| `esc` | Back (network detail) |
| `r` | Refresh inventory + re-check updates |
| `q`, `ctrl+c` | Quit (asks for confirmation while updates run) |

## How updates work

1. **Check** — the local image's repo digest is compared to the registry's manifest digest for the tag (HEAD request only). Multi-arch tags accept either the index or the platform manifest digest.
2. **Pull** — the new image is pulled via the Engine API with per-layer progress aggregated into one bar.
3. **Verify** — the pulled image's digest must match the digest seen during the check, otherwise the update fails before touching the container.
4. **Restart** — compose services are recreated with `up -d --force-recreate <service>`; standalone containers are stopped, recreated from their stored inspect config (env, ports, volumes, networks, restart policy) and started under the same name. On failure the old container is left stopped for manual recovery.

Images built locally (no repo digest), digest-pinned images, and containers managed by Kubernetes/Swarm/Podman pods are shown but never updated.

## Releasing (maintainers)

Releases are driven by git tags via [goreleaser](.goreleaser.yml) and [.github/workflows/release.yml](.github/workflows/release.yml):

1. Create the repos: `github.com/adev/dockupdate` (code + releases) and `github.com/adev/homebrew-tap` (formula).
2. Add repo secrets: `HOMEBREW_TAP_GITHUB_TOKEN` (PAT with write access to homebrew-tap) and `NPM_TOKEN` (npm granular access token).
3. Tag and push:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The workflow fails fast with a clear message if a secret is missing, then publishes GitHub Releases binaries, the Homebrew formula, and the npm package.

## Development

```sh
go build ./...                 # build
go vet ./... && go test ./...  # checks
go test -tags live ./internal/engine/   # smoke test against the real local engine
CGO_ENABLED=0 go build -o dockupdate ./cmd/dockupdate
```

Layout: `cmd/dockupdate` (entrypoint), `internal/engine` (socket detection + Engine API wrapper), `internal/registry` (digest checks + keychains), `internal/compose` (provider detection), `internal/updater` (update pipeline), `internal/tui` (Bubble Tea UI).

## License

MIT
