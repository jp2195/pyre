package theme

import "github.com/charmbracelet/lipgloss"

// ColorPalette contains all semantic colors used throughout the application.
// Each theme defines its own palette with colors appropriate for that theme.
type ColorPalette struct {
	// UI colors
	Primary lipgloss.Color // Main brand/accent color
	Accent  lipgloss.Color // Secondary accent color
	Success lipgloss.Color // Success states, allow actions, up status
	Error   lipgloss.Color // Error states, deny actions, down status
	Warning lipgloss.Color // Warning states, caution
	Info    lipgloss.Color // Informational, low severity

	// Severity colors (for threats, logs)
	Critical lipgloss.Color
	High     lipgloss.Color
	Medium   lipgloss.Color
	Low      lipgloss.Color

	// Text colors
	Text      lipgloss.Color // Normal text
	TextMuted lipgloss.Color // Muted/disabled text
	TextLight lipgloss.Color // Emphasized/light text
	TextLabel lipgloss.Color // Labels

	// Background colors
	Background    lipgloss.Color // Main background
	BackgroundAlt lipgloss.Color // Selected/highlighted background
	Border        lipgloss.Color // Border color
	Overlay       lipgloss.Color // Modal overlay color

	// Special colors
	White lipgloss.Color
	Black lipgloss.Color
}
