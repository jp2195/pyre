package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestDashboardScrollKeysMatchTableNavigation pins that scrolling a dashboard
// uses the same bindings as scrolling a table. They were briefly divergent
// (Ctrl+F/Ctrl+B, and no `g`), which would have been an odd thing to have to
// document.
func TestDashboardScrollKeysMatchTableNavigation(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	m.height = 40

	cases := []struct {
		name string
		key  tea.KeyPressMsg
		want int
	}{
		{"j", tea.KeyPressMsg{Code: 'j', Text: "j"}, 1},
		{"down", tea.KeyPressMsg{Code: tea.KeyDown}, 1},
		{"k", tea.KeyPressMsg{Code: 'k', Text: "k"}, -1},
		{"up", tea.KeyPressMsg{Code: tea.KeyUp}, -1},
		{"ctrl+d", tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}, 35},
		{"ctrl+u", tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, -35},
		{"g", tea.KeyPressMsg{Code: 'g', Text: "g"}, -scrollJump},
		{"G", tea.KeyPressMsg{Code: 'G', Text: "G"}, scrollJump},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := m.dashboardScrollDelta(tc.key)
			if !ok {
				t.Fatalf("%s is not recognised as a scroll key", tc.name)
			}
			if got != tc.want {
				t.Errorf("%s delta = %d, want %d", tc.name, got, tc.want)
			}
		})
	}

	// A non-scroll key must fall through to the view.
	if _, ok := m.dashboardScrollDelta(tea.KeyPressMsg{Code: 'x', Text: "x"}); ok {
		t.Error("'x' should not be treated as a scroll key")
	}
}
