## Why

Developers running containers locally (Docker or Podman) have no fast, visual way to answer three everyday questions: what is running and how it is grouped, what shares a network, and which running images have updates available in their registry. Existing tools are either CLI-only scripts (dockcheck), daemon-based auto-updaters (watchtower), or Docker-only GUIs. `dockupdate` fills this gap with a single-binary terminal UI that works against both Docker and Podman through the Docker-compatible Engine API.

## What Changes

- New Go CLI `dockupdate` built with the Bubble Tea TUI framework, shipped as a single static binary.
- Automatic detection of and connection to Docker or Podman engine sockets (no `docker`/`podman` CLI required for container operations; compose provider CLI used only for compose service recreation).
- Services view: containers grouped by compose project and service (via `com.docker.compose.*` labels), with standalone containers in their own group.
- Networks view: list of networks with drill-down into the containers attached to a selected network.
- Updates view: automatic background registry digest checks (local repo digest vs remote manifest digest) listing running containers with available updates.
- Interactive update execution: checkbox selection, concurrent per-item update pipeline (pull with live progress → verify digest → restart service → success/failure), each item progressing independently as a one-line status.
- Distribution plumbing: goreleaser producing GitHub Releases binaries, a Homebrew tap formula, and an npm wrapper package that downloads the correct binary at install time.
- Unit test suite covering parsing, digest comparison, progress aggregation, and the update state machine.

## Capabilities

### New Capabilities

- `engine-connectivity`: Detection of, connection to, and identification of Docker/Podman engine API sockets, including friendly handling of unreachable engines.
- `container-inventory`: Enumeration of containers and their grouping by compose project/service, plus standalone containers, with per-container metadata (state, image, update badge).
- `network-browser`: Listing of networks and drill-down into containers attached to a selected network.
- `update-checking`: Background and manual checking of available image updates via registry manifest digest comparison with Docker keychain authentication.
- `update-execution`: Interactive selection and concurrent execution of updates (pull with progress, digest verification, compose/standalone restart) with per-item one-line status states.
- `tui-layout`: The terminal UI layout, navigation model, keybindings, and visual states across the header, footer, and the Services/Networks/Updates tabs.
- `distribution`: Build and release of the single binary via goreleaser, GitHub Releases, Homebrew tap, and the npm wrapper package.

### Modified Capabilities

<!-- None - greenfield project, no existing specs. -->

## Impact

- **New codebase**: entire Go module (`cmd/dockupdate`, `internal/{engine,compose,registry,updater,tui}`), tests, and release automation (`.goreleaser.yml`, `.github/workflows/release.yml`, `npm/`).
- **External systems touched**: Docker/Podman Engine API sockets on the user's machine; OCI registries (Docker Hub, GHCR, ECR, private) read via `~/.docker/config.json` auth; compose provider CLIs (`docker compose`, `podman compose`, `docker-compose`, `podman-compose`) invoked for service recreation.
- **Key dependencies**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `docker/docker` client, `google/go-containerregistry`.
- **User prerequisites for release**: GitHub releases repo, a `homebrew-tap` repo, npm account/token (secrets only; all config is generated).
- No existing code is modified (greenfield repository).
