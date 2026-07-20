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
}

func (m Model) networksKey(key string) (tea.Model, tea.Cmd) {
	// Detail mode: only esc/back navigates out.
	if m.netDetail >= 0 {
		if key == "esc" {
			m.netDetail = -1
		}
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
	case "enter":
		if m.netCursor < len(m.networks) {
			m.netDetail = m.netCursor
		}
	}
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
	for i, n := range m.networks {
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
	for _, c := range n.Containers {
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
