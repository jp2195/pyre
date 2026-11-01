package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/tui/theme"
)

// Style variables - initialized by InitStyles()
var (
	// Header/Footer
	HeaderStyle lipgloss.Style
	FooterStyle lipgloss.Style

	// Status indicators
	ConnectedStyle    lipgloss.Style
	DisconnectedStyle lipgloss.Style
	StatusUpStyle     lipgloss.Style
	StatusDownStyle   lipgloss.Style

	// Messages
	ErrorStyle lipgloss.Style

	// Navigation tabs
	NavViewLabel    lipgloss.Style
	NavHeaderBorder lipgloss.Style

	// Spinner
	SpinnerStyle lipgloss.Style
)

// InitStyles initializes all styles using the current theme colors.
// Must be called after theme.Init().
func InitStyles() {
	c := theme.Colors()

	// Header/Footer
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.White).
		Background(c.Primary).
		Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Padding(0, 1)

	// Status indicators
	ConnectedStyle = lipgloss.NewStyle().
		Foreground(c.Success).
		Bold(true)

	DisconnectedStyle = lipgloss.NewStyle().
		Foreground(c.Error).
		Bold(true)

	StatusUpStyle = lipgloss.NewStyle().
		Foreground(c.Success)

	StatusDownStyle = lipgloss.NewStyle().
		Foreground(c.Error)

	// Messages
	ErrorStyle = lipgloss.NewStyle().
		Foreground(c.Error).
		Bold(true)

	NavViewLabel = lipgloss.NewStyle().
		Foreground(c.White).
		Bold(true)

	NavHeaderBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(c.Border)

	// Spinner
	SpinnerStyle = lipgloss.NewStyle().
		Foreground(c.Primary)
}

func StatusStyle(up bool) lipgloss.Style {
	if up {
		return StatusUpStyle
	}
	return StatusDownStyle
}
