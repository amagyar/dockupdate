## MODIFIED Requirements

### Requirement: Compose service restart

The system SHALL restart updated compose services by invoking the first available compose provider (`docker compose`, `podman compose`, `docker-compose`, `podman-compose`) with the project name, project directory, and config files taken from the container's compose labels, running `up -d --force-recreate <service>`. Within a single apply run, the system SHALL execute at most one recreate per compose project+service, regardless of how many running containers (replicas) that service has.

#### Scenario: Compose service restarted via podman compose

- **WHEN** an updated container belongs to compose project `webapp` service `web` on a Podman machine with `podman-compose` installed
- **THEN** the service is recreated through the detected provider using the labels' project, directory, and config files

#### Scenario: Scaled service recreated once

- **WHEN** service `web` of project `webapp` runs 3 replicas, all 3 rows are checked, and the user applies the selection
- **THEN** `up -d --force-recreate web` is invoked exactly once and all 3 replica rows track the shared task's pipeline state

#### Scenario: Partial selection still recreates the whole service once

- **WHEN** service `web` runs 3 replicas but only 1 replica row is checked and applied
- **THEN** the service is recreated once (compose recreates every replica) and only the checked row shows the pipeline state

#### Scenario: No compose provider available

- **WHEN** an updated compose-managed container exists but no compose provider binary is found
- **THEN** the item fails with an error stating that a compose provider is required, after the pull and verification have succeeded
