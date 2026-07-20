package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders header, tab bar, content and footer.
func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}
	if m.tooSmall() {
		return fmt.Sprintf("terminal too small: %dx%d (need at least %dx%d)\n", m.width, m.height, minWidth, minHeight)
	}

	header := m.headerView()
	tabs := m.tabBarView()
	footer := m.footerView()

	var content string
	if m.connErr != nil {
		content = m.errorView()
	} else {
		switch m.active {
		case TabServices:
			content = m.servicesView()
		case TabNetworks:
			content = m.networksView()
		case TabUpdates:
			content = m.updatesView()
		}
	}

	if m.confirmQuit {
		content = m.confirmQuitView()
	}

	// Fill the content area so the footer pins to the bottom.
	contentHeight := m.height - lipgloss.Height(header) - lipgloss.Height(tabs) - lipgloss.Height(footer)
	content = lipgloss.NewStyle().Height(max(0, contentHeight)).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, footer)
}

func (m Model) headerView() string {
	title := styleHeader.Render("dockupdate")
	var parts []string
	parts = append(parts, title)

	if m.Connected() {
		enginePart := styleHeaderDim.Render(fmt.Sprintf("─ %s %s · %s ",
			m.info.Kind, m.info.Version, shortenSocket(m.info.Socket, 42)))
		parts = append(parts, enginePart)
	} else {
		parts = append(parts, styleHeaderDim.Render("─ not connected "))
	}
	if n := m.updatesAvailable(); n > 0 {
		parts = append(parts, styleBadge.Render(fmt.Sprintf("%d updates", n)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func shortenSocket(s string, w int) string {
	if len(s) <= w {
		return s
	}
	scheme, rest, found := strings.Cut(s, "://")
	if !found {
		return "…" + s[len(s)-w+1:]
	}
	keep := w - len(scheme) - 4 // scheme + "://" + "…"
	if keep < 8 {
		keep = 8
	}
	if len(rest) > keep {
		rest = "…" + rest[len(rest)-keep:]
	}
	return scheme + "://" + rest
}

func (m Model) tabBarView() string {
	tabs := []string{"1 Services", "2 Networks", "3 Updates"}
	var parts []string
	for i, t := range tabs {
		if Tab(i) == m.active {
			parts = append(parts, styleTabActive.Render(t))
		} else {
			parts = append(parts, styleTabInactive.Render(t))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m Model) footerView() string {
	if m.confirmQuit {
		return styleWarn.Render("updates in progress — really quit? y/n")
	}
	var keys []string
	switch m.active {
	case TabServices:
		keys = []string{"↑/↓ move", "enter collapse/expand", "r refresh"}
	case TabNetworks:
		if m.netDetail >= 0 {
			keys = []string{"esc back", "r refresh"}
		} else {
			keys = []string{"↑/↓ move", "enter open", "r refresh"}
		}
	case TabUpdates:
		if m.updRunning {
			keys = []string{"updates running…", "r refresh"}
		} else {
			keys = []string{"↑/↓ move", "space select", "a all", "enter apply", "r refresh"}
		}
	}
	keys = append(keys, "tab/1-3 switch", "q quit")
	return styleFooter.Render(strings.Join(keys, " · "))
}

func (m Model) errorView() string {
	var b strings.Builder
	b.WriteString(styleBad.Render("engine unreachable") + "\n\n")
	b.WriteString(errSummary(m.connErr) + "\n\n")
	if len(m.probed) > 0 {
		b.WriteString(styleDim.Render("probed:") + "\n")
		for _, p := range m.probed {
			b.WriteString(styleDim.Render("  · "+p) + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(styleDim.Render("start Docker or Podman, then press r to retry") + "\n")
	return b.String()
}

func (m Model) confirmQuitView() string {
	return styleWarn.Render("updates are still running") + "\n\n" +
		"Quit anyway? in-flight updates may leave containers stopped. (y/n)\n"
}
