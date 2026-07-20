## ADDED Requirements

### Requirement: Single static binary builds

The system SHALL be buildable as a single static binary (`CGO_ENABLED=0`) per target platform: linux, darwin, and windows on amd64 and arm64, with the version injected at link time.

#### Scenario: Local build produces one binary

- **WHEN** a developer runs the goreleaser build locally
- **THEN** one self-contained binary per target platform is produced with no runtime dependencies

### Requirement: GitHub Releases publishing

The system SHALL publish versioned archives (tar.gz, zip for windows) plus a checksums file to GitHub Releases when a version tag is pushed.

#### Scenario: Tag triggers release

- **WHEN** a tag `v0.1.0` is pushed
- **THEN** the release workflow builds all targets and attaches archives and checksums to a GitHub Release named `v0.1.0`

### Requirement: Homebrew tap formula

The system SHALL publish a Homebrew formula for `dockupdate` to a dedicated tap repository on each release, installable via `brew install <owner>/tap/dockupdate`.

#### Scenario: Formula updated on release

- **WHEN** release `v0.1.0` is published
- **THEN** the tap repository's formula references the `v0.1.0` darwin/linux archives with matching SHA256 checksums

### Requirement: npm wrapper package

The system SHALL publish an npm package `dockupdate` whose postinstall script downloads the release archive matching the package version and the user's platform/arch from GitHub Releases, verifies its checksum, and installs the binary.

#### Scenario: Install on macOS arm64

- **WHEN** a user runs `npm install -g dockupdate` on macOS arm64 for package version `0.1.0`
- **THEN** the postinstall script downloads the `darwin/arm64` archive of release `v0.1.0`, verifies it against the published checksums, and places the `dockupdate` binary on the user's PATH

#### Scenario: Unsupported platform fails clearly

- **WHEN** the postinstall script runs on an unsupported platform/arch combination
- **THEN** it exits with a clear error naming the platform and the supported set, without leaving a broken binary shim

### Requirement: Release workflow prerequisites

The release workflow SHALL fail fast with an explicit message when required secrets or repositories (GitHub release repo, homebrew tap repo, npm token) are missing.

#### Scenario: Missing npm token

- **WHEN** the release workflow runs without the npm token secret configured
- **THEN** the workflow stops before publishing with an error naming the missing secret
