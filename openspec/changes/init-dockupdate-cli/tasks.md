## 1. Project Scaffold

- [x] 1.1 Initialize Go module `github.com/<owner>/dockupdate` with `cmd/dockupdate/main.go` and `internal/` layout; add dependencies: bubbletea, bubbles, lipgloss, docker/docker client, google/go-containerregistry
- [x] 1.2 Add version variable injected via ldflags and `--version` flag; add Cobra-free flag parsing (`--socket`, `--concurrency`, `--prune`, `--version`)
- [x] 1.3 Verify scaffold builds a single static binary with `CGO_ENABLED=0 go build ./cmd/dockupdate`

## 2. Engine Connectivity (spec: engine-connectivity)

- [x] 2.1 Implement `internal/engine` socket candidate resolution in the spec'd order (flag → DOCKUPDATE_HOST → DOCKER_HOST → /var/run/docker.sock → ~/.docker/run/docker.sock → podman machine inspect → $TMPDIR glob → rootless paths)
- [x] 2.2 Implement ping-based probing and engine identification (Docker vs Podman + server version) from the info endpoint
- [x] 2.3 Verify live connection to the local Podman machine socket (`podman machine inspect` path) with a smoke command

## 3. Container Inventory (spec: container-inventory)

- [x] 3.1 Implement container listing via Engine API with name, ID, image ref, state, and compose labels
- [x] 3.2 Implement grouping into project → service → containers tree plus the `Standalone` group
- [x] 3.3 Implement managed-workload exclusion (Kubernetes/Swarm/Podman-pod labels) marking containers not updatable

## 4. Update Checking (spec: update-checking)

- [x] 4.1 Implement `internal/registry` digest checker: parse image refs, read local repo digests, HEAD remote manifest via go-containerregistry with Docker keychain
- [x] 4.2 Implement classification: update available / up to date / local build / pinned by digest / check failed (with error)
- [x] 4.3 Implement background check worker emitting per-image results as tea messages, started automatically at launch and on `r`

## 5. Compose Provider (spec: update-execution)

- [x] 5.1 Implement `internal/compose` provider detection chain (`docker compose` → `podman compose` → `docker-compose` → `podman-compose`)
- [x] 5.2 Implement `up -d --force-recreate <service>` invocation built from compose labels (project, working dir, config files), capturing stderr for failure states

## 6. Update Execution (spec: update-execution)

- [x] 6.1 Implement `internal/updater` task state machine: pulling → verifying checksum → restarting service → success/failure, emitting progress events
- [x] 6.2 Implement bounded worker pool (default 3, `--concurrency`) executing tasks concurrently and independently
- [x] 6.3 Implement Engine API image pull with per-layer jsonmessage aggregation into overall percent/bytes
- [x] 6.4 Implement post-pull digest verification against the check-phase remote digest (fail on mismatch)
- [x] 6.5 Implement standalone recreation: inspect → stop (10s) → remove → create from stored config → start → reconnect networks → preserve name
- [x] 6.6 Implement old-image retention by default and removal after success behind `--prune`

## 7. TUI (spec: tui-layout, container-inventory, network-browser, update-execution)

- [x] 7.1 Implement root Bubble Tea model with header/tab bar/content/footer regions and tab switching (`tab`, `shift+tab`, `1`-`3`)
- [x] 7.2 Implement header (app name, engine kind/version, shortened socket, updates badge) and context-sensitive footer
- [x] 7.3 Implement Services tab: collapsible project → service → container tree with state icons and `⬆` update badges
- [x] 7.4 Implement Networks tab: network table and drill-down detail view with container name/IP/project, `esc` back preserving selection
- [x] 7.5 Implement Updates tab: checkbox list (`space`, `a`, `enter`), one-line status states per spec (checking…, pulling bar, verifying checksum, restarting service, ✔/✖), updatable-first section split
- [x] 7.6 Implement empty states, engine-unreachable error state with retry, terminal resize handling (min 80×24), and quit confirmation while updates are in flight

## 8. Tests

- [x] 8.1 Unit tests for socket candidate ordering and override precedence (engine)
- [x] 8.2 Unit tests for image ref parsing and digest comparison outcomes (registry), using httptest registry fixtures
- [x] 8.3 Unit tests for pull progress aggregation across layers (updater)
- [x] 8.4 Unit tests for the update state machine transitions and failure isolation, against a fake engine/compose provider
- [x] 8.5 Unit tests for TUI model: tab switching, checkbox toggling, one-liner state rendering, empty states (via `Update(msg)` assertions with `tea.WithoutRenderer`)
- [x] 8.6 `go vet` + `go test ./...` green in CI on linux and darwin

## 9. Distribution (spec: distribution)

- [x] 9.1 Add `.goreleaser.yml`: static builds for linux/darwin/windows × amd64/arm64, archives + checksums, Homebrew tap config
- [x] 9.2 Add `npm/package.json`, `npm/bin/dockupdate` shim, and `npm/install.js` (platform/arch detection, GitHub release download, checksum verification, unsupported-platform error)
- [x] 9.3 Add `.github/workflows/release.yml`: tag-triggered goreleaser + npm publish with fail-fast secret checks
- [x] 9.4 Validate with `goreleaser release --snapshot --clean` and `node npm/install.js` against a snapshot release layout

## 10. Documentation & Dogfood

- [x] 10.1 Write README: features, install (brew/npm/binary), usage, keybindings, flags, repo/secret setup for releasing
- [x] 10.2 Dogfood against the local Podman machine: create a throwaway compose stack with older-pinned tags (e.g. nginx, redis), verify grouping, network drill-down, update detection, and the concurrent one-liner update flow
