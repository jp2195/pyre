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
	// Enter sends the expression; it becomes the stored query only once the
	// device answers it, because a refused one is never stored.
	if got := m.Query(); got != "" {
		t.Errorf("Query() = %q before the device answered, want empty", got)
	}
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{Req: req}, nil)
	if m.Query() != "addr.src in 203.0.113.5" {
		t.Errorf("Query() = %q, want the accepted expression", m.Query())
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
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM", Description: "keep me"}}, LogPageMeta{}, nil)

	m, _ = m.SetSystemLogs(nil, LogPageMeta{}, errors.New("syntax error at 14:50:57"))

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

// Zero rows under an active query must say what was asked, so a bound that
// silently matched nothing is visible rather than reading as "no traffic".
func TestLogsModel_ZeroRowsNamesTheSentExpression(t *testing.T) {
	sent := "(receive_time geq '2026/09/05 11:57:00') and ((receive_time geq '2025/13/45 99:99:99'))"
	m := NewLogsModel().SetSize(120, 40)
	m, _ = m.SetSystemLogs(nil, LogPageMeta{Sent: sent}, nil)

	view := m.View()
	if !strings.Contains(view, "0 rows matched") {
		t.Errorf("view does not report a zero match:\n%s", view)
	}
	if !strings.Contains(view, "2025/13/45") {
		t.Errorf("view does not name the expression that was sent:\n%s", view)
	}
}

// With no query at all, the plain empty state is still the right message.
func TestLogsModel_ZeroRowsWithNoQueryKeepsPlainMessage(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m, _ = m.SetSystemLogs(nil, LogPageMeta{}, nil)

	view := m.View()
	if !strings.Contains(view, "No system logs found") {
		t.Errorf("plain empty state missing:\n%s", view)
	}
	if strings.Contains(view, "0 rows matched") {
		t.Errorf("query-specific empty state shown with no query:\n%s", view)
	}
}

// A long assembled expression must wrap inside the terminal, never widen it,
// same as every other line this view renders.
func TestLogsModel_ZeroRowsExpressionWrapsInsteadOfWidening(t *testing.T) {
	sent := strings.Repeat("(receive_time geq '2026/09/05 11:57:00') and ", 10)
	const width = 60
	m := NewLogsModel().SetSize(width, 40)
	m, _ = m.SetSystemLogs(nil, LogPageMeta{Sent: sent}, nil)

	for _, line := range splitLines(m.View()) {
		if got := lipgloss.Width(line); got > width {
			t.Errorf("width %d: line is %d cells, %d over\n  %q", width, got, got-width, line)
		}
	}
}

// Switching to a tab with no rows yet triggers a fetch. Until that fetch
// completes, the tab must show the loading state rather than asserting
// there are no logs — the same lie the zero-match empty state exists to
// fix, just on the way in instead of on the way out.
func TestLogsModel_SwitchingToUnfetchedTabShowsLoadingNotEmpty(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)

	m, cmd := m.Update(tea.KeyPressMsg{Code: ']', Text: "]"}) // System -> Traffic
	if m.activeLogType != models.LogTypeTraffic {
		t.Fatalf("expected Traffic after ], got %v", m.activeLogType)
	}
	if cmd == nil {
		t.Fatal("switching to an unfetched tab did not request a fetch")
	}
	if !m.Loading {
		t.Error("switching to an unfetched tab did not set Loading")
	}

	view := m.View()
	if strings.Contains(view, "No traffic logs found") {
		t.Errorf("view claims no traffic logs while the fetch is in flight:\n%s", view)
	}
}

// A local `/` filter that empties an otherwise-populated tab must blame the
// filter, not the device: rowCount is > 0, so the device did its job and the
// sent expression (if any) had nothing to do with the empty table.
func TestLogsModel_LocalFilterZeroingRowsBlamesTheFilterNotTheDevice(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{
		{Severity: "informational", Type: "general", Description: "admin login ok"},
	}, LogPageMeta{}, nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = typeString(m, "zzz-no-match")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	view := m.View()
	if strings.Contains(view, "0 rows matched") {
		t.Errorf("filter-emptied table read as a device zero match:\n%s", view)
	}
	if !strings.Contains(view, "loaded)") {
		t.Errorf("view does not name the filter as what emptied the table:\n%s", view)
	}
}

// Same as above, but with a device query also in play. This is the exact
// case a review caught: naming the sent expression whenever a tab renders
// zero filtered rows, without checking whether the raw fetch actually came
// back empty, misattributes a client-side filter result to the server.
func TestLogsModel_LocalFilterZeroingRowsIgnoresConcurrentDeviceQuery(t *testing.T) {
	sent := "(receive_time geq '2026/09/05 11:57:00')"
	m := NewLogsModel().SetSize(120, 40)
	m, _ = m.SetSystemLogs([]models.SystemLogEntry{
		{Severity: "informational", Type: "general", Description: "admin login ok"},
	}, LogPageMeta{Sent: sent}, nil)

	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = typeString(m, "zzz-no-match")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	view := m.View()
	if strings.Contains(view, sent) {
		t.Errorf("view blames the device expression for a client-side filter result:\n%s", view)
	}
	if strings.Contains(view, "0 rows matched") {
		t.Errorf("filter-emptied table under an active device query read as a device zero match:\n%s", view)
	}
	if !strings.Contains(view, "loaded)") {
		t.Errorf("view does not name the filter as what emptied the table:\n%s", view)
	}
}

// Both the filter-emptied-tab empty state and the "Filter: ... (N results)"
// info line format the user's own filter text into a rendered line with no
// width bound of their own — the same defect class the sent-expression
// empty state exists to fix. Filter.CharLimit is 100, so a filter typed all
// the way to that limit (not a short stand-in) is what actually reaches
// these lines.
func TestLogsModel_FilterTextFitsTheTerminalWidth(t *testing.T) {
	filterVal := strings.Repeat("west ", 20) // exactly 100 chars: the CharLimit

	newFixture := func(width, height int) LogsModel {
		m := NewLogsModel().SetSize(width, height)
		m, _ = m.SetSystemLogs([]models.SystemLogEntry{
			{Severity: "informational", Type: "general", Description: "admin login ok"},
		}, LogPageMeta{}, nil)
		return m
	}
	applyLongFilter := func(m LogsModel) LogsModel {
		m.Filter.SetValue(filterVal)
		m.applyFilter()
		return m
	}

	// At the supported floor (60 columns), every line — filter info and
	// the filter-emptied empty state included — must fit. No exceptions.
	const floor = 60
	m := applyLongFilter(newFixture(floor, 30))
	if !m.IsFiltered() {
		t.Fatal("filter value did not register as active")
	}
	if got := len(m.FilterValue()); got != 100 {
		t.Fatalf("filter value is %d chars, want the full 100-char CharLimit", got)
	}
	for _, line := range splitLines(m.View()) {
		if got := lipgloss.Width(line); got > floor {
			t.Errorf("width %d: line is %d cells, %d over\n  %q", floor, got, got-floor, line)
		}
	}

	// Below the floor (40 columns), the tab bar itself already overflows
	// regardless of the filter text — the same pre-existing gap
	// TestLogsModel_QueryBarFitsTheTerminalWidth documents and pins rather
	// than tries to close here. What the filter text must not do is make
	// that any worse.
	const belowFloor = 40
	baseline := maxLineWidth(newFixture(belowFloor, 30).View())

	mn := applyLongFilter(newFixture(belowFloor, 30))
	if got := maxLineWidth(mn.View()); got > baseline {
		t.Errorf("width %d: filter text widened the screen from %d cells (pre-existing) to %d", belowFloor, baseline, got)
	}
}
