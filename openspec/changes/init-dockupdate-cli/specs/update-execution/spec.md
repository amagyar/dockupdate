## ADDED Requirements

### Requirement: Interactive update selection

The system SHALL let the user select which containers with available updates to update, via checkboxes: `space` toggles the focused row, `a` toggles all rows, and `enter` applies the current selection.

#### Scenario: Select and apply two of three

- **WHEN** three updates are listed, the user checks two rows with `space` and presses `enter`
- **THEN** exactly the two checked updates start and the third row remains untouched

#### Scenario: Apply with empty selection is a no-op

- **WHEN** the user presses `enter` with no rows checked
- **THEN** no update starts and the view remains unchanged

### Requirement: Concurrent independent update pipeline

The system SHALL execute selected updates concurrently with a bounded worker pool (default 3 workers, configurable via `--concurrency`), where each item proceeds through `pulling` → `verifying checksum` → `restarting service` → `success`/`failure` independently of the others.

#### Scenario: Fast item finishes while slow item still downloads

- **WHEN** updates for a 100MB image and a 3GB image are applied together
- **THEN** the 100MB item advances through verify and restart to `success` while the 3GB item is still in the `pulling` state

#### Scenario: Concurrency bound respected

- **WHEN** five updates are applied with concurrency 3
- **THEN** at most three pulls are in flight at any moment and the remaining items start as workers free up

### Requirement: Pull progress reporting

The system SHALL render the `pulling` state as a one-line progress bar with percentage and bytes, aggregating per-layer download progress reported by the Engine API into a single overall value. When the engine reports no byte-level progress (e.g. Podman), the row SHALL fall back to completed-layer counts (`N/M layers`).

#### Scenario: Progress bar updates during pull

- **WHEN** an item is in the `pulling` state and layers are downloading
- **THEN** its row shows a progress bar, overall percentage, and downloaded/total bytes, updating live

#### Scenario: Engine without byte progress

- **WHEN** an item pulls on an engine that reports only per-layer status transitions (Podman)
- **THEN** its row shows a progress bar driven by completed layers with `N/M layers` text instead of bytes

### Requirement: Digest verification

The system SHALL verify, after pulling, that the pulled image's repo digest matches the remote digest observed during the update check, showing `verifying checksum` during this step and failing the item on mismatch.

#### Scenario: Checksum matches

- **WHEN** the pulled image's repo digest equals the previously observed remote digest
- **THEN** the item advances to `restarting service`

#### Scenario: Checksum mismatch

- **WHEN** the pulled image's repo digest differs from the previously observed remote digest
- **THEN** the item fails with a `checksum mismatch` error and the container is not restarted

### Requirement: Compose service restart

The system SHALL restart updated compose services by invoking the first available compose provider (`docker compose`, `podman compose`, `docker-compose`, `podman-compose`) with the project name, project directory, and config files taken from the container's compose labels, running `up -d --force-recreate <service>`.

#### Scenario: Compose service restarted via podman compose

- **WHEN** an updated container belongs to compose project `webapp` service `web` on a Podman machine with `podman-compose` installed
- **THEN** the service is recreated through the detected provider using the labels' project, directory, and config files

#### Scenario: No compose provider available

- **WHEN** an updated compose-managed container exists but no compose provider binary is found
- **THEN** the item fails with an error stating that a compose provider is required, after the pull and verification have succeeded

### Requirement: Standalone container recreation

The system SHALL update standalone containers by pulling the new image and recreating the container via the Engine API from its stored inspect configuration (environment, ports, volumes, command, restart policy), reconnecting it to all previously attached networks, and preserving its name.

#### Scenario: Standalone container recreated with same config

- **WHEN** a standalone container with custom env, published ports, volumes, and membership in two networks is updated
- **THEN** the replacement container runs the new image with the same env, ports, volumes, name, and both network attachments

#### Scenario: Recreate failure preserves diagnosis path

- **WHEN** container creation with the new image fails after the old container was stopped
- **THEN** the item fails with the engine error shown and the old container remains present (stopped) for manual recovery

### Requirement: Failure isolation between items

The system SHALL confine an item's failure to that item; other in-flight updates SHALL continue unaffected.

#### Scenario: One failure among three

- **WHEN** three updates run concurrently and one fails during restart
- **THEN** the failing row shows `failed` with the error reason while the other two proceed to completion

### Requirement: Old image retention

The system SHALL keep the previous image after a successful update by default, and SHALL remove it after success only when started with `--prune`.

#### Scenario: Default keeps old image

- **WHEN** an update succeeds without `--prune`
- **THEN** the previously used image remains in the engine's image store

#### Scenario: Prune removes old image

- **WHEN** an update succeeds with `--prune` enabled
- **THEN** the previously used image is removed after the new container is running
