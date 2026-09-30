## Context

The release pipeline gained keyless cosign signing (`.sigstore.json` bundles, cosign v3 `--bundle` flow) and SLSA provenance in #10/#11, but the spec still describes "archives plus a checksums file" and the README only documents checksum verification. No SBOMs are produced anywhere.

## Goals / Non-Goals

**Goals:**
- SBOMs generated and attached on every release, covered by the existing attestation.
- Spec matches reality (signatures + provenance) plus the new SBOM behavior.
- README gives users a complete verification path.

**Non-Goals:**
- Signing the SBOMs individually (they ride on the provenance attestation; per-artifact cosign already covers binaries/archives).
- SBOM-driven vulnerability scanning in CI (govulncheck already runs; OSV scanner integration is separate).
- Changing the npm/Homebrew flows.

## Decisions

**goreleaser `sboms` block with the default syft generator.** `artifacts: archives` produces one SBOM per archive; `documents` gets uploaded automatically. Syft ships inside the goreleaser action environment — no extra installer step. Format: default (CycloneDX) is fine; SPDX adds nothing for this audience today.

**Extend the attestation `subject-path`, not a new step.** Add `dist/*.sbom` (exact glob per goreleaser's output naming) to the existing `attest-build-provenance` input; one provenance bundle stays attached as `provenance.intoto.jsonl`.

**Spec drift fixed in the same change.** Rewriting the publishing requirement to describe today's signing/provenance behavior (already live) plus SBOMs keeps one coherent delta instead of pretending the signatures don't exist.

**README verification section:** order the instructions checksum → cosign bundle → provenance, with the exact `cosign verify-blob` command per platform archive.

## Risks / Trade-offs

- [syft not present in the runner] → goreleaser errors loudly at the sbom phase, failing the release pre-publish — acceptable fail-fast behavior; alternatively pin an anchore/sbom-action step (rejected: one more action to pin for something goreleaser does natively).
- [SBOM asset naming breaks npm/Homebrew asset assumptions] → Those consume only `*.tar.gz` and `checksums.txt`; additive assets are ignored.
- [Attestation subject glob misses the SBOM names] → Mitigated by a tasks-item: verify the actual goreleaser output filenames in a snapshot build before wiring the glob.
