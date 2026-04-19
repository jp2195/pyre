package theme

import "image/color"

// ColorPalette contains all semantic colors used throughout the application.
// Each theme defines its own palette with colors appropriate for that theme.
//
// In lipgloss v2, colors are represented by the standard library
// image/color.Color interface. Construct concrete values via lipgloss.Color
// ("#RRGGBB" or ANSI index strings); this type intentionally uses the
// interface so themes remain profile-agnostic.
type ColorPalette struct {
	// UI colors
	Primary color.Color // Main brand/accent color
	Accent  color.Color // Secondary accent color
	Success color.Color // Success states, allow actions, up status
	Error   color.Color // Error states, deny actions, down status
	Warning color.Color // Warning states, caution
	Info    color.Color // Informational, low severity

	// Severity colors (for threats, logs)
	Critical color.Color
	High     color.Color
	Medium   color.Color
	Low      color.Color

	// Text colors
	Text      color.Color // Normal text
	TextMuted color.Color // Muted/disabled text
	TextLight color.Color // Emphasized/light text
	TextLabel color.Color // Labels

	// Background colors
	Background    color.Color // Main background
	BackgroundAlt color.Color // Selected/highlighted background
	Border        color.Color // Border color
	Overlay       color.Color // Modal overlay color

	// Special colors
	White color.Color
	Black color.Color
}
