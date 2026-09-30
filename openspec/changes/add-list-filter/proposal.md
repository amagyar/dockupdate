## Why

Hosts running dozens of containers make the Services and Updates tabs tedious: finding one container means scrolling a long list. Every comparable TUI (k9s, lazydocker, htop) offers incremental text filtering; dockupdate has none.

## What Changes

- `/` opens a filter prompt on the Services, Networks, and Updates tabs; typing filters the visible rows by case-insensitive substring match against container name and image reference (networks: name).
- `enter` applies the filter and returns focus to the list; `esc` in the prompt cancels the edit but keeps the previous filter; `esc` on a filtered list clears the filter.
- The active filter is shown in the footer area, and match counts are implied by what renders (no separate count UI).
- Filtering is a view concern only: selection state, running tasks, and check results are preserved regardless of what is visible.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `tui-layout`: new "List filtering" requirement, and the "Keybinding reference" requirement gains `/` plus the filter-mode keys.

## Impact

- **Code**: `internal/tui/model.go` (filter state, key routing in filter mode), `internal/tui/services.go`/`networks.go`/`updates.go` (filtered row views), `internal/tui/view.go` (footer/filter line rendering), TUI model tests.
- **UX**: `/` becomes a reserved key on list tabs; all other keybindings are unchanged outside filter mode.
