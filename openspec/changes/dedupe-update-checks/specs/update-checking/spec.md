## ADDED Requirements

### Requirement: Per-pass remote lookup deduplication

Within a single check pass, the system SHALL perform the remote registry lookup (HEAD plus optional manifest fetch) at most once per image reference. Concurrent checks for the same reference SHALL share one in-flight lookup, and later checks in the same pass SHALL reuse its result. The cache SHALL be discarded when the pass ends, so a manual refresh always queries the registry anew.

#### Scenario: Ten containers on one image

- **WHEN** ten running containers use `nginx:1.25` and a check pass starts
- **THEN** exactly one remote lookup for `nginx:1.25` is issued and all ten rows resolve from it

#### Scenario: Same ref, different local images

- **WHEN** two containers use `nginx:1.25` but run different local image IDs (one current, one older)
- **THEN** the shared remote digest is compared against each container's own local repo digests, flagging only the outdated one as update-available

#### Scenario: Refresh re-queries

- **WHEN** a check pass completed and the user presses `r`
- **THEN** the registry is queried again for each referenced image (no stale reuse across passes)

#### Scenario: Shared lookup failure is classified per container

- **WHEN** the single remote lookup for a shared reference fails
- **THEN** every container using that reference shows `check failed` with the same cause, and references with successful lookups are unaffected
