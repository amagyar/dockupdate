## ADDED Requirements

### Requirement: Network listing

The system SHALL list all networks via the Engine API showing name, driver, subnet (when available), and the count of connected containers.

#### Scenario: Networks listed

- **WHEN** the user opens the Networks tab
- **THEN** every network is shown with its name, driver, subnet, and connected-container count

### Requirement: Network drill-down

The system SHALL let the user open a selected network to view all containers connected to it, showing container name, IPv4 address, and compose project (or `standalone`).

#### Scenario: Open network shows connected containers

- **WHEN** the user selects the network `main` and presses `enter`
- **THEN** the view shows every container attached to `main` with its name, IP address, and compose project membership

#### Scenario: Empty network

- **WHEN** the user opens a network with no connected containers
- **THEN** the detail view shows an explicit "no containers connected" message instead of a blank area

### Requirement: Drill-down navigation

The system SHALL let the user return from a network detail view to the network list with `esc`, preserving the previous list selection.

#### Scenario: Back to list

- **WHEN** the user presses `esc` inside a network detail view
- **THEN** the network list is shown again with the previously selected network still highlighted
