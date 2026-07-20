## ADDED Requirements

### Requirement: Digest-based update detection

For each updatable running container, the system SHALL compare the local image's repo digest against the remote registry manifest digest for the same repository and tag, using a HEAD request (no image download). An update is available when the remote digest differs from every local repo digest for that image.

#### Scenario: Update available

- **WHEN** container runs `nginx:1.25` whose local repo digest differs from the registry's current manifest digest for `nginx:1.25`
- **THEN** the container is flagged as having an update available

#### Scenario: Image up to date

- **WHEN** the local repo digest matches the remote manifest digest
- **THEN** the container is flagged as up to date

### Requirement: Registry authentication

The system SHALL authenticate registry requests using the Docker keychain (`~/.docker/config.json` and platform credential helpers), covering Docker Hub and private registries.

#### Scenario: Private registry check

- **WHEN** an image reference points to a private registry the user has credentials for
- **THEN** the digest check authenticates with those credentials and returns a result instead of an auth error

### Requirement: Automatic background check

The system SHALL start update checks automatically in a background worker at startup without blocking the TUI, streaming per-image results into the Updates view as they complete.

#### Scenario: Non-blocking startup check

- **WHEN** the TUI starts with 10 running containers
- **THEN** the UI is immediately interactive and update results appear per image as each check finishes, with a per-image `checking…` indicator until then

### Requirement: Manual refresh

The system SHALL re-run update checks for all updatable running containers when the user presses `r`.

#### Scenario: Manual refresh

- **WHEN** the user presses `r` on the Updates tab
- **THEN** all updatable running containers are re-checked and rows return to the `checking…` state until results arrive

### Requirement: Not-updatable classification

The system SHALL classify and label images that cannot be checked: locally built images without repo digests, images pinned by digest, and containers excluded as managed workloads.

#### Scenario: Locally built image

- **WHEN** a container's image has no repo digest (built locally, never pushed)
- **THEN** its row shows `local build` and it is not offered for update

#### Scenario: Digest-pinned image

- **WHEN** a container was started from `image@sha256:…`
- **THEN** its row shows `pinned by digest` and it is not offered for update

### Requirement: Per-image check failure isolation

The system SHALL surface a per-image check error (e.g. registry unreachable, auth denied) on that image's row without affecting other images' checks.

#### Scenario: One registry down

- **WHEN** checks run for images on two registries and one registry is unreachable
- **THEN** only that registry's rows show `check failed`; the other registry's rows complete normally
