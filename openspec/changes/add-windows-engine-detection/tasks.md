## 1. Platform-aware candidates

- [ ] 1.1 Add a `GOOS` field to `engine.Environment`, defaulted from `runtime.GOOS` in `DefaultEnvironment`
- [ ] 1.2 Branch `Candidates`: on `windows` return `[npipe:////./pipe/docker_engine, PodmanSocket()]` (plus override/env precedence as today); on other GOOS keep the current unix list exactly
- [ ] 1.3 Verify `normalize` passes `npipe://` through unchanged (add a unit test if missing)

## 2. Tests

- [ ] 2.1 Unit test: `Candidates` with `GOOS: "windows"` yields the npipe path first and no unix paths
- [ ] 2.2 Unit test: override/env precedence still collapses the list on windows
- [ ] 2.3 Regression: existing unix candidate-ordering tests pass unchanged
- [ ] 2.4 `GOOS=windows go build ./...` and `GOOS=windows go vet ./...` pass

## 3. Docs

- [ ] 3.1 README: document the Windows defaults (Docker Desktop named pipe, Podman machine) in the Features/Usage section
