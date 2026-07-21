package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/amagyar/dockupdate/internal/engine"
)

func (m *Model) clampNetCursor() {
	if m.netCursor >= len(m.networks) {
		m.netCursor = max(0, len(m.networks)-1)
	}
	if m.netDetail >= len(m.networks) {
		m.netDetail = -1
	}
	m.ensureNetVisible()
}

func (m Model) networksKey(key string) (tea.Model, tea.Cmd) {
	// Detail mode: up/down/pgup/pgdn scroll the container list, esc goes back.
	if m.netDetail >= 0 {
		page := m.contentRows() - 3
		switch key {
		case "esc":
			m.netDetail = -1
			m.netDetOffset = 0
		case "up", "k":
			m.netDetOffset--
		case "down", "j":
			m.netDetOffset++
		case "pgup":
			m.netDetOffset -= page
		case "pgdown":
			m.netDetOffset += page
		}
		m.clampNetDetailOffset()
		return m, nil
	}
	switch key {
	case "up", "k":
		if m.netCursor > 0 {
			m.netCursor--
		}
	case "down", "j":
		if m.netCursor < len(m.networks)-1 {
			m.netCursor++
		}
	case "pgup":
		m.netCursor = max(0, m.netCursor-m.contentRows())
	case "pgdown":
		m.netCursor = min(max(0, len(m.networks)-1), m.netCursor+m.contentRows())
	case "enter":
		if m.netCursor < len(m.networks) {
			m.netDetail = m.netCursor
			m.netDetOffset = 0
		}
	}
	m.ensureNetVisible()
	return m, nil
}

// networksView renders the network table or the drill-down detail.
func (m Model) networksView() string {
	if m.netDetail >= 0 && m.netDetail < len(m.networks) {
		return m.networkDetailView(m.networks[m.netDetail])
	}
	if len(m.networks) == 0 {
		return styleDim.Render("no networks found") + "\n" + styleDim.Render("press r to refresh")
	}
	var b strings.Builder
	header := fmt.Sprintf("%-20s %-10s %-18s %s", "NAME", "DRIVER", "SUBNET", "CONTAINERS")
	b.WriteString(styleDim.Render(header) + "\n")
	end := min(len(m.networks), m.netOffset+m.contentRows()-1)
	for i := m.netOffset; i < end; i++ {
		n := m.networks[i]
		subnet := n.Subnet
		if subnet == "" {
			subnet = "-"
		}
		line := fmt.Sprintf("%-20s %-10s %-18s %d", truncate(n.Name, 20), n.Driver, subnet, len(n.Containers))
		if i == m.netCursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) networkDetailView(n engine.Network) string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("network: "+n.Name) + "\n\n")
	if len(n.Containers) == 0 {
		b.WriteString(styleDim.Render("no containers connected") + "\n")
		return b.String()
	}
	header := fmt.Sprintf("%-24s %-18s %s", "CONTAINER", "IPv4", "PROJECT")
	b.WriteString(styleDim.Render(header) + "\n")
	start := min(m.netDetOffset, max(0, len(n.Containers)-1))
	end := min(len(n.Containers), start+max(1, m.contentRows()-3))
	for _, c := range n.Containers[start:end] {
		project := c.Project
		if project == "" {
			project = "standalone"
		}
		ip := c.IPv4
		if ip == "" {
			ip = "-"
		}
		fmt.Fprintf(&b, "%-24s %-18s %s\n", truncate(c.Name, 24), ip, project)
	}
	return b.String()
}

func truncate(s string, w int) string {
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "…"
}
