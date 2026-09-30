## ADDED Requirements

### Requirement: Cancellable update run

Each apply run SHALL execute on a cancellable context owned by the caller. Cancelling the context SHALL stop in-flight work (image pulls, digest verification, compose provider invocations, standalone recreation) and SHALL move every not-yet-terminal task to the failed state with a cancellation reason instead of leaving it hanging.

#### Scenario: Cancel during pull

- **WHEN** the user confirms quit while a task is in the `pulling` phase
- **THEN** the pull request to the engine is cancelled, the task ends in `failed` with a cancellation reason, and the worker exits without starting further queued tasks

#### Scenario: Queued tasks never start after cancel

- **WHEN** cancellation happens while tasks are still queued behind the worker pool
- **THEN** queued tasks terminate as cancelled without pulling or restarting anything

#### Scenario: Cancellation is reported, not mistaken for a registry/engine failure

- **WHEN** a task is aborted by cancellation
- **THEN** its error clearly indicates the update was cancelled (e.g. `update cancelled`), not a pull/verify/restart failure

## MODIFIED Requirements

### Requirement: Failure isolation between items

The system SHALL confine an item's failure to that item; other in-flight updates SHALL continue unaffected. Run-wide cancellation is the explicit exception: it terminates all remaining tasks of that run by design.

#### Scenario: One failure among three

- **WHEN** three updates run concurrently and one fails during restart
- **THEN** the failing row shows `failed` with the error reason while the other two proceed to completion

#### Scenario: Cancellation affects the whole run only

- **WHEN** a run is cancelled while two of three tasks are in flight
- **THEN** both in-flight tasks end as cancelled and no further restarts happen for that run, while already-completed tasks keep their `success` state
