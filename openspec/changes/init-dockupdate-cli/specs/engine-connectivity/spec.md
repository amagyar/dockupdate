## ADDED Requirements

### Requirement: Automatic engine socket detection

The system SHALL detect a reachable Docker-compatible Engine API socket by probing candidates in this order: `--socket` flag, `DOCKUPDATE_HOST` env var, `DOCKER_HOST` env var, `/var/run/docker.sock`, `~/.docker/run/docker.sock`, the Podman machine socket resolved via `podman machine inspect`, glob fallback `$TMPDIR/podman/*-api.sock`, `$XDG_RUNTIME_DIR/podman/podman.sock`, `/run/user/<uid>/podman/podman.sock`. The first candidate that answers a ping SHALL be used.

#### Scenario: Podman machine detected on macOS without DOCKER_HOST

- **WHEN** the tool starts on macOS with no `DOCKER_HOST` set, no `/var/run/docker.sock`, and a running Podman machine
- **THEN** it connects to the socket reported by `podman machine inspect` and becomes operational

#### Scenario: Docker daemon detected on Linux

- **WHEN** the tool starts on Linux with a Docker daemon listening on `/var/run/docker.sock`
- **THEN** it connects to `/var/run/docker.sock` without requiring any flags or environment variables

#### Scenario: Explicit override wins

- **WHEN** the tool is started with `--socket unix:///custom/docker.sock` while `DOCKER_HOST` is also set
- **THEN** it connects to `unix:///custom/docker.sock` and ignores all other candidates

### Requirement: Engine identification

The system SHALL identify the connected engine by querying the Engine API info endpoint and SHALL display the engine kind (Docker or Podman) and server version in the TUI header.

#### Scenario: Podman identified

- **WHEN** the connected socket is served by Podman
- **THEN** the header shows `Podman` and its server version (e.g. `Podman 6.0.1`)

#### Scenario: Docker identified

- **WHEN** the connected socket is served by the Docker Engine
- **THEN** the header shows `Docker` and its server version

### Requirement: Unreachable engine handling

The system SHALL present a friendly error state when no candidate socket is reachable, listing what was probed, and SHALL offer a retry action without exiting the TUI.

#### Scenario: No engine running

- **WHEN** no candidate socket answers a ping at startup
- **THEN** the TUI shows an error view naming the probed locations and a `r` keybinding to retry detection

#### Scenario: Engine disappears mid-session

- **WHEN** the engine connection fails while the TUI is running
- **THEN** the TUI shows the error state and recovers to normal operation when a retry succeeds
