## ADDED Requirements

### Requirement: Overall screen layout

The TUI SHALL render four regions on every screen: a header (top), a tab bar (below header), a content area (fills remaining space), and a footer (bottom). The reference layout at 100×30 is:

```
┌ dockupdate ─ Podman 6.0.1 · unix://…/podman-machine-default-api.sock ── 3 updates ┐
│ [1 Services]  [2 Networks]  [3 Updates]                                           │
│                                                                                    │
│  <content area>                                                                    │
│                                                                                    │
│ tab/1-3 switch · r refresh · space select · enter apply · q quit                   │
└────────────────────────────────────────────────────────────────────────────────────┘
```

#### Scenario: Layout renders on startup

- **WHEN** the TUI starts in a terminal of at least 80×24
- **THEN** header, tab bar, content area, and footer are all visible in the arrangement above

### Requirement: Header content

The header SHALL show the application name, the connected engine kind and version, the socket in use (shortened when too long), and a badge with the count of containers that have updates available (hidden when zero).

#### Scenario: Header with updates

- **WHEN** three containers have updates available
- **THEN** the header badge shows `3 updates`

#### Scenario: Header without engine

- **WHEN** no engine is connected
- **THEN** the header shows the application name and a `not connected` indicator instead of engine info

### Requirement: Tab bar and navigation

The tab bar SHALL show the three tabs `Services`, `Networks`, `Updates` with the active tab visually highlighted. The user SHALL be able to switch tabs with `tab`/`shift+tab` (cycle) and with the number keys `1`, `2`, `3` (direct).

#### Scenario: Direct tab switch

- **WHEN** the user presses `2` while on Services
- **THEN** the Networks tab becomes active and highlighted

#### Scenario: Cycling wraps

- **WHEN** the user presses `tab` while on Updates (the last tab)
- **THEN** the Services tab becomes active

### Requirement: Footer content

The footer SHALL show the keybindings relevant to the current tab and view state, and SHALL change when the view state changes (e.g. during an active update run).

#### Scenario: Updates tab footer while idle

- **WHEN** the Updates tab is active and no update is running
- **THEN** the footer includes `space select`, `a all`, `enter apply`, `r refresh`

#### Scenario: Footer while updates running

- **WHEN** at least one update is in flight
- **THEN** the footer replaces `enter apply` with an indication that updates are running and selection keys are disabled for in-flight rows

### Requirement: Services tab layout

The Services tab SHALL render a collapsible tree: compose projects as top-level groups (with a collapse indicator and container count), services nested beneath, containers as leaves. Each container row SHALL show state icon (running/exited), name, image:tag, and an update badge `⬆` when an update is available. Standalone containers appear under a `Standalone` top-level group.

#### Scenario: Tree rendering

- **WHEN** project `webapp` has services `web` (1 container, update available) and `db` (1 container), and one standalone container exists
- **THEN** the tree shows `▾ webapp (2)` with nested `web` and `db` groups, the `web` container row carrying a `⬆` badge, followed by a `▾ Standalone (1)` group

#### Scenario: Collapse and expand

- **WHEN** the user presses `enter` on the focused `▾ webapp` group
- **THEN** the group collapses to `▸ webapp (2)` hiding its children, and pressing `enter` again re-expands it

### Requirement: Networks tab layout

The Networks tab SHALL render a table of networks (name, driver, subnet, containers count). Pressing `enter` on a network SHALL replace the list with a detail view titled with the network name, listing each connected container's name, IPv4 address, and compose project. `esc` returns to the list.

#### Scenario: Detail view for network `main`

- **WHEN** the user opens network `main` which has containers `web` (10.89.0.2, project `webapp`) and `redis` (10.89.0.3, standalone)
- **THEN** the detail view shows a header `network: main` and rows for `web` with `10.89.0.2 · webapp` and `redis` with `10.89.0.3 · standalone`

### Requirement: Updates tab layout

The Updates tab SHALL render one row per updatable running container: a checkbox, container name, image:tag, and a status field. The status field SHALL be exactly one line per row and progress through these visual states:

```
[ ] web    nginx:1.25     update available
[ ] web    nginx:1.25     checking…
[x] web    nginx:1.25     ⠋ pulling  ████████░░░░░░ 45%  45MB/100MB
[x] web    nginx:1.25     ⠋ verifying checksum…
[x] web    nginx:1.25     ⠋ restarting service…
[x] web    nginx:1.25     ✔ updated in 12s
[x] cache  redis:7        ✖ failed: checksum mismatch
[ ] db     postgres:16    up to date
[ ] local  myapp:dev      local build (not updatable)
```

#### Scenario: One-liner state transitions

- **WHEN** the user applies a checked update
- **THEN** that row's status field transitions through the states above in order, never occupying more than one terminal line

#### Scenario: Section split

- **WHEN** some containers have updates and others are up to date or not updatable
- **THEN** the Updates tab shows updatable rows first, followed by a `not updatable / up to date` section

### Requirement: Empty and error states

Every tab SHALL render an explicit empty state when its data is empty, and a full-screen error state SHALL replace the content area when the engine is unreachable.

#### Scenario: No containers

- **WHEN** the engine has zero containers
- **THEN** the Services tab shows `no containers found` with a hint to refresh

#### Scenario: Engine unreachable

- **WHEN** the engine connection fails
- **THEN** the content area shows the unreachable message, the probed sockets, and `r retry`

### Requirement: Keybinding reference

The global keybindings SHALL be: `q`/`ctrl+c` quit, `tab`/`shift+tab` cycle tabs, `1`/`2`/`3` direct tabs, `r` refresh, `↑`/`↓`/`j`/`k` move, `enter` expand/open/apply (context-dependent), `esc` back, `space` toggle checkbox, `a` toggle all (Updates tab).

#### Scenario: Help footer matches behavior

- **WHEN** the user presses any keybinding listed in the footer
- **THEN** the corresponding documented action occurs

### Requirement: Quit confirmation during updates

The system SHALL ask for confirmation before quitting while any update is in flight; quitting only proceeds after confirmation.

#### Scenario: Quit with updates running

- **WHEN** the user presses `q` while an update is in flight
- **THEN** a confirmation prompt appears and the program exits only after the user confirms
