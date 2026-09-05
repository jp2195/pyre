package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TestFooter_FitsNarrowTerminals covers a defect that affected every screen,
// not just the footer. The key hints were concatenated at a fixed length of
// about 95 cells, and the content is joined vertically, so on any terminal
// narrower than that the footer padded every other line to its own width and
// pushed the whole interface past the right edge.
func TestFooter_FitsNarrowTerminals(t *testing.T) {
	for _, width := range []int{70, 80, 90, 100, 120, 200} {
		m := newTestModel(t, ViewDashboard)
		u, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
		m = u.(Model)

		for i, line := range strings.Split(m.renderContent(), "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d cells wide", width, i+1, w)
				break
			}
		}
	}
}

// TestFooter_KeepsTheMostImportantHints checks the trimming drops hints from
// the end rather than rendering nothing useful.
func TestFooter_KeepsTheMostImportantHints(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	u, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	m = u.(Model)

	footer := m.renderFooter()
	if !strings.Contains(footer, "section") {
		t.Errorf("narrow footer dropped the navigation hint: %q", footer)
	}
}
