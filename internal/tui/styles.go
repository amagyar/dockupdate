package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("86")  // cyan
	colorDim     = lipgloss.Color("241") // gray
	colorGood    = lipgloss.Color("42")  // green
	colorBad     = lipgloss.Color("196") // red
	colorWarn    = lipgloss.Color("214") // orange

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	styleHeaderDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("62"))

	styleBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(colorWarn).
			Padding(0, 1)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorDim).
				Padding(0, 1)

	styleFooter   = lipgloss.NewStyle().Foreground(colorDim)
	styleDim      = lipgloss.NewStyle().Foreground(colorDim)
	styleSelected = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	styleGood     = lipgloss.NewStyle().Foreground(colorGood)
	styleBad      = lipgloss.NewStyle().Foreground(colorBad)
	styleWarn     = lipgloss.NewStyle().Foreground(colorWarn)
	styleTitle    = lipgloss.NewStyle().Bold(true)
)
