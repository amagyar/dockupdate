// Package tui implements the dockupdate terminal UI (Bubble Tea).
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/adev/dockupdate/internal/engine"
	"github.com/adev/dockupdate/internal/registry"
	"github.com/adev/dockupdate/internal/updater"
)

// Tab identifies the active view.
type Tab int

const (
	TabServices Tab = iota
	TabNetworks
	TabUpdates
	tabCount
)

// Options carries CLI flags into the TUI.
type Options struct {
	Socket      string
	Concurrency int
	Prune       bool
	Version     string
}

// updateRow is one container's row on the Updates tab.
type updateRow struct {
	container engine.Container
	checking  bool
	result    registry.Result
	note      string // non-registry classification ("managed", "not running")
	checked   bool
	task      *updater.Task
}

// engineAPI abstracts the engine client so TUI tests can use a fake.
// *engine.Client satisfies it.
type engineAPI interface {
	Info() engine.Info
	Socket() string
	Close() error
	ListContainers(ctx context.Context) ([]engine.Container, error)
	ListNetworks(ctx context.Context) ([]engine.Network, error)
	FillNetworkProjects(ctx context.Context, nets []engine.Network) error
	RepoDigests(ctx context.Context, imageRef string) ([]string, error)
	updater.Engine
}

// Model is the root Bubble Tea model.
type Model struct {
	opts Options

	width, height int
	active        Tab

	// connection state
	eng     engineAPI
	info    engine.Info
	connErr error
	probed  []string

	// inventory
	containers []engine.Container
	networks   []engine.Network

	// services tab
	groups    []*projectGroup
	svcRows   []svcRow
	svcCursor int

	// networks tab
	netCursor int
	netDetail int // -1: list mode; otherwise index into networks

	// updates tab
	updRows    []*updateRow
	updCursor  int
	inFlight   int
	updEvents  chan updater.Event
	updRunning bool

	confirmQuit bool
	spinner     spinner.Model
}

// Messages.

type connectMsg struct {
	eng    engineAPI
	probed []string
	err    error
}

type containersMsg struct {
	containers []engine.Container
	err        error
}

type networksMsg struct {
	networks []engine.Network
	err      error
}

type checkResultMsg struct {
	containerID string
	res         registry.Result
}

type updateEventMsg struct{ ev updater.Event }
type updatesFinishedMsg struct{}

// New creates the root model.
func New(opts Options) Model {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	return Model{
		opts:      opts,
		active:    TabServices,
		netDetail: -1,
		spinner:   sp,
	}
}

// Run starts the Bubble Tea program.
func Run(opts Options) error {
	p := tea.NewProgram(New(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, connectCmd(m.opts.Socket))
}

// Commands.

func connectCmd(socket string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		eng, probed, err := engine.Connect(ctx, engine.DefaultEnvironment(socket))
		return connectMsg{eng: eng, probed: probed, err: err}
	}
}

func loadContainersCmd(eng engineAPI) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		list, err := eng.ListContainers(ctx)
		return containersMsg{containers: list, err: err}
	}
}

func loadNetworksCmd(eng engineAPI) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		nets, err := eng.ListNetworks(ctx)
		if err != nil {
			return networksMsg{err: err}
		}
		_ = eng.FillNetworkProjects(ctx, nets)
		return networksMsg{networks: nets}
	}
}

// checkCmd checks one container's image for updates.
func checkCmd(eng engineAPI, info engine.Info, c engine.Container) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if registry.IsPinned(c.Image) {
			return checkResultMsg{containerID: c.ID, res: registry.Result{Kind: registry.KindPinned}}
		}
		digests, err := eng.RepoDigests(ctx, c.ImageID)
		if err != nil {
			return checkResultMsg{containerID: c.ID, res: registry.Result{Kind: registry.KindFailed, Err: err}}
		}
		checker := registry.NewChecker(registry.PlatformFromEngine(info.OSType, info.Arch), false)
		res := checker.Check(ctx, c.Image, digests)
		return checkResultMsg{containerID: c.ID, res: res}
	}
}

func waitUpdateEvent(ch <-chan updater.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return updatesFinishedMsg{}
		}
		return updateEventMsg{ev: ev}
	}
}

// Update.

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)

	case connectMsg:
		m.probed = msg.probed
		if msg.err != nil {
			m.connErr = msg.err
			m.eng = nil
			return m, nil
		}
		m.connErr = nil
		m.eng = msg.eng
		m.info = msg.eng.Info()
		return m, tea.Batch(loadContainersCmd(m.eng), loadNetworksCmd(m.eng))

	case containersMsg:
		if msg.err != nil {
			m.connErr = msg.err
			return m, nil
		}
		m.containers = msg.containers
		m.rebuildUpdateRows()
		m.rebuildGroups()
		return m, m.startChecksCmd()

	case networksMsg:
		if msg.err == nil {
			m.networks = msg.networks
			m.clampNetCursor()
		}
		return m, nil

	case checkResultMsg:
		for _, r := range m.updRows {
			if r.container.ID == msg.containerID {
				r.checking = false
				r.result = msg.res
				break
			}
		}
		m.sortUpdateRows()
		m.rebuildGroups() // refresh ⬆ badges
		return m, nil

	case updateEventMsg:
		m.applyUpdateEvent(msg.ev)
		if m.updEvents != nil {
			return m, waitUpdateEvent(m.updEvents)
		}
		return m, nil

	case updatesFinishedMsg:
		m.updRunning = false
		m.updEvents = nil
		// Spec: inventory refresh after updates complete.
		if m.eng != nil {
			return m, tea.Batch(loadContainersCmd(m.eng), loadNetworksCmd(m.eng))
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Quit confirmation dialog has its own key handling.
	if m.confirmQuit {
		switch key {
		case "y", "Y":
			return m, tea.Quit
		case "n", "N", "esc":
			m.confirmQuit = false
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.inFlight > 0 {
			m.confirmQuit = true
			return m, nil
		}
		return m, tea.Quit
	case "tab":
		m.active = (m.active + 1) % tabCount
		return m, nil
	case "shift+tab":
		m.active = (m.active + tabCount - 1) % tabCount
		return m, nil
	case "1":
		m.active = TabServices
		return m, nil
	case "2":
		m.active = TabNetworks
		return m, nil
	case "3":
		m.active = TabUpdates
		return m, nil
	case "r":
		return m.refresh()
	}

	switch m.active {
	case TabServices:
		return m.servicesKey(key)
	case TabNetworks:
		return m.networksKey(key)
	case TabUpdates:
		return m.updatesKey(key)
	}
	return m, nil
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	if m.eng == nil {
		// Not connected: retry detection.
		m.connErr = nil
		return m, connectCmd(m.opts.Socket)
	}
	// Spec: manual refresh re-checks all updatable running containers.
	for _, r := range m.updRows {
		if r.task == nil && r.note == "" {
			r.checking = true
			r.result = registry.Result{}
		}
	}
	return m, tea.Batch(loadContainersCmd(m.eng), loadNetworksCmd(m.eng))
}

// startChecksCmd launches background digest checks for all updatable rows.
func (m Model) startChecksCmd() tea.Cmd {
	var cmds []tea.Cmd
	for _, r := range m.updRows {
		if !r.checking {
			continue
		}
		cmds = append(cmds, checkCmd(m.eng, m.info, r.container))
	}
	return tea.Batch(cmds...)
}

// rebuildUpdateRows rebuilds the Updates tab rows from the inventory,
// preserving selection/task state across refreshes by container ID.
// (Recreated containers get new IDs and therefore fresh rows.)
func (m *Model) rebuildUpdateRows() {
	prev := map[string]*updateRow{}
	for _, r := range m.updRows {
		prev[r.container.ID] = r
	}

	var updatable, other []*updateRow
	for _, c := range m.containers {
		row := prev[c.ID]
		if row == nil {
			row = &updateRow{container: c, checking: true}
		}
		row.container = c
		switch {
		case c.Managed():
			row.note = "managed by orchestrator"
			row.checking = false
			other = append(other, row)
		case c.State != "running":
			row.note = "not running"
			row.checking = false
			other = append(other, row)
		default:
			updatable = append(updatable, row)
		}
	}
	// Rows with an available update first, then the rest of updatable,
	// then the not-updatable section.
	var available, rest []*updateRow
	for _, r := range updatable {
		if r.result.Kind == registry.KindUpdateAvailable && r.task == nil {
			available = append(available, r)
		} else {
			rest = append(rest, r)
		}
	}
	m.updRows = append(append(available, rest...), other...)
	m.clampUpdCursor()
}

// sortUpdateRows re-applies the available-first ordering as check results
// arrive, keeping the cursor anchored to the same container.
func (m *Model) sortUpdateRows() {
	var cursorID string
	if m.updCursor < len(m.updRows) {
		cursorID = m.updRows[m.updCursor].container.ID
	}
	var available, rest, other []*updateRow
	for _, r := range m.updRows {
		switch {
		case r.note != "":
			other = append(other, r)
		case r.result.Kind == registry.KindUpdateAvailable && r.task == nil:
			available = append(available, r)
		default:
			rest = append(rest, r)
		}
	}
	m.updRows = append(append(available, rest...), other...)
	for i, r := range m.updRows {
		if r.container.ID == cursorID {
			m.updCursor = i
			break
		}
	}
	m.clampUpdCursor()
}

// updatesAvailable counts rows with a pending available update.
func (m Model) updatesAvailable() int {
	n := 0
	for _, r := range m.updRows {
		if r.result.Kind == registry.KindUpdateAvailable && (r.task == nil || r.task.Phase != updater.PhaseDone) {
			n++
		}
	}
	return n
}

// applyUpdateEvent folds a worker event into row state.
func (m *Model) applyUpdateEvent(ev updater.Event) {
	for _, r := range m.updRows {
		if r.task == nil || r.task.ID != ev.TaskID {
			continue
		}
		wasActive := r.task.Phase != updater.PhaseDone && r.task.Phase != updater.PhaseFailed
		r.task.Apply(ev)
		isActive := r.task.Phase != updater.PhaseDone && r.task.Phase != updater.PhaseFailed
		if wasActive && !isActive {
			m.inFlight--
		}
	}
}

func (m *Model) clampUpdCursor() {
	if m.updCursor >= len(m.updRows) {
		m.updCursor = max(0, len(m.updRows)-1)
	}
	if m.updCursor < 0 {
		m.updCursor = 0
	}
}

// Connected reports whether the engine is connected.
func (m Model) Connected() bool { return m.eng != nil && m.connErr == nil }

// layout constants
const (
	minWidth  = 80
	minHeight = 24
)

// tooSmall reports whether the terminal is below the minimum size.
func (m Model) tooSmall() bool {
	return m.width > 0 && (m.width < minWidth || m.height < minHeight)
}
