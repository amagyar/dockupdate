package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adev/dockupdate/internal/engine"
	"github.com/adev/dockupdate/internal/registry"
	"github.com/adev/dockupdate/internal/updater"
)

// fakeEngine implements engineAPI for TUI tests.
type fakeEngine struct {
	info       engine.Info
	containers []engine.Container
	networks   []engine.Network
	digests    map[string][]string
	recreated  []string
}

func (f *fakeEngine) Info() engine.Info { return f.info }
func (f *fakeEngine) Socket() string    { return "unix:///fake.sock" }
func (f *fakeEngine) Close() error      { return nil }

func (f *fakeEngine) ListContainers(ctx context.Context) ([]engine.Container, error) {
	return f.containers, nil
}

func (f *fakeEngine) ListNetworks(ctx context.Context) ([]engine.Network, error) {
	return f.networks, nil
}

func (f *fakeEngine) FillNetworkProjects(ctx context.Context, nets []engine.Network) error {
	return nil
}

func (f *fakeEngine) RepoDigests(ctx context.Context, ref string) ([]string, error) {
	return f.digests[ref], nil
}

func (f *fakeEngine) PullImage(ctx context.Context, ref, auth string, progress func(engine.Progress)) error {
	if progress != nil {
		progress(engine.Progress{Current: 50, Total: 100, LayersDone: 1, LayersTotal: 2})
		progress(engine.Progress{Current: 100, Total: 100, LayersDone: 2, LayersTotal: 2})
	}
	return nil
}

func (f *fakeEngine) RecreateContainer(ctx context.Context, id, newImage string) error {
	f.recreated = append(f.recreated, id)
	return nil
}

func (f *fakeEngine) ImageID(ctx context.Context, containerID string) (string, error) {
	return "sha256:old", nil
}

func (f *fakeEngine) ImageIDForRef(ctx context.Context, ref string) (string, error) {
	return "sha256:new", nil
}

func (f *fakeEngine) RemoveImage(ctx context.Context, imageID string) error { return nil }

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case " ":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func sized(t *testing.T, m Model) Model {
	t.Helper()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return nm.(Model)
}

func composeContainer(id, name, project, service, image string) engine.Container {
	return engine.Container{
		ID: id, Name: name, Image: image, ImageID: "img-" + id, State: "running",
		Labels: map[string]string{
			"com.docker.compose.project": project,
			"com.docker.compose.service": service,
		},
	}
}

func TestTabSwitching(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	if m.active != TabServices {
		t.Fatalf("initial tab = %v", m.active)
	}

	nm, _ := m.Update(key("tab"))
	m = nm.(Model)
	if m.active != TabNetworks {
		t.Fatalf("tab -> %v", m.active)
	}

	nm, _ = m.Update(key("tab"))
	m = nm.(Model)
	if m.active != TabUpdates {
		t.Fatalf("tab -> %v", m.active)
	}

	// Cycling wraps to the first tab.
	nm, _ = m.Update(key("tab"))
	m = nm.(Model)
	if m.active != TabServices {
		t.Fatalf("wrap -> %v", m.active)
	}

	nm, _ = m.Update(key("shift+tab"))
	m = nm.(Model)
	if m.active != TabUpdates {
		t.Fatalf("shift+tab -> %v", m.active)
	}

	nm, _ = m.Update(key("2"))
	m = nm.(Model)
	if m.active != TabNetworks {
		t.Fatalf("direct 2 -> %v", m.active)
	}
	nm, _ = m.Update(key("1"))
	m = nm.(Model)
	if m.active != TabServices {
		t.Fatalf("direct 1 -> %v", m.active)
	}
	nm, _ = m.Update(key("3"))
	m = nm.(Model)
	if m.active != TabUpdates {
		t.Fatalf("direct 3 -> %v", m.active)
	}
}

func TestServicesTreeGroupingAndCollapse(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	m.containers = []engine.Container{
		composeContainer("c1", "web-1", "webapp", "web", "nginx:1.25"),
		composeContainer("c2", "db-1", "webapp", "db", "postgres:16"),
		{ID: "c3", Name: "cache", Image: "redis:7", ImageID: "img-c3", State: "running", Labels: map[string]string{}},
	}
	m.rebuildGroups()

	// Expect: webapp group, web service, web-1, db service, db-1, Standalone group, cache.
	if len(m.svcRows) != 7 {
		t.Fatalf("rows = %d, want 7: %+v", len(m.svcRows), m.svcRows)
	}
	if m.svcRows[0].kind != rowProject || m.svcRows[0].group.name != "webapp" {
		t.Fatalf("first row must be webapp project: %+v", m.svcRows[0])
	}
	if m.svcRows[5].kind != rowProject || m.svcRows[5].group.name != "Standalone" {
		t.Fatalf("standalone group expected at row 5: %+v", m.svcRows[5])
	}
	if m.svcRows[6].kind != rowContainer || m.svcRows[6].entry.c.Name != "cache" {
		t.Fatalf("standalone container at row 6: %+v", m.svcRows[6])
	}

	// Collapse the project with enter.
	nm, _ := m.Update(key("enter"))
	m = nm.(Model)
	if m.svcRows[0].group.expanded {
		t.Fatal("project must be collapsed after enter")
	}
	if len(m.svcRows) != 3 { // webapp ▸, Standalone ▾, cache
		t.Fatalf("collapsed rows = %d, want 3", len(m.svcRows))
	}
	view := m.servicesView()
	if !strings.Contains(view, "▸ webapp (2)") {
		t.Fatalf("collapsed indicator missing: %q", view)
	}

	// Expand again.
	nm, _ = m.Update(key("enter"))
	m = nm.(Model)
	if !m.svcRows[0].group.expanded || len(m.svcRows) != 7 {
		t.Fatal("project must re-expand")
	}
}

func TestUpdateBadgeShown(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	c := composeContainer("c1", "web-1", "webapp", "web", "nginx:1.25")
	m.containers = []engine.Container{c}
	m.updRows = []*updateRow{{container: c, result: registry.Result{Kind: registry.KindUpdateAvailable}}}
	m.rebuildGroups()

	view := m.servicesView()
	if !strings.Contains(view, "⬆") {
		t.Fatalf("update badge missing: %q", view)
	}
}

func TestNetworksDrillDown(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	nm, _ := m.Update(networksMsg{networks: []engine.Network{
		{Name: "main", Driver: "bridge", Subnet: "10.89.0.0/24", Containers: []engine.NetworkContainer{
			{ID: "c1", Name: "web", IPv4: "10.89.0.2", Project: "webapp"},
			{ID: "c2", Name: "redis", IPv4: "10.89.0.3"},
		}},
		{Name: "empty", Driver: "bridge"},
	}})
	m = nm.(Model)
	m.active = TabNetworks

	view := m.networksView()
	if !strings.Contains(view, "main") || !strings.Contains(view, "10.89.0.0/24") {
		t.Fatalf("network list missing data: %q", view)
	}

	// Open detail.
	nm, _ = m.Update(key("enter"))
	m = nm.(Model)
	if m.netDetail != 0 {
		t.Fatalf("netDetail = %d", m.netDetail)
	}
	detail := m.networksView()
	if !strings.Contains(detail, "network: main") ||
		!strings.Contains(detail, "web") || !strings.Contains(detail, "10.89.0.2") || !strings.Contains(detail, "webapp") ||
		!strings.Contains(detail, "redis") || !strings.Contains(detail, "standalone") {
		t.Fatalf("detail view wrong: %q", detail)
	}

	// esc returns, preserving selection.
	nm, _ = m.Update(key("esc"))
	m = nm.(Model)
	if m.netDetail != -1 || m.netCursor != 0 {
		t.Fatalf("esc must return to list preserving cursor: detail=%d cursor=%d", m.netDetail, m.netCursor)
	}

	// Empty network shows explicit message.
	nm, _ = m.Update(key("down"))
	m = nm.(Model)
	nm, _ = m.Update(key("enter"))
	m = nm.(Model)
	if !strings.Contains(m.networksView(), "no containers connected") {
		t.Fatalf("empty network message missing: %q", m.networksView())
	}
}

func TestUpdatesSelectionAndApplyFlow(t *testing.T) {
	m := sized(t, New(Options{Version: "test", Concurrency: 2}))
	fake := &fakeEngine{
		info:    engine.Info{Kind: "Podman", Version: "6.0.1", OSType: "linux", Arch: "arm64"},
		digests: map[string][]string{"redis:7": {"sha256:remote"}},
	}
	m.eng = fake
	m.info = fake.info

	c := engine.Container{ID: "c1", Name: "cache", Image: "redis:7", ImageID: "img-c1", State: "running", Labels: map[string]string{}}
	nm, _ := m.Update(containersMsg{containers: []engine.Container{c}})
	m = nm.(Model)
	m.active = TabUpdates

	if len(m.updRows) != 1 || !m.updRows[0].checking {
		t.Fatalf("row must start checking: %+v", m.updRows)
	}

	// Check result arrives: update available.
	nm, _ = m.Update(checkResultMsg{containerID: "c1", res: registry.Result{Kind: registry.KindUpdateAvailable, RemoteDigest: "sha256:remote"}})
	m = nm.(Model)
	if !m.updRows[0].selectable() {
		t.Fatal("row must be selectable after available result")
	}

	// Apply with empty selection is a no-op.
	nm, _ = m.Update(key("enter"))
	m = nm.(Model)
	if m.updRunning {
		t.Fatal("enter with no selection must not start updates")
	}

	// Toggle with space, then apply.
	nm, _ = m.Update(key(" "))
	m = nm.(Model)
	if !m.updRows[0].checked {
		t.Fatal("space must check the row")
	}
	nm, _ = m.Update(key("enter"))
	m = nm.(Model)
	if !m.updRunning || m.updEvents == nil {
		t.Fatal("apply must start the update pipeline")
	}

	// Pump worker events until finished.
	for i := 0; i < 100; i++ {
		msg := waitUpdateEvent(m.updEvents)()
		nm, _ := m.Update(msg)
		m = nm.(Model)
		if _, done := msg.(updatesFinishedMsg); done {
			break
		}
	}
	if m.updRunning {
		t.Fatal("pipeline must be finished")
	}
	row := m.updRows[0]
	if row.task == nil || row.task.Phase != updater.PhaseDone {
		t.Fatalf("task must be done: %+v", row.task)
	}
	if len(fake.recreated) != 1 || fake.recreated[0] != "c1" {
		t.Fatalf("standalone container must be recreated: %v", fake.recreated)
	}
	if !strings.Contains(m.rowStatus(row), "✔ updated") {
		t.Fatalf("success one-liner missing: %q", m.rowStatus(row))
	}
}

func TestRowStatusOneLiners(t *testing.T) {
	m := New(Options{Version: "test"})

	r := &updateRow{checking: true}
	if s := m.rowStatus(r); !strings.Contains(s, "checking…") {
		t.Fatalf("checking: %q", s)
	}

	r = &updateRow{note: "managed by orchestrator"}
	if s := m.rowStatus(r); s != "managed by orchestrator" {
		t.Fatalf("note: %q", s)
	}

	r = &updateRow{result: registry.Result{Kind: registry.KindUpdateAvailable}}
	if s := m.rowStatus(r); !strings.Contains(s, "update available") {
		t.Fatalf("available: %q", s)
	}

	r = &updateRow{result: registry.Result{Kind: registry.KindFailed, Err: errors.New("timeout")}}
	if s := m.rowStatus(r); !strings.Contains(s, "check failed: timeout") {
		t.Fatalf("failed: %q", s)
	}

	r = &updateRow{result: registry.Result{Kind: registry.KindLocalBuild}}
	if s := m.rowStatus(r); !strings.Contains(s, "local build") {
		t.Fatalf("local build: %q", s)
	}

	// Task pipeline states.
	task := &updater.Task{Phase: updater.PhasePulling, Current: 45 * 1e6, Total: 100 * 1e6}
	if s := m.taskStatus(task); !strings.Contains(s, "pulling") || !strings.Contains(s, "45%") || !strings.Contains(s, "45MB/100MB") {
		t.Fatalf("pulling one-liner: %q", s)
	}
	// Layer-count fallback when the engine reports no byte progress.
	task = &updater.Task{Phase: updater.PhasePulling, LayersDone: 1, LayersTotal: 4}
	if s := m.taskStatus(task); !strings.Contains(s, "pulling") || !strings.Contains(s, "1/4 layers") {
		t.Fatalf("pulling layers fallback: %q", s)
	}
	// Indeterminate state before any progress event.
	task = &updater.Task{Phase: updater.PhasePulling}
	if s := m.taskStatus(task); !strings.Contains(s, "pulling…") {
		t.Fatalf("pulling indeterminate: %q", s)
	}
	task.Phase = updater.PhaseVerifying
	if s := m.taskStatus(task); !strings.Contains(s, "verifying checksum") {
		t.Fatalf("verifying: %q", s)
	}
	task.Phase = updater.PhaseRestarting
	if s := m.taskStatus(task); !strings.Contains(s, "restarting service") {
		t.Fatalf("restarting: %q", s)
	}
	task.Phase = updater.PhaseDone
	task.Started = time.Now().Add(-12 * time.Second)
	task.Finished = time.Now()
	if s := m.taskStatus(task); !strings.Contains(s, "✔ updated in 12s") {
		t.Fatalf("done: %q", s)
	}
	task.Phase = updater.PhaseFailed
	task.Err = errors.New("checksum mismatch")
	if s := m.taskStatus(task); !strings.Contains(s, "✖ failed: checksum mismatch") {
		t.Fatalf("failed: %q", s)
	}
}

func TestEmptyAndErrorStates(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))

	if v := m.servicesView(); !strings.Contains(v, "no containers found") {
		t.Fatalf("services empty state: %q", v)
	}
	if v := m.networksView(); !strings.Contains(v, "no networks found") {
		t.Fatalf("networks empty state: %q", v)
	}
	if v := m.updatesView(); !strings.Contains(v, "no containers found") {
		t.Fatalf("updates empty state: %q", v)
	}

	// Engine unreachable error state with probed list and retry hint.
	m.connErr = errors.New("no reachable engine socket")
	m.probed = []string{"unix:///var/run/docker.sock", "unix:///podman.sock"}
	v := m.View()
	if !strings.Contains(v, "engine unreachable") ||
		!strings.Contains(v, "unix:///var/run/docker.sock") ||
		!strings.Contains(v, "press r to retry") {
		t.Fatalf("error view: %q", v)
	}
}

func TestTooSmallTerminal(t *testing.T) {
	m := New(Options{Version: "test"})
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	m = nm.(Model)
	if v := m.View(); !strings.Contains(v, "terminal too small") {
		t.Fatalf("too-small message missing: %q", v)
	}
}

func TestQuitConfirmationWhileUpdating(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	m.inFlight = 1

	nm, cmd := m.Update(key("q"))
	m = nm.(Model)
	if !m.confirmQuit || cmd != nil {
		t.Fatal("q with in-flight updates must ask for confirmation, not quit")
	}
	if v := m.View(); !strings.Contains(v, "really quit? y/n") {
		t.Fatalf("confirm footer missing: %q", v)
	}

	nm, _ = m.Update(key("n"))
	m = nm.(Model)
	if m.confirmQuit {
		t.Fatal("n must cancel the quit")
	}

	nm, _ = m.Update(key("q"))
	m = nm.(Model)
	nm, cmd = m.Update(key("y"))
	_ = nm
	if cmd == nil {
		t.Fatal("y must quit")
	}
}

func TestUpdateRowsSortAvailableFirst(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	containers := []engine.Container{
		{ID: "a", Name: "aaa", Image: "x:1", State: "running", Labels: map[string]string{}},
		{ID: "b", Name: "bbb", Image: "y:1", State: "running", Labels: map[string]string{}},
		{ID: "c", Name: "ccc", Image: "z:1", State: "running", Labels: map[string]string{}},
	}
	m.containers = containers
	m.rebuildUpdateRows()

	// Cursor on the first row (a), then b's check arrives as available.
	nm, _ := m.Update(checkResultMsg{containerID: "b", res: registry.Result{Kind: registry.KindUpdateAvailable, RemoteDigest: "sha256:r"}})
	m = nm.(Model)
	if m.updRows[0].container.ID != "b" {
		t.Fatalf("available row must sort first: %+v", m.updRows[0].container)
	}
	// Cursor stays anchored to the same container.
	if m.updRows[m.updCursor].container.ID != "a" {
		t.Fatalf("cursor must stay on container a, now on %q", m.updRows[m.updCursor].container.ID)
	}
}

func TestManualRefreshRechecks(t *testing.T) {
	m := sized(t, New(Options{Version: "test"}))
	m.eng = &fakeEngine{}
	c := engine.Container{ID: "c1", Name: "x", Image: "app:1", State: "running", Labels: map[string]string{}}
	m.containers = []engine.Container{c}
	m.rebuildUpdateRows()
	m.updRows[0].checking = false
	m.updRows[0].result = registry.Result{Kind: registry.KindUpToDate}

	nm, _ := m.Update(key("r"))
	m = nm.(Model)
	if !m.updRows[0].checking || m.updRows[0].result.Kind == registry.KindUpToDate {
		t.Fatal("r must reset updatable rows to checking state")
	}
}
