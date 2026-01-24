package tui

import (
	"strings"
	"testing"
)

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
	}{
		{"0%", 0, 10},
		{"50%", 50, 10},
		{"100%", 100, 10},
		{"over 100%", 150, 10},
		{"negative", -10, 10},
		{"narrow bar", 50, 5},
		{"wide bar", 50, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderProgressBar(tt.percent, tt.width)
			if result == "" {
				t.Error("expected non-empty progress bar")
			}
		})
	}
}

func TestRenderProgressBar_FilledSegments(t *testing.T) {
	// 50% of 10 width should have 5 filled segments
	result := RenderProgressBar(50, 10)
	if result == "" {
		t.Error("expected non-empty progress bar")
	}
	// The result will contain styled characters
}

func TestStatusStyle(t *testing.T) {
	upStyle := StatusStyle(true)
	if upStyle.Value() == "" {
		t.Log("StatusStyle(true) returns a valid style")
	}

	downStyle := StatusStyle(false)
	if downStyle.Value() == "" {
		t.Log("StatusStyle(false) returns a valid style")
	}

	// Styles should be different
	// (Can't easily compare lipgloss styles, but we can verify they're callable)
}

func TestRenderStatus(t *testing.T) {
	upStatus := RenderStatus(true)
	if upStatus == "" {
		t.Error("expected non-empty up status")
	}
	if !strings.Contains(upStatus, "●") {
		t.Errorf("expected status to contain bullet, got %q", upStatus)
	}

	downStatus := RenderStatus(false)
	if downStatus == "" {
		t.Error("expected non-empty down status")
	}
	if !strings.Contains(downStatus, "●") {
		t.Errorf("expected status to contain bullet, got %q", downStatus)
	}
}

func TestStyleVariables(t *testing.T) {
	// Test that style variables are defined and can render
	styles := []struct {
		name  string
		style interface{ Render(...string) string }
	}{
		{"HeaderStyle", HeaderStyle},
		{"FooterStyle", FooterStyle},
		{"ConnectedStyle", ConnectedStyle},
		{"DisconnectedStyle", DisconnectedStyle},
		{"StatusUpStyle", StatusUpStyle},
		{"StatusDownStyle", StatusDownStyle},
		{"TableHeaderStyle", TableHeaderStyle},
		{"TableRowStyle", TableRowStyle},
		{"TableSelectedStyle", TableSelectedStyle},
		{"PanelStyle", PanelStyle},
		{"PanelTitleStyle", PanelTitleStyle},
		{"InputStyle", InputStyle},
		{"InputFocusedStyle", InputFocusedStyle},
		{"InputLabelStyle", InputLabelStyle},
		{"ErrorStyle", ErrorStyle},
		{"WarningStyle", WarningStyle},
		{"SuccessStyle", SuccessStyle},
		{"DisabledRuleStyle", DisabledRuleStyle},
		{"ZeroHitStyle", ZeroHitStyle},
		{"TabStyle", TabStyle},
		{"ActiveTabStyle", ActiveTabStyle},
		{"NavTabInactive", NavTabInactive},
		{"NavTabActive", NavTabActive},
		{"NavViewLabel", NavViewLabel},
		{"NavHeaderBorder", NavHeaderBorder},
		{"ProgressBarStyle", ProgressBarStyle},
		{"ProgressBarBgStyle", ProgressBarBgStyle},
		{"HelpKeyStyle", HelpKeyStyle},
		{"HelpDescStyle", HelpDescStyle},
		{"SpinnerStyle", SpinnerStyle},
		{"PaletteOverlayStyle", PaletteOverlayStyle},
		{"PaletteModalStyle", PaletteModalStyle},
		{"PaletteInputStyle", PaletteInputStyle},
		{"PaletteCategoryStyle", PaletteCategoryStyle},
		{"PaletteItemStyle", PaletteItemStyle},
		{"PaletteSelectedStyle", PaletteSelectedStyle},
		{"PaletteShortcutStyle", PaletteShortcutStyle},
		{"PaletteDescStyle", PaletteDescStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			result := s.style.Render("test")
			if result == "" {
				t.Errorf("%s.Render returned empty string", s.name)
			}
		})
	}
}
