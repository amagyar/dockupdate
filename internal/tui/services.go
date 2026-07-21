package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/amagyar/dockupdate/internal/engine"
	"github.com/amagyar/dockupdate/internal/registry"
)

// projectGroup is a collapsible compose project (or the Standalone group).
type projectGroup struct {
	name       string
	standalone bool
	expanded   bool
	services   []serviceGroup
}

type serviceGroup struct {
	name       string
	containers []containerEntry
}

type containerEntry struct {
	c         engine.Container
	hasUpdate bool
}

type svcRowKind int

const (
	rowProject svcRowKind = iota
	rowService
	rowContainer
)

// svcRow is one visible line in the flattened tree.
type svcRow struct {
	kind    svcRowKind
	depth   int
	group   *projectGroup
	service string
	entry   *containerEntry
}

// rebuildGroups rebuilds the project → service → container tree from the
// inventory, preserving expanded state and attaching update badges.
func (m *Model) rebuildGroups() {
	prevExpanded := map[string]bool{}
	for _, g := range m.groups {
		prevExpanded[g.name] = g.expanded
	}

	updateAvailable := map[string]bool{}
	for _, r := range m.updRows {
		if r.result.Kind == registry.KindUpdateAvailable {
			updateAvailable[r.container.ID] = true
		}
	}

	byProject := map[string]map[string][]engine.Container{}
	var standalone []engine.Container
	for _, c := range m.containers {
		if p := c.Project(); p != "" {
			if byProject[p] == nil {
				byProject[p] = map[string][]engine.Container{}
			}
			svc := c.Service()
			byProject[p][svc] = append(byProject[p][svc], c)
		} else {
			standalone = append(standalone, c)
		}
	}

	var groups []*projectGroup
	projects := sortedKeys(byProject)
	for _, p := range projects {
		g := &projectGroup{name: p, expanded: true}
		if e, ok := prevExpanded[p]; ok {
			g.expanded = e
		}
		for _, svc := range sortedKeys(byProject[p]) {
			sg := serviceGroup{name: svc}
			for _, c := range byProject[p][svc] {
				sg.containers = append(sg.containers, containerEntry{c: c, hasUpdate: updateAvailable[c.ID]})
			}
			g.services = append(g.services, sg)
		}
		groups = append(groups, g)
	}
	if len(standalone) > 0 {
		g := &projectGroup{name: "Standalone", standalone: true, expanded: true}
		if e, ok := prevExpanded["Standalone"]; ok {
			g.expanded = e
		}
		sg := serviceGroup{}
		for _, c := range standalone {
			sg.containers = append(sg.containers, containerEntry{c: c, hasUpdate: updateAvailable[c.ID]})
		}
		g.services = append(g.services, sg)
		groups = append(groups, g)
	}
	m.groups = groups
	m.rebuildSvcRows()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// rebuildSvcRows flattens the tree into visible rows honoring expansion.
func (m *Model) rebuildSvcRows() {
	m.svcRows = m.svcRows[:0]
	for _, g := range m.groups {
		m.svcRows = append(m.svcRows, svcRow{kind: rowProject, depth: 0, group: g})
		if !g.expanded {
			continue
		}
		for si := range g.services {
			sg := &g.services[si]
			if !g.standalone {
				m.svcRows = append(m.svcRows, svcRow{kind: rowService, depth: 1, group: g, service: sg.name})
			}
			depth := 2
			if g.standalone {
				depth = 1
			}
			for ci := range sg.containers {
				m.svcRows = append(m.svcRows, svcRow{
					kind:    rowContainer,
					depth:   depth,
					group:   g,
					service: sg.name,
					entry:   &sg.containers[ci],
				})
			}
		}
	}
	if m.svcCursor >= len(m.svcRows) {
		m.svcCursor = max(0, len(m.svcRows)-1)
	}
	m.ensureSvcVisible()
}

func (m Model) servicesKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.svcCursor > 0 {
			m.svcCursor--
		}
	case "down", "j":
		if m.svcCursor < len(m.svcRows)-1 {
			m.svcCursor++
		}
	case "pgup":
		m.svcCursor = max(0, m.svcCursor-m.contentRows())
	case "pgdown":
		m.svcCursor = min(max(0, len(m.svcRows)-1), m.svcCursor+m.contentRows())
	case "enter":
		if m.svcCursor < len(m.svcRows) {
			row := m.svcRows[m.svcCursor]
			if row.kind == rowProject {
				row.group.expanded = !row.group.expanded
				m.rebuildSvcRows()
			}
		}
	}
	m.ensureSvcVisible()
	return m, nil
}

// servicesView renders the collapsible tree, windowed to the visible area.
func (m Model) servicesView() string {
	if len(m.containers) == 0 {
		return styleDim.Render("no containers found") + "\n" + styleDim.Render("press r to refresh")
	}
	var b strings.Builder
	end := min(len(m.svcRows), m.svcOffset+m.contentRows())
	for i := m.svcOffset; i < end; i++ {
		line := m.renderSvcRow(m.svcRows[i])
		if i == m.svcCursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderSvcRow(row svcRow) string {
	indent := strings.Repeat("  ", row.depth)
	switch row.kind {
	case rowProject:
		indicator := "▾"
		if !row.group.expanded {
			indicator = "▸"
		}
		count := 0
		for _, sg := range row.group.services {
			count += len(sg.containers)
		}
		name := row.group.name
		if !row.group.standalone {
			name = styleTitle.Render(name)
		}
		return fmt.Sprintf("%s%s %s (%d)", indent, indicator, name, count)
	case rowService:
		return fmt.Sprintf("%s%s", indent, styleDim.Render(row.service))
	case rowContainer:
		e := row.entry
		icon := "●"
		state := e.c.State
		if state != "running" {
			icon = "○"
		}
		badge := ""
		if e.hasUpdate {
			badge = " " + styleWarn.Render("⬆")
		}
		return fmt.Sprintf("%s%s %s  %s  %s%s", indent, icon, e.c.Name, styleDim.Render(e.c.Image), styleDim.Render(state), badge)
	}
	return ""
}
