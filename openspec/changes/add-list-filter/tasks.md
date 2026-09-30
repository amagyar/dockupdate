## 1. Filter state and key routing

- [ ] 1.1 Add filter state to the model (`filterText`, edit buffer, `filterEditing`); route keys to the filter buffer while editing (`enter` applies, `esc` cancels edit, printable keys edit)
- [ ] 1.2 `/` opens the prompt on Services/Networks/Updates; `esc` with prompt closed clears an applied filter
- [ ] 1.3 Clamp cursors/offsets whenever the applied filter changes

## 2. Per-tab filtering

- [ ] 2.1 Add a shared match helper (case-insensitive substring on name+image for containers, name for networks)
- [ ] 2.2 Services tab: filter container leaves; render project/service group headers only when they contain a visible container
- [ ] 2.3 Networks tab: filter the network list (detail view unchanged)
- [ ] 2.4 Updates tab: filter rows; render the "not updatable" header only when a filtered note-row exists; fix `updVisibleRows` math for the filtered set
- [ ] 2.5 Explicit `no matches` empty state per tab

## 3. Footer/rendering

- [ ] 3.1 Show the prompt while editing and a dimmed `filter: text (esc to clear)` hint when applied
- [ ] 3.2 Update the footer keybinding hints to include `/`

## 4. Tests

- [ ] 4.1 Model tests: typing filters rows, `enter` applies, `esc` cancel keeps previous filter, `esc` on list clears
- [ ] 4.2 Model test: filter mode intercepts `a`/`space` (no checkbox toggling while editing)
- [ ] 4.3 Model test: hidden rows keep checkbox/task state; cursor clamps when matches shrink
- [ ] 4.4 Model test: no-matches state renders
- [ ] 4.5 README keybindings table gains `/`
