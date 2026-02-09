package tui

import (
	"github.com/charmbracelet/lipgloss"

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

	// Panels/Boxes
	PanelStyle      lipgloss.Style
	PanelTitleStyle lipgloss.Style

	// Input fields
	InputLabelStyle lipgloss.Style

	// Messages
	ErrorStyle   lipgloss.Style
	WarningStyle lipgloss.Style
	SuccessStyle lipgloss.Style

	// Specific elements
	DisabledRuleStyle lipgloss.Style
	ZeroHitStyle      lipgloss.Style

	// View tabs
	TabStyle       lipgloss.Style
	ActiveTabStyle lipgloss.Style

	// Navigation tabs
	NavTabInactive  lipgloss.Style
	NavTabActive    lipgloss.Style
	NavViewLabel    lipgloss.Style
	NavHeaderBorder lipgloss.Style

	// Progress bars
	ProgressBarStyle   lipgloss.Style
	ProgressBarBgStyle lipgloss.Style

	// Spinner
	SpinnerStyle lipgloss.Style

	// Command Palette
	PaletteOverlayStyle  lipgloss.Style
	PaletteModalStyle    lipgloss.Style
	PaletteInputStyle    lipgloss.Style
	PaletteCategoryStyle lipgloss.Style
	PaletteItemStyle     lipgloss.Style
	PaletteSelectedStyle lipgloss.Style
	PaletteShortcutStyle lipgloss.Style
	PaletteDescStyle     lipgloss.Style
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

	// Panels/Boxes
	PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Border).
		Padding(1, 2)

	PanelTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.Primary).
		MarginBottom(1)

	// Input fields
	InputLabelStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		MarginBottom(1)

	// Messages
	ErrorStyle = lipgloss.NewStyle().
		Foreground(c.Error).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(c.Warning)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(c.Success)

	// Specific elements
	DisabledRuleStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Italic(true)

	ZeroHitStyle = lipgloss.NewStyle().
		Foreground(c.Warning)

	// View tabs
	TabStyle = lipgloss.NewStyle().
		Padding(0, 2)

	ActiveTabStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true).
		Foreground(c.Primary).
		Underline(true)

	// Navigation tabs
	NavTabInactive = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Padding(0, 1)

	NavTabActive = lipgloss.NewStyle().
		Foreground(c.Primary).
		Bold(true).
		Padding(0, 1)

	NavViewLabel = lipgloss.NewStyle().
		Foreground(c.White).
		Bold(true)

	NavHeaderBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(c.Border)

	// Progress bars
	ProgressBarStyle = lipgloss.NewStyle().
		Foreground(c.Primary)

	ProgressBarBgStyle = lipgloss.NewStyle().
		Foreground(c.Border)

	// Spinner
	SpinnerStyle = lipgloss.NewStyle().
		Foreground(c.Primary)

	// Command Palette
	PaletteOverlayStyle = lipgloss.NewStyle().
		Background(c.Overlay)

	PaletteModalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Primary).
		Padding(0, 1).
		Width(60)

	PaletteInputStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(c.Primary).
		Padding(0, 0)

	PaletteCategoryStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Bold(true).
		MarginTop(1)

	PaletteItemStyle = lipgloss.NewStyle().
		Padding(0, 1)

	PaletteSelectedStyle = lipgloss.NewStyle().
		Background(c.Primary).
		Foreground(c.White).
		Padding(0, 1)

	PaletteShortcutStyle = lipgloss.NewStyle().
		Foreground(c.TextLabel)

	PaletteDescStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)
}

func RenderProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += ProgressBarStyle.Render("█")
		} else {
			bar += ProgressBarBgStyle.Render("░")
		}
	}
	return bar
}

func StatusStyle(up bool) lipgloss.Style {
	if up {
		return StatusUpStyle
	}
	return StatusDownStyle
}

func RenderStatus(up bool) string {
	if up {
		return StatusUpStyle.Render("●")
	}
	return StatusDownStyle.Render("●")
}
