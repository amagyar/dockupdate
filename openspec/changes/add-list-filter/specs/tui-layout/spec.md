## ADDED Requirements

### Requirement: List filtering

On the Services, Networks, and Updates tabs, pressing `/` SHALL open an inline filter prompt. While the prompt is open, typed characters SHALL narrow the visible rows to those matching the entered text as a case-insensitive substring (containers: name and image reference; networks: name). Pressing `enter` SHALL apply the filter and return focus to the list; pressing `esc` in the prompt SHALL cancel the edit, keeping the previously applied filter; pressing `esc` on a filtered list (prompt closed) SHALL clear the filter. Filtering SHALL NOT alter underlying state: checkbox selections, running tasks, and check results persist independent of visibility.

#### Scenario: Filter updates by image name

- **WHEN** the Updates tab lists 20 containers and the user types `/redis` then `enter`
- **THEN** only rows whose name or image contains `redis` remain visible and navigable

#### Scenario: Filter preserves selection and task state

- **WHEN** a row is checked or has an update in flight and a filter hides it
- **THEN** its checkbox state and running task continue unaffected, and reappear when the filter is cleared

#### Scenario: Esc clears an applied filter

- **WHEN** a list is filtered and the user presses `esc` with the prompt closed
- **THEN** the filter clears and all rows reappear with the cursor position clamped into range

#### Scenario: Prompt cancel keeps previous filter

- **WHEN** a filter `web` is applied, the user opens the prompt, types `db`, and presses `esc`
- **THEN** the list stays filtered by `web`

#### Scenario: No matches

- **WHEN** the filter text matches no row
- **THEN** the list area shows an explicit no-matches state instead of a blank screen

## MODIFIED Requirements

### Requirement: Keybinding reference

The global keybindings SHALL be: `q`/`ctrl+c` quit, `tab`/`shift+tab` cycle tabs, `1`/`2`/`3` direct tabs, `r` refresh, `↑`/`↓`/`j`/`k` move, `enter` expand/open/apply (context-dependent), `esc` back / clear filter, `space` toggle checkbox, `a` toggle all (Updates tab), `/` filter (list tabs). While the filter prompt is open, printable keys edit the filter and list actions are suspended.

#### Scenario: Help footer matches behavior

- **WHEN** the user presses any keybinding listed in the footer
- **THEN** the corresponding documented action occurs

#### Scenario: Filter mode intercepts list keys

- **WHEN** the filter prompt is open on the Updates tab and the user types `a`
- **THEN** the letter `a` is appended to the filter text instead of toggling all checkboxes
