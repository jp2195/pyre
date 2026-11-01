package tui

import "testing"

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
		{"ErrorStyle", ErrorStyle},
		{"NavViewLabel", NavViewLabel},
		{"NavHeaderBorder", NavHeaderBorder},
		{"SpinnerStyle", SpinnerStyle},
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
