## MODIFIED Requirements

### Requirement: Quit confirmation during updates

The system SHALL ask for confirmation before quitting while any update is in flight, for both `q` and `ctrl+c`; quitting only proceeds after confirmation. Confirming the quit SHALL cancel the in-flight update run before the program exits.

#### Scenario: Quit with updates running

- **WHEN** the user presses `q` while an update is in flight
- **THEN** a confirmation prompt appears and the program exits only after the user confirms

#### Scenario: Ctrl+C with updates running

- **WHEN** the user presses `ctrl+c` while an update is in flight
- **THEN** the same confirmation prompt appears instead of an immediate quit

#### Scenario: Ctrl+C with no updates running

- **WHEN** the user presses `ctrl+c` while no update is in flight
- **THEN** the program quits immediately, as today

#### Scenario: Confirmed quit cancels in-flight work

- **WHEN** the user confirms the quit prompt while updates are in flight
- **THEN** the update run's context is cancelled (pulls and provider invocations stop) before the program exits
