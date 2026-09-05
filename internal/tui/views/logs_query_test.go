package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

func typeString(m LogsModel, s string) LogsModel {
	for _, r := range s {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestLogsModel_FOpensQueryBar(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if !m.IsQueryMode() {
		t.Fatal("f did not open the query bar")
	}
}

func TestLogsModel_QueryCommitsOnEnter(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	m = typeString(m, "addr.src in 203.0.113.5")

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.IsQueryMode() {
		t.Error("enter did not leave query mode")
	}
	if m.Query() != "addr.src in 203.0.113.5" {
		t.Errorf("Query() = %q, want the typed expression", m.Query())
	}
	if cmd == nil {
		t.Fatal("committing a query did not refetch")
	}
	req, ok := cmd().(FetchLogsCmd)
	if !ok {
		t.Fatalf("expected FetchLogsCmd, got %T", cmd())
	}
	if req.Query != "addr.src in 203.0.113.5" {
		t.Errorf("request carried %q", req.Query)
	}
}

func TestLogsModel_QueryCancelsOnEsc(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	m = typeString(m, "addr.src in 203.0.113.5")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})

	if m.IsQueryMode() {
		t.Error("esc did not leave query mode")
	}
	if m.Query() != "" {
		t.Errorf("esc stored the query anyway: %q", m.Query())
	}
}

// The device rejects a bad query before any job exists. The previous rows
// must stay on screen and the device's own message must be shown.
func TestLogsModel_RejectedQueryKeepsRowsAndShowsMessage(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM", Description: "keep me"}}, LogPageMeta{}, nil)

	m = m.SetSystemLogs(nil, LogPageMeta{}, errors.New("syntax error at 14:50:57"))

	if got := m.rowCount(models.LogTypeSystem); got != 1 {
		t.Errorf("rows after a rejected query = %d, want 1 (previous rows kept)", got)
	}
	view := m.View()
	if !strings.Contains(view, "syntax error at 14:50:57") {
		t.Errorf("view does not show the device message:\n%s", view)
	}
}

// maxLineWidth is the widest rendered line in a view, measured the same way
// logs_fit_test.go measures the log tables.
func maxLineWidth(view string) int {
	width := 0
	for _, line := range splitLines(view) {
		if w := lipgloss.Width(line); w > width {
			width = w
		}
	}
	return width
}

// TestLogsModel_QueryBarFitsTheTerminalWidth checks that the device query
// bar never renders wider than the terminal, even with a long value.
//
// The query input had no width bound: bubbles' textinput only clamps or
// scrolls its value when Width() > 0, and at the unset default of 0 it
// renders the entire value. The composed line (label + input) was also
// never truncated the way statusLine() truncates itself to m.Width. A
// query typed or pasted near the 512-character limit, at the 60-column
// floor this application supports, would overflow the line -- the same
// defect class the log tables themselves were fixed for.
func TestLogsModel_QueryBarFitsTheTerminalWidth(t *testing.T) {
	long := strings.Repeat("addr.src in 203.0.113.5 and ", 20) // ~580 characters

	// At the supported floor (60 columns), the whole screen -- query bar
	// included -- must fit. No exceptions here.
	const floor = 60
	m := NewLogsModel().SetSize(floor, 30)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if !m.IsQueryMode() {
		t.Fatal("f did not open the query bar")
	}
	m.queryInput.SetValue(long)
	for _, line := range splitLines(m.View()) {
		if got := lipgloss.Width(line); got > floor {
			t.Errorf("width %d: line is %d cells, %d over\n  %q", floor, got, got-floor, line)
		}
	}

	// Below the floor (40 columns), the tab bar itself already overflows
	// regardless of the query bar -- renderTabBar has no fallback for its
	// own three-label content not fitting, and the project has never
	// claimed to support anything under 60 columns (logs_fit_test.go: "60
	// columns is the narrowest terminal the application supports"). That
	// pre-existing gap is not this task's to close. What the query bar
	// must not do is make it any worse, so this pins the pre-existing
	// width as a ceiling instead of asserting a perfection nothing else in
	// this view provides below the floor.
	const belowFloor = 40
	baseline := maxLineWidth(NewLogsModel().SetSize(belowFloor, 30).View())

	mn := NewLogsModel().SetSize(belowFloor, 30)
	mn, _ = mn.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if !mn.IsQueryMode() {
		t.Fatal("f did not open the query bar")
	}
	mn.queryInput.SetValue(long)
	if got := maxLineWidth(mn.View()); got > baseline {
		t.Errorf("width %d: query bar widened the screen from %d cells (pre-existing) to %d", belowFloor, baseline, got)
	}
}
