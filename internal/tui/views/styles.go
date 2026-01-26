package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/tui/theme"
)

// Style variables - initialized by InitStyles()
var (
	// Panel styles - used for card/box containers
	ViewPanelStyle    lipgloss.Style
	ViewTitleStyle    lipgloss.Style
	ViewSubtitleStyle lipgloss.Style

	// Table styles - used for list/table views
	TableHeaderStyle      lipgloss.Style
	TableRowSelectedStyle lipgloss.Style
	TableRowNormalStyle   lipgloss.Style
	TableRowDisabledStyle lipgloss.Style

	// Label/Value styles - used for detail panels
	DetailLabelStyle   lipgloss.Style
	DetailValueStyle   lipgloss.Style
	DetailDimStyle     lipgloss.Style
	DetailSectionStyle lipgloss.Style

	// Status/State styles
	StatusActiveStyle   lipgloss.Style
	StatusInactiveStyle lipgloss.Style
	StatusWarningStyle  lipgloss.Style
	StatusMutedStyle    lipgloss.Style

	// Action styles - for policy actions
	ActionAllowStyle lipgloss.Style
	ActionDenyStyle  lipgloss.Style

	// Severity styles - for logs and threats
	SeverityCriticalStyle lipgloss.Style
	SeverityHighStyle     lipgloss.Style
	SeverityMediumStyle   lipgloss.Style
	SeverityLowStyle      lipgloss.Style
	SeverityInfoStyle     lipgloss.Style

	// Tag/Badge styles
	TagStyle lipgloss.Style

	// Input/Filter styles
	FilterActiveStyle    lipgloss.Style
	FilterInfoStyle      lipgloss.Style
	FilterClearHintStyle lipgloss.Style

	// Help styles
	HelpKeyStyle  lipgloss.Style
	HelpDescStyle lipgloss.Style

	// Message styles
	ErrorMsgStyle   lipgloss.Style
	WarningMsgStyle lipgloss.Style
	SuccessMsgStyle lipgloss.Style
	LoadingMsgStyle lipgloss.Style
	EmptyMsgStyle   lipgloss.Style

	// Banner style - for view headers with title and info
	BannerStyle        lipgloss.Style
	BannerTitleStyle   lipgloss.Style
	BannerInfoStyle    lipgloss.Style
	LoadingBannerStyle lipgloss.Style

	// Tab styles - for tab bar navigation
	TabActiveStyle   lipgloss.Style
	TabInactiveStyle lipgloss.Style

	// Detail panel styles - for expanded log/session details
	DetailPanelStyle  lipgloss.Style
	FilterBorderStyle lipgloss.Style

	// Card styles - for interface/item cards
	CardStyle         lipgloss.Style
	CardSelectedStyle lipgloss.Style
	TextBoldStyle     lipgloss.Style

	// Modal styles - for command palette and dialogs
	ModalStyle         lipgloss.Style
	ModalInputStyle    lipgloss.Style
	ModalSelectedStyle lipgloss.Style
	ModalHelpStyle     lipgloss.Style

	// Input field styles - for login forms
	InputStyle        lipgloss.Style
	InputFocusedStyle lipgloss.Style

	// Navbar styles - for navigation tabs
	NavTabActiveStyle lipgloss.Style
	NavKeyHintStyle   lipgloss.Style

	// Special indicator styles
	PanoramaStyle     lipgloss.Style
	SubtitleBoldStyle lipgloss.Style

	// Loading spinner style
	LoadingSpinnerStyle lipgloss.Style
)

// InitStyles initializes all view styles using the current theme colors.
// Must be called after theme.Init().
func InitStyles() {
	c := theme.Colors()

	// Panel styles
	ViewPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Border).
		Padding(1, 2)

	ViewTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.Accent)

	ViewSubtitleStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.White).
		Background(c.Border).
		Padding(0, 1)

	TableRowSelectedStyle = lipgloss.NewStyle().
		Background(c.BackgroundAlt).
		Foreground(c.White).
		Padding(0, 1)

	TableRowNormalStyle = lipgloss.NewStyle().
		Padding(0, 1)

	TableRowDisabledStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Italic(true).
		Padding(0, 1)

	// Label/Value styles
	DetailLabelStyle = lipgloss.NewStyle().
		Foreground(c.TextLabel)

	DetailValueStyle = lipgloss.NewStyle().
		Foreground(c.TextLight)

	DetailDimStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	DetailSectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.TextLabel).
		MarginTop(1)

	// Status/State styles
	StatusActiveStyle = lipgloss.NewStyle().
		Foreground(c.Success).
		Bold(true)

	StatusInactiveStyle = lipgloss.NewStyle().
		Foreground(c.Error).
		Bold(true)

	StatusWarningStyle = lipgloss.NewStyle().
		Foreground(c.Warning)

	StatusMutedStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	// Action styles
	ActionAllowStyle = lipgloss.NewStyle().
		Foreground(c.Success).
		Bold(true)

	ActionDenyStyle = lipgloss.NewStyle().
		Foreground(c.Error).
		Bold(true)

	// Severity styles
	SeverityCriticalStyle = lipgloss.NewStyle().
		Foreground(c.Critical).
		Bold(true)

	SeverityHighStyle = lipgloss.NewStyle().
		Foreground(c.High).
		Bold(true)

	SeverityMediumStyle = lipgloss.NewStyle().
		Foreground(c.Medium)

	SeverityLowStyle = lipgloss.NewStyle().
		Foreground(c.Low)

	SeverityInfoStyle = lipgloss.NewStyle().
		Foreground(c.Success)

	// Tag/Badge styles
	TagStyle = lipgloss.NewStyle().
		Foreground(c.Accent)

	// Input/Filter styles
	FilterActiveStyle = lipgloss.NewStyle().
		Foreground(c.Primary).
		Bold(true)

	FilterInfoStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Italic(true)

	FilterClearHintStyle = lipgloss.NewStyle().
		Foreground(c.TextLabel)

	// Help styles
	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(c.Accent)

	HelpDescStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	// Message styles
	ErrorMsgStyle = lipgloss.NewStyle().
		Foreground(c.Error)

	WarningMsgStyle = lipgloss.NewStyle().
		Foreground(c.Warning)

	SuccessMsgStyle = lipgloss.NewStyle().
		Foreground(c.Success)

	LoadingMsgStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	EmptyMsgStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	// Banner style
	BannerStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(c.Border).
		Padding(0, 1).
		MarginBottom(1)

	BannerTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.White)

	BannerInfoStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted)

	LoadingBannerStyle = lipgloss.NewStyle().
		Background(c.Warning).
		Foreground(c.Black).
		Bold(true).
		Padding(0, 2)

	// Tab styles
	TabActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.White).
		Background(c.Primary).
		Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
		Foreground(c.TextLabel).
		Padding(0, 2)

	// Detail panel styles
	DetailPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(c.Primary).
		Padding(0, 2).
		MarginTop(1)

	FilterBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Primary).
		Padding(0, 1)

	// Card styles
	CardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Border).
		Padding(0, 1)

	CardSelectedStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Primary).
		Padding(0, 1)

	TextBoldStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(c.Text)

	// Modal styles
	ModalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Primary).
		Padding(0, 1)

	ModalInputStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(c.Primary).
		Padding(0, 0).
		MarginBottom(1)

	ModalSelectedStyle = lipgloss.NewStyle().
		Background(c.Primary).
		Foreground(c.White).
		Padding(0, 1)

	ModalHelpStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(c.Border).
		Padding(0, 1).
		MarginTop(1)

	// Input field styles
	InputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Border).
		Padding(0, 1).
		MarginBottom(1)

	InputFocusedStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Primary).
		Padding(0, 1).
		MarginBottom(1)

	// Navbar styles
	NavTabActiveStyle = lipgloss.NewStyle().
		Foreground(c.Primary).
		Bold(true).
		Padding(0, 1)

	NavKeyHintStyle = lipgloss.NewStyle().
		Foreground(c.TextMuted).
		Padding(0, 0)

	// Special indicator styles
	PanoramaStyle = lipgloss.NewStyle().
		Foreground(c.Accent)

	SubtitleBoldStyle = lipgloss.NewStyle().
		Foreground(c.Primary).
		Bold(true)

	LoadingSpinnerStyle = lipgloss.NewStyle().
		Foreground(c.Primary)
}

// SeverityStyle returns the appropriate style for a severity level
func SeverityStyle(severity string) lipgloss.Style {
	switch severity {
	case "critical":
		return SeverityCriticalStyle
	case "high":
		return SeverityHighStyle
	case "medium", "warning":
		return SeverityMediumStyle
	case "low":
		return SeverityLowStyle
	case "informational", "info":
		return SeverityInfoStyle
	default:
		return StatusMutedStyle
	}
}

// ActionStyle returns the appropriate style for an action type
func ActionStyle(action string) lipgloss.Style {
	switch action {
	case "allow", "accept", "ACTIVE":
		return ActionAllowStyle
	case "deny", "drop", "reject", "block", "reset-client", "reset-server", "reset-both":
		return ActionDenyStyle
	default:
		return StatusMutedStyle
	}
}

// StatusStyle returns up/down style based on boolean
func StatusStyle(up bool) lipgloss.Style {
	if up {
		return StatusActiveStyle
	}
	return StatusInactiveStyle
}
