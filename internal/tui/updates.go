package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/amagyar/dockupdate/internal/compose"
	"github.com/amagyar/dockupdate/internal/registry"
	"github.com/amagyar/dockupdate/internal/updater"
)

func (m Model) updatesKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.updCursor > 0 {
			m.updCursor--
		}
	case "down", "j":
		if m.updCursor < len(m.updRows)-1 {
			m.updCursor++
		}
	case " ":
		if m.updCursor < len(m.updRows) {
			r := m.updRows[m.updCursor]
			if r.selectable() {
				r.checked = !r.checked
			}
		}
	case "a":
		// Toggle all selectable rows: select all unless all are selected.
		all := true
		count := 0
		for _, r := range m.updRows {
			if r.selectable() {
				count++
				if !r.checked {
					all = false
				}
			}
		}
		for _, r := range m.updRows {
			if r.selectable() {
				r.checked = !all && count > 0
			}
		}
	case "enter":
		return m.applySelectedUpdates()
	}
	return m, nil
}

// selectable reports whether a row can be checked for update.
func (r *updateRow) selectable() bool {
	return r.task == nil && !r.checking && r.result.Kind == registry.KindUpdateAvailable
}

// applySelectedUpdates starts the worker pool for checked rows. No-op when
// nothing is checked or updates are already running.
func (m Model) applySelectedUpdates() (tea.Model, tea.Cmd) {
	if m.updRunning || m.eng == nil {
		return m, nil
	}
	var tasks []*updater.Task
	for _, r := range m.updRows {
		if !r.checked || !r.selectable() {
			continue
		}
		t := &updater.Task{
			ID:           r.container.ID,
			Name:         r.container.Name,
			ImageRef:     r.container.Image,
			RemoteDigest: r.result.RemoteDigest,
			Prune:        m.opts.Prune,
		}
		if p := r.container.Project(); p != "" {
			t.Compose = &updater.ComposeTarget{
				Project:     p,
				Service:     r.container.Service(),
				ProjectDir:  r.container.ProjectDir(),
				ConfigFiles: compose.SplitConfigFiles(r.container.ConfigFiles()),
			}
		}
		t.Auth, _ = registry.AuthHeader(t.ImageRef)
		r.task = t
		r.checked = false
		tasks = append(tasks, t)
	}
	if len(tasks) == 0 {
		return m, nil
	}

	// Compose provider is detected once per run; standalone updates don't
	// need it, and compose tasks fail cleanly when it's missing.
	var provider updater.Compose
	if p, err := compose.Detect(); err == nil {
		provider = p
	}

	m.updEvents = make(chan updater.Event, 64)
	m.updRunning = true
	m.inFlight = len(tasks)
	runner := updater.NewRunner(m.eng, provider, m.opts.Concurrency)
	go runner.Run(context.Background(), tasks, m.updEvents)
	return m, waitUpdateEvent(m.updEvents)
}

// updatesView renders the checkbox list with one-line statuses.
func (m Model) updatesView() string {
	if len(m.containers) == 0 {
		return styleDim.Render("no containers found") + "\n" + styleDim.Render("press r to refresh")
	}
	if len(m.updRows) == 0 {
		return styleGood.Render("✔ everything is up to date")
	}
	var b strings.Builder
	otherHeader := false
	for i, r := range m.updRows {
		if r.note != "" && !otherHeader {
			otherHeader = true
			b.WriteString("\n" + styleDim.Render("not updatable:") + "\n")
		}
		line := m.renderUpdateRow(r)
		if i == m.updCursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderUpdateRow(r *updateRow) string {
	box := "[ ]"
	if r.checked {
		box = "[x]"
	}
	name := truncate(r.container.Name, 16)
	image := truncate(r.container.Image, 22)
	return fmt.Sprintf("%s %-16s %-22s %s", box, name, image, m.rowStatus(r))
}

// rowStatus renders the single-line status field per the UI spec.
func (m Model) rowStatus(r *updateRow) string {
	if r.note != "" {
		return styleDim.Render(r.note)
	}
	if r.task != nil {
		return m.taskStatus(r.task)
	}
	if r.checking {
		return m.spinner.View() + styleDim.Render(" checking…")
	}
	switch r.result.Kind {
	case registry.KindUpdateAvailable:
		return styleWarn.Render("update available")
	case registry.KindUpToDate:
		return styleDim.Render("up to date")
	case registry.KindLocalBuild:
		return styleDim.Render("local build (not updatable)")
	case registry.KindPinned:
		return styleDim.Render("pinned by digest")
	case registry.KindFailed:
		return styleBad.Render("check failed: " + errSummary(r.result.Err))
	}
	return styleDim.Render("waiting…")
}

// taskStatus renders the one-liner update pipeline state.
func (m Model) taskStatus(t *updater.Task) string {
	switch t.Phase {
	case updater.PhasePending:
		return styleDim.Render("queued…")
	case updater.PhasePulling:
		bar := progress.New(progress.WithWidth(14), progress.WithSolidFill("86"), progress.WithoutPercentage())
		pct := int(t.Percent() * 100)
		// Byte progress when the engine reports it (Docker); layer-count
		// fallback when it doesn't (Podman).
		if t.Total > 0 {
			return m.spinner.View() + " pulling  " + bar.ViewAs(t.Percent()) +
				fmt.Sprintf(" %3d%%  %s/%s", pct, humanBytes(t.Current), humanBytes(t.Total))
		}
		if t.LayersTotal > 0 {
			return m.spinner.View() + " pulling  " + bar.ViewAs(t.Percent()) +
				fmt.Sprintf(" %3d%%  %d/%d layers", pct, t.LayersDone, t.LayersTotal)
		}
		return m.spinner.View() + styleDim.Render(" pulling…")
	case updater.PhaseVerifying:
		return m.spinner.View() + styleDim.Render(" verifying checksum…")
	case updater.PhaseRestarting:
		return m.spinner.View() + styleDim.Render(" restarting service…")
	case updater.PhaseDone:
		d := t.Finished.Sub(t.Started).Round(time.Second)
		return styleGood.Render(fmt.Sprintf("✔ updated in %s", d))
	case updater.PhaseFailed:
		return styleBad.Render("✖ failed: " + errSummary(t.Err))
	}
	return ""
}

func errSummary(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()
	if i := strings.Index(msg, "\n"); i >= 0 {
		msg = msg[:i]
	}
	return truncate(msg, 80)
}

// humanBytes formats bytes with SI (1000-based) units, matching how
// Docker reports download sizes (e.g. 45MB/100MB).
func humanBytes(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}
