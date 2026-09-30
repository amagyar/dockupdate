## ADDED Requirements

### Requirement: Container enumeration

The system SHALL list all containers (running and stopped) via the Engine API with, at minimum: name, ID, image reference, state, and compose labels.

#### Scenario: Containers listed on startup

- **WHEN** the tool connects to an engine with containers present
- **THEN** the Services tab shows every container with its name, image:tag, and state

### Requirement: Compose project grouping

The system SHALL group containers carrying the `com.docker.compose.project` label by project name, and within a project by the `com.docker.compose.service` label, forming a project → service → containers hierarchy.

#### Scenario: Two services in one project

- **WHEN** containers exist with labels project=`webapp`, services=`web` and `db`
- **THEN** the Services tab shows a `webapp` group containing a `web` group and a `db` group, each holding their containers

#### Scenario: Standalone containers grouped separately

- **WHEN** containers exist without the `com.docker.compose.project` label
- **THEN** they appear under a `Standalone` group, not under any compose project

### Requirement: Managed-workload exclusion

The system SHALL exclude containers identified as managed by Kubernetes, Docker Swarm, or Podman pods (via their identifying labels) from update eligibility, while still displaying them.

#### Scenario: Kubernetes-managed container shown but not updatable

- **WHEN** a container carries `io.kubernetes.container.name`
- **THEN** it is listed in the Services view but marked as not updatable and excluded from update checks

### Requirement: Inventory refresh

The system SHALL refresh the container inventory when the user requests a refresh and after any update completes.

#### Scenario: Refresh after update

- **WHEN** an update finishes (success or failure)
- **THEN** the container inventory is re-queried so states and image references reflect reality
