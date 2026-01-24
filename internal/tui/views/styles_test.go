package views

import (
	"testing"

	"github.com/jp2195/pyre/internal/tui/theme"
)

func init() {
	// Initialize theme and styles before tests run
	theme.Init("dark")
	InitStyles()
}

func TestSeverityStyle(t *testing.T) {
	tests := []struct {
		severity string
		want     string // We'll check the style renders correctly
	}{
		{"critical", "critical"},
		{"high", "high"},
		{"medium", "medium"},
		{"warning", "warning"}, // Maps to medium
		{"low", "low"},
		{"informational", "informational"},
		{"info", "info"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			style := SeverityStyle(tt.severity)
			// Verify style can render without panic
			rendered := style.Render(tt.want)
			if rendered == "" && tt.want != "" {
				t.Errorf("SeverityStyle(%q) rendered empty string", tt.severity)
			}
		})
	}
}

func TestSeverityStyle_ReturnsCorrectStyles(t *testing.T) {
	// Test that specific severities return styles that render the same as expected
	testCases := []struct {
		severity     string
		expectedText string
	}{
		{"critical", SeverityCriticalStyle.Render("X")},
		{"high", SeverityHighStyle.Render("X")},
		{"medium", SeverityMediumStyle.Render("X")},
		{"warning", SeverityMediumStyle.Render("X")},
		{"low", SeverityLowStyle.Render("X")},
		{"info", SeverityInfoStyle.Render("X")},
		{"informational", SeverityInfoStyle.Render("X")},
		{"unknown", StatusMutedStyle.Render("X")},
	}

	for _, tc := range testCases {
		t.Run(tc.severity, func(t *testing.T) {
			got := SeverityStyle(tc.severity).Render("X")
			if got != tc.expectedText {
				t.Errorf("SeverityStyle(%q) rendered differently than expected", tc.severity)
			}
		})
	}
}

func TestActionStyle(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"allow", "allow"},
		{"accept", "accept"},
		{"ACTIVE", "ACTIVE"},
		{"deny", "deny"},
		{"drop", "drop"},
		{"reject", "reject"},
		{"block", "block"},
		{"reset-client", "reset-client"},
		{"reset-server", "reset-server"},
		{"reset-both", "reset-both"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			style := ActionStyle(tt.action)
			// Verify style can render without panic
			rendered := style.Render(tt.want)
			if rendered == "" && tt.want != "" {
				t.Errorf("ActionStyle(%q) rendered empty string", tt.action)
			}
		})
	}
}

func TestActionStyle_ReturnsCorrectStyles(t *testing.T) {
	allowExpected := ActionAllowStyle.Render("X")
	denyExpected := ActionDenyStyle.Render("X")
	mutedExpected := StatusMutedStyle.Render("X")

	// Allow actions
	for _, action := range []string{"allow", "accept", "ACTIVE"} {
		got := ActionStyle(action).Render("X")
		if got != allowExpected {
			t.Errorf("expected %q to render like ActionAllowStyle", action)
		}
	}

	// Deny actions
	for _, action := range []string{"deny", "drop", "reject", "block", "reset-client", "reset-server", "reset-both"} {
		got := ActionStyle(action).Render("X")
		if got != denyExpected {
			t.Errorf("expected %q to render like ActionDenyStyle", action)
		}
	}

	// Unknown actions
	got := ActionStyle("unknown").Render("X")
	if got != mutedExpected {
		t.Error("expected unknown action to render like StatusMutedStyle")
	}
}

func TestStatusStyle(t *testing.T) {
	activeExpected := StatusActiveStyle.Render("X")
	inactiveExpected := StatusInactiveStyle.Render("X")

	upGot := StatusStyle(true).Render("X")
	if upGot != activeExpected {
		t.Error("expected StatusStyle(true) to render like StatusActiveStyle")
	}

	downGot := StatusStyle(false).Render("X")
	if downGot != inactiveExpected {
		t.Error("expected StatusStyle(false) to render like StatusInactiveStyle")
	}
}

func TestStatusStyle_Rendering(t *testing.T) {
	// Test that styles render correctly
	upRendered := StatusStyle(true).Render("UP")
	if upRendered == "" {
		t.Error("StatusStyle(true) rendered empty string")
	}

	downRendered := StatusStyle(false).Render("DOWN")
	if downRendered == "" {
		t.Error("StatusStyle(false) rendered empty string")
	}
}

// Test that all theme colors are valid
func TestThemeColors(t *testing.T) {
	c := theme.Colors()

	colors := []struct {
		name  string
		color string
	}{
		{"Primary", string(c.Primary)},
		{"Accent", string(c.Accent)},
		{"Success", string(c.Success)},
		{"Error", string(c.Error)},
		{"Warning", string(c.Warning)},
		{"Info", string(c.Info)},
		{"Critical", string(c.Critical)},
		{"High", string(c.High)},
		{"Medium", string(c.Medium)},
		{"Low", string(c.Low)},
		{"Text", string(c.Text)},
		{"TextMuted", string(c.TextMuted)},
		{"TextLight", string(c.TextLight)},
		{"TextLabel", string(c.TextLabel)},
		{"Background", string(c.Background)},
		{"BackgroundAlt", string(c.BackgroundAlt)},
		{"Border", string(c.Border)},
		{"Overlay", string(c.Overlay)},
		{"White", string(c.White)},
		{"Black", string(c.Black)},
	}

	for _, tc := range colors {
		t.Run(tc.name, func(t *testing.T) {
			if tc.color == "" {
				t.Errorf("%s is empty", tc.name)
			}
			// Check it starts with # (hex color)
			if tc.color[0] != '#' {
				t.Errorf("%s doesn't start with #: %s", tc.name, tc.color)
			}
		})
	}
}

// Test that all style variables can render without panic
func TestStylesCanRender(t *testing.T) {
	styles := []struct {
		name  string
		style func() string
	}{
		{"ViewPanelStyle", func() string { return ViewPanelStyle.Render("test") }},
		{"ViewTitleStyle", func() string { return ViewTitleStyle.Render("test") }},
		{"ViewSubtitleStyle", func() string { return ViewSubtitleStyle.Render("test") }},
		{"TableHeaderStyle", func() string { return TableHeaderStyle.Render("test") }},
		{"TableRowSelectedStyle", func() string { return TableRowSelectedStyle.Render("test") }},
		{"TableRowNormalStyle", func() string { return TableRowNormalStyle.Render("test") }},
		{"TableRowDisabledStyle", func() string { return TableRowDisabledStyle.Render("test") }},
		{"DetailLabelStyle", func() string { return DetailLabelStyle.Render("test") }},
		{"DetailValueStyle", func() string { return DetailValueStyle.Render("test") }},
		{"DetailDimStyle", func() string { return DetailDimStyle.Render("test") }},
		{"DetailSectionStyle", func() string { return DetailSectionStyle.Render("test") }},
		{"StatusActiveStyle", func() string { return StatusActiveStyle.Render("test") }},
		{"StatusInactiveStyle", func() string { return StatusInactiveStyle.Render("test") }},
		{"StatusWarningStyle", func() string { return StatusWarningStyle.Render("test") }},
		{"StatusMutedStyle", func() string { return StatusMutedStyle.Render("test") }},
		{"ActionAllowStyle", func() string { return ActionAllowStyle.Render("test") }},
		{"ActionDenyStyle", func() string { return ActionDenyStyle.Render("test") }},
		{"SeverityCriticalStyle", func() string { return SeverityCriticalStyle.Render("test") }},
		{"SeverityHighStyle", func() string { return SeverityHighStyle.Render("test") }},
		{"SeverityMediumStyle", func() string { return SeverityMediumStyle.Render("test") }},
		{"SeverityLowStyle", func() string { return SeverityLowStyle.Render("test") }},
		{"SeverityInfoStyle", func() string { return SeverityInfoStyle.Render("test") }},
		{"TagStyle", func() string { return TagStyle.Render("test") }},
		{"FilterActiveStyle", func() string { return FilterActiveStyle.Render("test") }},
		{"FilterInfoStyle", func() string { return FilterInfoStyle.Render("test") }},
		{"FilterClearHintStyle", func() string { return FilterClearHintStyle.Render("test") }},
		{"HelpKeyStyle", func() string { return HelpKeyStyle.Render("test") }},
		{"HelpDescStyle", func() string { return HelpDescStyle.Render("test") }},
		{"ErrorMsgStyle", func() string { return ErrorMsgStyle.Render("test") }},
		{"WarningMsgStyle", func() string { return WarningMsgStyle.Render("test") }},
		{"SuccessMsgStyle", func() string { return SuccessMsgStyle.Render("test") }},
		{"LoadingMsgStyle", func() string { return LoadingMsgStyle.Render("test") }},
		{"EmptyMsgStyle", func() string { return EmptyMsgStyle.Render("test") }},
		{"BannerStyle", func() string { return BannerStyle.Render("test") }},
		{"BannerTitleStyle", func() string { return BannerTitleStyle.Render("test") }},
		{"BannerInfoStyle", func() string { return BannerInfoStyle.Render("test") }},
		{"LoadingBannerStyle", func() string { return LoadingBannerStyle.Render("test") }},
		{"TabActiveStyle", func() string { return TabActiveStyle.Render("test") }},
		{"TabInactiveStyle", func() string { return TabInactiveStyle.Render("test") }},
		{"DetailPanelStyle", func() string { return DetailPanelStyle.Render("test") }},
		{"FilterBorderStyle", func() string { return FilterBorderStyle.Render("test") }},
		{"CardStyle", func() string { return CardStyle.Render("test") }},
		{"CardSelectedStyle", func() string { return CardSelectedStyle.Render("test") }},
		{"TextBoldStyle", func() string { return TextBoldStyle.Render("test") }},
		{"ModalStyle", func() string { return ModalStyle.Render("test") }},
		{"ModalInputStyle", func() string { return ModalInputStyle.Render("test") }},
		{"ModalSelectedStyle", func() string { return ModalSelectedStyle.Render("test") }},
		{"ModalHelpStyle", func() string { return ModalHelpStyle.Render("test") }},
		{"InputStyle", func() string { return InputStyle.Render("test") }},
		{"InputFocusedStyle", func() string { return InputFocusedStyle.Render("test") }},
		{"NavTabActiveStyle", func() string { return NavTabActiveStyle.Render("test") }},
		{"NavKeyHintStyle", func() string { return NavKeyHintStyle.Render("test") }},
		{"PanoramaStyle", func() string { return PanoramaStyle.Render("test") }},
		{"SubtitleBoldStyle", func() string { return SubtitleBoldStyle.Render("test") }},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			// This will panic if there's an issue with the style
			result := s.style()
			if result == "" {
				t.Errorf("%s rendered empty string", s.name)
			}
		})
	}
}

// Test that all themes can be initialized
func TestThemeInit(t *testing.T) {
	themes := []string{"dark", "light", "nord", "dracula", "default", "unknown"}

	for _, themeName := range themes {
		t.Run(themeName, func(t *testing.T) {
			theme.Init(themeName)
			InitStyles()

			// Verify theme was set
			name := theme.Name()
			if name == "" {
				t.Errorf("theme name is empty after Init(%q)", themeName)
			}

			// Verify colors are accessible
			c := theme.Colors()
			if string(c.Primary) == "" {
				t.Errorf("Primary color is empty for theme %q", themeName)
			}
		})
	}

	// Reset to dark for other tests
	theme.Init("dark")
	InitStyles()
}
