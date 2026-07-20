# Contributing to dockupdate

## Getting Started

```sh
go build ./...
go vet ./... && go test ./...
```

See the [Development](README.md#development) section for more details.

## Pull Request Process

1. Fork the repository and create a branch from `main`.
2. Make your changes and ensure `go vet ./... && go test ./...` passes.
3. Run `go test -race ./...` to check for data races.
4. Add tests for new functionality.
5. Update documentation if your change affects user-facing behavior.
6. Open a pull request with a clear description of the change and why it should be merged.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep functions small and focused.
- Use meaningful variable names.
- Add comments for exported symbols where the intent is not obvious from the signature.

## Reporting Bugs

Open an issue on [GitHub Issues](https://github.com/adev/dockupdate/issues) with:

- `dockupdate --version` output
- Operating system and arch
- Engine (Docker or Podman) and version
- Steps to reproduce
- Expected vs actual behavior

## Feature Requests

Open a [GitHub Issue](https://github.com/adev/dockupdate/issues) and describe the use case —
what problem would this feature solve for you?
