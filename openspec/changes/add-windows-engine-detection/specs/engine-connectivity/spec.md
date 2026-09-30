## MODIFIED Requirements

### Requirement: Automatic engine socket detection

The system SHALL detect a reachable Docker-compatible Engine API socket by probing a platform-aware candidate list, with explicit configuration always taking precedence: `--socket` flag, then `DOCKUPDATE_HOST`, then `DOCKER_HOST` are the only candidates when set.

On Linux and macOS the remaining candidates, in order, are: `/var/run/docker.sock`, `~/.docker/run/docker.sock`, the Podman machine socket resolved via `podman machine inspect`, glob fallback `$TMPDIR/podman/*-api.sock`, `$XDG_RUNTIME_DIR/podman/podman.sock`, `/run/user/<uid>/podman/podman.sock`.

On Windows the remaining candidates, in order, are: the Docker Desktop named pipe `npipe:////./pipe/docker_engine`, then the Podman machine socket resolved via `podman machine inspect`. Unix filesystem paths SHALL NOT be probed on Windows.

The first candidate that answers a ping SHALL be used.

#### Scenario: Docker Desktop detected on Windows

- **WHEN** the tool starts on Windows with no `DOCKER_HOST` set and Docker Desktop running
- **THEN** it connects to `npipe:////./pipe/docker_engine` without requiring any flags or environment variables

#### Scenario: Podman machine detected on Windows

- **WHEN** the tool starts on Windows with Docker Desktop absent and a running Podman machine
- **THEN** it connects to the socket reported by `podman machine inspect`

#### Scenario: Podman machine detected on macOS without DOCKER_HOST

- **WHEN** the tool starts on macOS with no `DOCKER_HOST` set, no `/var/run/docker.sock`, and a running Podman machine
- **THEN** it connects to the socket reported by `podman machine inspect` and becomes operational

#### Scenario: Docker daemon detected on Linux

- **WHEN** the tool starts on Linux with a Docker daemon listening on `/var/run/docker.sock`
- **THEN** it connects to `/var/run/docker.sock` without requiring any flags or environment variables

#### Scenario: Explicit override wins

- **WHEN** the tool is started with `--socket unix:///custom/docker.sock` while `DOCKER_HOST` is also set
- **THEN** it connects to `unix:///custom/docker.sock` and ignores all other candidates

#### Scenario: Explicit npipe override on Windows

- **WHEN** the tool is started on Windows with `DOCKER_HOST=npipe:////./pipe/docker_engine`
- **THEN** it connects to that named pipe (scheme passthrough, no `unix://` rewriting)
