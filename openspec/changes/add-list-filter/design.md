## Context

The three tabs maintain parallel cursor/offset state (`svcCursor`/`svcOffset`, `netCursor`/`netOffset`, `updCursor`/`updOffset`) over full row lists. There is no modal input state today — keys are either global or routed to the active tab.

## Goals / Non-Goals

**Goals:**
- One filter implementation shared by all three tabs.
- Filter never mutates model data — it only selects a view subset.
- Cursor/scroll always valid under any filter, including filter-while-scrolled.

**Non-Goals:**
- Regex or field-qualified queries (`image:redis`) — substring only.
- Filtering the network *detail* view (small lists; `esc` already means back there).
- Persisting filters across refreshes is in scope (state survives `r`), but not across restarts.

## Decisions

**Modal filter state on the model: `filterText string` (applied) + `filterEditing bool` + edit buffer.** When `filterEditing` is true, `handleKey` routes keys to the buffer before global keys — except `enter`/`esc`/`tab` which close the prompt. `tab` closes the prompt *and* switches tab (filter stays applied; it is per-model, not per-tab — see below).

**One filter per model, not per tab.** Simpler state, and switching tabs with an active filter just filters the new tab's rows. Alternative considered: per-tab filters — rejected as surprising (hidden state the user can't see) for marginal benefit.

**Filtering at render/row-access, not in the row slices.** Each tab gets a `visibleIndices()` helper returning the matching row indices; cursor and offset then index into that subset. `rebuildUpdateRows`/`sortUpdateRows` keep operating on full data. The scroll-window helpers already work on `(cursor, offset, total, visible)` — they receive the filtered total. Cursor is clamped whenever the filter changes.

**Match predicate:** case-insensitive `strings.Contains` on `name` and `image` (containers), `name` (networks). Compose group headers on the Services tab render only when they have at least one visible descendant.

**Footer surfaces state:** while editing, the footer shows `filter: <text>` with a cursor; while applied, a dimmed `filter: text (esc to clear)` hint joins the keybinding list.

## Risks / Trade-offs

- [`q` quits while the user meant to type q in the filter] → Filter mode intercepts printable keys, so `q` only quits when the prompt is closed — the standard k9s behavior users expect.
- [Filtered-out in-flight tasks become invisible] → Accepted: the header badge and the footer "updates running" hint still signal activity; clearing the filter reveals rows. (A persistent in-flight summary line is a possible follow-up.)
- [Interaction with the "not updatable" section header] → The header renders only when at least one note-row passes the filter; windowing math (`updVisibleRows`) uses the filtered row set.
