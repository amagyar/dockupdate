## Why

Two loose ends in the release supply chain. First, releases ship no SBOM — users and security tooling cannot enumerate what is inside the binary, and the SBOM is a standard complement to the cosign signatures and SLSA provenance added in #10. Second, the distribution spec and README predate that work: the spec promises only "archives plus a checksums file" while releases actually attach `.sigstore.json` bundles and `provenance.intoto.jsonl`, and the README's binary-install section tells users to "verify against checksums.txt" without mentioning the signatures they could verify instead.

## What Changes

- goreleaser generates a CycloneDX/SPDX SBOM per release (covering all archives) and uploads it with the other assets; the release attestation step covers the SBOM files too.
- The distribution spec is brought up to date with reality: signed artifacts (sigstore bundles) and provenance attestation are now spec'd release behavior, plus the new SBOM requirement.
- README's binary-install section documents `cosign verify-blob --bundle` and the provenance file, replacing the checksums-only verification story.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `distribution`: the "GitHub Releases publishing" requirement changes — releases attach sigstore signature bundles, SLSA provenance, and SBOMs in addition to archives and checksums; and a new requirement covers user-facing verification documentation.

## Impact

- **Code/config**: `.goreleaser.yml` (`sboms` block), `.github/workflows/release.yml` (attestation `subject-path` extended), `README.md`, `AGENTS.md` release notes if asset names are referenced.
- **Consumers**: npm wrapper and Homebrew cask are unaffected (they consume archives + checksums only).
- **Scorecard**: SBOMs strengthen the supply-chain posture in the same spirit as #10.
