## 1. Endpoint settings preservation

- [ ] 1.1 Add a sanitize helper in `internal/engine` that copies an `EndpointSettings` from inspect, keeping user config (`IPAMConfig`, `Aliases`, `Links`, `DriverOpts`, `NetworkID`) and clearing runtime-assigned fields (`EndpointID`, runtime addresses, MAC)
- [ ] 1.2 Use sanitized settings in `RecreateContainer` for both the primary network (create) and extra networks (connect)

## 2. Health gate

- [ ] 2.1 Implement a post-start viability gate: poll inspect until `healthy` (healthcheck present, fast-fail on `unhealthy`, 60s timeout) or still-running after a ~2s grace (no healthcheck)
- [ ] 2.2 Only remove the old container after the gate passes

## 3. Rollback

- [ ] 3.1 On gate failure: remove the replacement, rename the old container back, start it; combine rollback errors with the gate error in the returned error
- [ ] 3.2 Keep the existing create/start failure paths (restore name, leave old container stopped) unchanged

## 4. Tests

- [ ] 4.1 Unit test the sanitize helper: aliases/IPAM/links kept, endpoint ID and runtime addresses cleared
- [ ] 4.2 Live test (`-tags live`): update a standalone container with a static IP + alias on a user-defined network; assert both are preserved
- [ ] 4.3 Live test: replacement image with a failing healthcheck rolls back to the old container running under the original name
- [ ] 4.4 README "How updates work" step 4: mention the health gate and automatic rollback
