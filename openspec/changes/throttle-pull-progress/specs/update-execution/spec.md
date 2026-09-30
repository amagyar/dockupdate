## MODIFIED Requirements

### Requirement: Pull progress reporting

The system SHALL render the `pulling` state as a one-line progress bar with percentage and bytes, aggregating per-layer download progress reported by the Engine API into a single overall value. When the engine reports no byte-level progress (e.g. Podman), the row SHALL fall back to completed-layer counts (`N/M layers`).

Progress samples SHALL be coalesced per task with latest-wins semantics and emitted at a bounded rate (at most one sample per task per 100ms), so UI render cost stays independent of engine event frequency. Phase changes, completion, and failure events SHALL NOT be throttled, and the final progress sample of a pull SHALL be emitted before the phase advances out of `pulling`.

#### Scenario: Progress bar updates during pull

- **WHEN** an item is in the `pulling` state and layers are downloading
- **THEN** its row shows a progress bar, overall percentage, and downloaded/total bytes, updating live

#### Scenario: Bursty engine events are coalesced

- **WHEN** the engine emits 500 progress events within one second for one task
- **THEN** at most ~10 progress samples reach the event channel for that task, the latest sample always reflecting the newest byte counts

#### Scenario: Final sample precedes phase change

- **WHEN** a pull completes
- **THEN** the last progress sample (100% or final layer count) is emitted before the `verifying checksum` phase change

#### Scenario: Engine without byte progress

- **WHEN** an item pulls on an engine that reports only per-layer status transitions (Podman)
- **THEN** its row shows a progress bar driven by completed layers with `N/M layers` text instead of bytes
