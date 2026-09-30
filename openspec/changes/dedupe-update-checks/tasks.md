## 1. Registry package

- [ ] 1.1 Factor the comparison half of `Checker.Check` into a pure function (ref + local digests + remote candidates → `Result`), leaving `Check` as a thin composition
- [ ] 1.2 Add a singleflight-style per-ref memoizer around the remote lookup (shared in-flight call, result+error broadcast, no new dependency)

## 2. TUI wiring

- [ ] 2.1 Create one memoizing checker per check pass in `startChecksCmd` and thread it through `checkCmd`; drop the per-container `NewChecker` construction
- [ ] 2.2 Ensure a fresh cache per pass (startup, `r`, post-update refresh)

## 3. Tests

- [ ] 3.1 Registry test: N concurrent checks for one ref hit the test registry exactly once (count requests in the httptest handler)
- [ ] 3.2 Registry test: same ref with different local digest sets classifies each container correctly from one lookup
- [ ] 3.3 Registry test: a failed shared lookup marks all containers on that ref failed, other refs unaffected
- [ ] 3.4 Existing fuzz targets and registry tests still pass (`go test ./internal/registry/`)
