# AGENTS.md

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <subject>
```

- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`
- **Subject**: imperative mood, lowercase, no trailing period, max 72 chars
- **Scope**: optional, the affected area — e.g. `feat(tui):`, `fix(registry):`, `ci:`
- **Breaking changes**: append `!` after type/scope (`feat(api)!: ...`) or add a `BREAKING CHANGE:` footer
- **Body** (optional): explain what and why, wrap at 72 chars

Examples:

- `feat(updates): add --prune flag for old images`
- `fix(registry): handle multi-arch index digest mismatch`
- `ci: pin workflow actions by sha`
- `chore: initial commit`

## Development

```sh
go build ./...                            # build
go vet ./... && go test ./...             # checks
go test -race ./...                       # race detector
go test -tags live ./internal/engine/     # live engine smoke tests (needs Docker/Podman)
CGO_ENABLED=0 go build -o dockupdate ./cmd/dockupdate
```

Layout: `cmd/dockupdate` (entrypoint), `internal/engine` (Engine API + socket detection),
`internal/registry` (digest checks + auth), `internal/compose` (provider detection),
`internal/updater` (update pipeline), `internal/tui` (Bubble Tea UI).

## Workflows

GitHub Actions are pinned by full SHA (with a version comment) and use
`step-security/harden-runner` as the first step of every job. Keep both when
editing workflows. Dependabot updates the pins.
