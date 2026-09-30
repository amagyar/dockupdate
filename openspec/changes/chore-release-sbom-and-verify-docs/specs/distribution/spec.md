## ADDED Requirements

### Requirement: Software bill of materials

The system SHALL generate SBOM documents (CycloneDX or SPDX) covering the release archives on every release and SHALL attach them to the GitHub Release alongside the other assets. The SBOM files SHALL be included in the build-provenance attestation subjects.

#### Scenario: Release carries SBOMs

- **WHEN** a version tag is pushed and the release workflow completes
- **THEN** the GitHub Release contains SBOM file(s) describing the Go module dependencies of the built binaries

#### Scenario: SBOM covered by provenance

- **WHEN** the attestation step runs
- **THEN** the SBOM files are among the attested subjects in `provenance.intoto.jsonl`

### Requirement: Verification documentation

The README SHALL document how users verify downloaded release artifacts: checksum verification against `checksums.txt`, signature verification via `cosign verify-blob --bundle <artifact>.sigstore.json <artifact>`, and the purpose of `provenance.intoto.jsonl`.

#### Scenario: User verifies a download

- **WHEN** a user follows the README's binary-install section
- **THEN** they can verify both the checksum and the cosign signature bundle of the archive using only the documented commands

## MODIFIED Requirements

### Requirement: GitHub Releases publishing

The system SHALL publish versioned archives (tar.gz) plus a checksums file to GitHub Releases when a version tag is pushed. Every artifact (archives and checksums) SHALL be signed with keyless cosign, attaching one `<artifact>.sigstore.json` bundle per artifact. The release SHALL also carry `provenance.intoto.jsonl` (SLSA build provenance) and SBOM documents.

#### Scenario: Tag triggers release

- **WHEN** a tag `v0.1.0` is pushed
- **THEN** the release workflow builds all targets and attaches archives, checksums, one `.sigstore.json` per artifact, SBOMs, and the provenance file to a GitHub Release named `v0.1.0`

#### Scenario: Signature verification

- **WHEN** a user downloads an archive and its `.sigstore.json` bundle
- **THEN** `cosign verify-blob --bundle <bundle> <archive>` succeeds, proving the artifact was built and signed by the repository's release workflow
