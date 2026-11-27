package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// openHelp sizes the model and presses ? to open the help overlay.
func openHelp(t *testing.T, w, h int) Model {
	t.Helper()
	m := newTestModel(t, ViewDashboard)
	u, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = u.(Model)
	u, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = u.(Model)
	if !m.showHelp {
		t.Fatal("? did not open the help overlay")
	}
	return m
}

// TestHelpOverlay_ShowsFullHelp guards against the overlay rendering the
// one-line ShortHelp: the bindings below appear only in KeyMap.FullHelp.
func TestHelpOverlay_ShowsFullHelp(t *testing.T) {
	m := openHelp(t, 120, 30)
	out := m.renderHelp()
	for _, want := range []string{"page up", "page down", "filter", "search", "back/cancel"} {
		if !strings.Contains(out, want) {
			t.Errorf("help overlay missing full-help binding %q:\n%s", want, out)
		}
	}
	if !strings.Contains(m.renderContent(), "page down") {
		t.Error("rendered screen does not show the full help overlay")
	}
	// The shared help model must stay in short mode.
	if m.help.ShowAll {
		t.Error("renderHelp leaked ShowAll into the shared help model")
	}
}

// TestHelpOverlay_FitsShortTerminal checks the overlay never makes the screen
// taller than the terminal once the header and footer are added.
func TestHelpOverlay_FitsShortTerminal(t *testing.T) {
	for _, h := range []int{5, 6, 7, 8, 10, 24} {
		m := openHelp(t, 80, h)
		if got := lipgloss.Height(m.renderContent()); got > h {
			t.Errorf("height %d: help screen renders %d lines", h, got)
		}
	}
}
