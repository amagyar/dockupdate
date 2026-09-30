## ADDED Requirements

### Requirement: Non-interactive check mode

When started with `--check`, the system SHALL connect to the engine, run registry digest checks for every updatable running container (same eligibility rules as the TUI: excluding managed workloads and non-running containers), print one line per container with its name, image reference, and check outcome, and exit without starting the TUI.

#### Scenario: Mixed outcomes reported

- **WHEN** the engine has containers with updates available, up-to-date containers, a local build, and a digest-pinned image
- **THEN** stdout lists each container with its respective outcome, e.g. `web  nginx:1.25  update available`

#### Scenario: No containers

- **WHEN** the engine has no updatable running containers
- **THEN** the report states that no updatable containers were found and the exit code reflects "no updates"

### Requirement: Scriptable exit codes

Check mode SHALL exit with code `0` when no updates are available, `1` when at least one update is available, and `2` on operational errors (engine unreachable, or at least one check failed). Output intended for humans SHALL go to stdout; error diagnostics SHALL go to stderr.

#### Scenario: Updates pending in cron

- **WHEN** `dockupdate --check` runs and two containers have updates available
- **THEN** it prints the report and exits with code 1, so a wrapper script can notify

#### Scenario: Engine unreachable

- **WHEN** no engine socket answers
- **THEN** a diagnostic naming the probed locations goes to stderr and the exit code is 2

#### Scenario: One registry failing

- **WHEN** one container's registry check fails while others succeed
- **THEN** the failed container is listed with its error, the report still completes, and the exit code is 2

### Requirement: Check mode honors connection flags

Check mode SHALL honor `--socket`, `DOCKUPDATE_HOST`, and `DOCKER_HOST` exactly like the TUI mode. Flags that only apply to interactive updates (`--concurrency`, `--prune`) SHALL have no effect in check mode.

#### Scenario: Explicit socket

- **WHEN** `dockupdate --check --socket unix:///custom/docker.sock` runs
- **THEN** only that socket is probed, as in TUI mode
