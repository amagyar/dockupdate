## 1. SBOM generation

- [ ] 1.1 Add an `sboms` block to `.goreleaser.yml` covering the release archives
- [ ] 1.2 Run a local `goreleaser release --snapshot --clean` to confirm SBOM filenames and that they land in `dist/`
- [ ] 1.3 Extend the `attest-build-provenance` step's `subject-path` in `.github/workflows/release.yml` to include the SBOM files

## 2. Docs

- [ ] 2.1 README binary-install section: document checksum verification, `cosign verify-blob --bundle <artifact>.sigstore.json <artifact>`, and the provenance file
- [ ] 2.2 Check `AGENTS.md` for stale asset references (`.sig`/`.pem`) and update if present

## 3. Verification

- [ ] 3.1 `goreleaser check` passes
- [ ] 3.2 After the next tag, confirm the release assets include SBOMs and that the provenance bundle attests them
