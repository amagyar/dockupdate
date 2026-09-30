## MODIFIED Requirements

### Requirement: Standalone container recreation

The system SHALL update standalone containers by pulling the new image and recreating the container via the Engine API from its stored inspect configuration (environment, ports, volumes, command, restart policy), reconnecting it to all previously attached networks while preserving user-configured endpoint settings (static IP/IPAM config, network aliases, links), and preserving its name. Runtime-assigned endpoint fields (endpoint IDs, runtime addresses) SHALL NOT be carried over.

After the replacement container starts, the system SHALL gate removal of the old container on the replacement proving viable: when the container defines a healthcheck, wait until it reports `healthy` (bounded by a timeout); otherwise verify it is still running after a short grace period. When the gate fails, the update SHALL fail and the system SHALL attempt an automatic rollback: remove the failed replacement, restore the old container's original name, and start it again.

#### Scenario: Standalone container recreated with same config

- **WHEN** a standalone container with custom env, published ports, volumes, and membership in two networks is updated
- **THEN** the replacement container runs the new image with the same env, ports, volumes, name, and both network attachments

#### Scenario: Static IP and alias preserved

- **WHEN** a standalone container has a static IPv4 and an alias on a user-defined network and is updated
- **THEN** the replacement container attaches to that network with the same static IPv4 and alias

#### Scenario: Healthy replacement retires the old container

- **WHEN** the replacement container defines a healthcheck and reports `healthy` within the timeout
- **THEN** the old container is removed and the update succeeds

#### Scenario: Unhealthy replacement rolls back

- **WHEN** the replacement container reports `unhealthy` (or exits during the grace period)
- **THEN** the update fails, the replacement is removed, and the old container is restarted under its original name

#### Scenario: Recreate failure preserves diagnosis path

- **WHEN** container creation with the new image fails after the old container was stopped
- **THEN** the item fails with the engine error shown and the old container remains present (stopped) for manual recovery
