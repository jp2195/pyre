package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	errorColor     = lipgloss.Color("#EF4444")
	warningColor   = lipgloss.Color("#F59E0B")
	mutedColor     = lipgloss.Color("#6B7280")
	borderColor    = lipgloss.Color("#374151")
	highlightColor = lipgloss.Color("#1F2937")

	// Header/Footer
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)

	// Status indicators
	ConnectedStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	DisconnectedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	StatusUpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	StatusDownStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// Tables
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#374151")).
				Padding(0, 1)

	TableRowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	TableSelectedStyle = lipgloss.NewStyle().
				Background(highlightColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)

	// Panels/Boxes
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Input fields
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	InputFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	InputLabelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(1)

	// Messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Specific elements
	DisabledRuleStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true)

	ZeroHitStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// View tabs
	TabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(primaryColor).
			Underline(true)

	// Progress bars
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(primaryColor)

	ProgressBarBgStyle = lipgloss.NewStyle().
				Foreground(borderColor)

	// Help
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Spinner
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// Command Palette
	PaletteOverlayStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#000000"))

	PaletteModalStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1).
				Width(60)

	PaletteInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(primaryColor).
				Padding(0, 0)

	PaletteCategoryStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true).
				MarginTop(1)

	PaletteItemStyle = lipgloss.NewStyle().
				Padding(0, 1)

	PaletteSelectedStyle = lipgloss.NewStyle().
				Background(primaryColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)

	PaletteShortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF"))

	PaletteDescStyle = lipgloss.NewStyle().
				Foreground(mutedColor)
)

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
