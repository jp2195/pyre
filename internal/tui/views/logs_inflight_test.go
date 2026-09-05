package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
)

// logKey builds the key press for a single-rune binding in this view.
func logKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// mustFetchReq unwraps the fetch request a key handler emitted. Asserting on
// the request rather than driving a device keeps these tests at the seam the
// view actually owns.
func mustFetchReq(t *testing.T, cmd tea.Cmd) FetchLogsCmd {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a fetch command, got nil")
	}
	req, ok := cmd().(FetchLogsCmd)
	if !ok {
		t.Fatalf("expected FetchLogsCmd, got %T", cmd())
	}
	return req
}

// noFetch fails when cmd asks the device for another page.
func noFetch(t *testing.T, cmd tea.Cmd, what string) {
	t.Helper()
	if cmd == nil {
		return
	}
	if req, ok := cmd().(FetchLogsCmd); ok {
		t.Fatalf("%s: unexpected fetch %+v", what, req)
	}
}

// seedSystemPage puts a full first page on the System tab, the state every
// paging test starts from.
func seedSystemPage(t *testing.T) LogsModel {
	t.Helper()
	m := NewLogsModel().SetSize(120, 40)
	return m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{HasMore: true}, nil)
}

// A page takes one to four seconds to arrive, so m is easy to press twice
// inside one fetch. Skip comes from the rows already loaded and that only
// moves when a page lands, so both requests asked for the same rows: the same
// 500 appended twice, fetched became 1500, and the next m started at 1500 --
// silently stepping over rows 1001-1500, which no scroll ever reaches.
func TestLogs_SecondMoreIsIgnoredWhileAPageIsInFlight(t *testing.T) {
	m := seedSystemPage(t)

	m, cmd := m.Update(logKey('m'))
	first := mustFetchReq(t, cmd)
	if first.Skip != 500 || !first.Append {
		t.Fatalf("first page request = {Skip:%d Append:%v}, want {500 true}", first.Skip, first.Append)
	}

	var second tea.Cmd
	m, second = m.Update(logKey('m'))
	noFetch(t, second, "a second m while the first page was still in flight")

	// The one page that was asked for lands, and appends exactly once.
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{Req: first, HasMore: true}, nil)
	if got := m.rowCount(models.LogTypeSystem); got != 1000 {
		t.Fatalf("rows = %d, want 1000", got)
	}

	m, cmd = m.Update(logKey('m'))
	if next := mustFetchReq(t, cmd); next.Skip != 1000 {
		t.Errorf("next page Skip = %d, want 1000: paging stepped over rows nobody saw", next.Skip)
	}
}

// A page issued under one range must never land in a tab that has moved to
// another. The in-flight page would append to rows fetched under the new
// range and then stamp the mixture with it, so one table held two windows at
// once, the status line named only one of them, and the tab read as fresh
// from then on.
func TestLogs_PageFromASupersededRangeIsDropped(t *testing.T) {
	m := seedSystemPage(t)

	m, cmd := m.Update(logKey('m'))
	stalePage := mustFetchReq(t, cmd)

	// The operator changes the range before that page comes back.
	m, cmd = m.Update(logKey('t'))
	fresh := mustFetchReq(t, cmd)
	if fresh.Range == stalePage.Range {
		t.Fatal("t did not change the range")
	}
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 20), LogPageMeta{Req: fresh}, nil)

	// Only now does the page from the previous range arrive.
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{Req: stalePage, HasMore: true}, nil)

	if got := m.rowCount(models.LogTypeSystem); got != 20 {
		t.Errorf("rows = %d, want 20: a page from the previous range was applied", got)
	}
	if m.tabState(models.LogTypeSystem).hasMore {
		t.Error("the superseded page's hasMore was applied to the current one")
	}
}

// The same race across tabs: a background tab's page can be in flight when
// the range changes, and it has no newer request to be superseded by. It must
// be dropped, the tab left stale so it refetches when next shown, and its
// in-flight marker cleared so paging is not blocked on a fetch that will
// never be applied.
func TestLogs_BackgroundTabPageFromAnOldRangeIsDropped(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)

	m, cmd := m.Update(logKey(']')) // System -> Traffic, which fetches
	trafficReq := mustFetchReq(t, cmd)

	m, cmd = m.Update(logKey('[')) // back to System, which is fresh
	noFetch(t, cmd, "returning to a tab that already holds current rows")

	m, _ = m.Update(logKey('t')) // the range changes while Traffic is still fetching
	m = m.SetTrafficLogs(make([]models.TrafficLogEntry, 5), LogPageMeta{Req: trafficReq}, nil)

	if got := m.rowCount(models.LogTypeTraffic); got != 0 {
		t.Errorf("traffic rows = %d, want 0: a page from the previous range was kept", got)
	}
	if !m.tabStale(models.LogTypeTraffic) {
		t.Error("the traffic tab does not read as stale, so it will show rows it never fetched")
	}
	if m.tabState(models.LogTypeTraffic).loading {
		t.Error("the dropped page left the tab loading with nothing on its way")
	}
}

// Loading is per tab because the three tabs are three independent fetches. A
// System response arriving while Traffic is still fetching used to clear one
// view-level flag, and Traffic then rendered "No traffic logs found" -- the
// false negative the loading state exists to prevent.
func TestLogs_ResponseForOneTabDoesNotClearAnothersLoading(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)

	m, cmd := m.Update(logKey(']')) // System -> Traffic, which fetches
	mustFetchReq(t, cmd)
	if !m.Loading {
		t.Fatal("switching to an unfetched tab did not mark it loading")
	}

	// A System page issued before the switch lands.
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)

	if !m.Loading {
		t.Error("a System response cleared the Traffic tab's loading state")
	}
	if view := m.View(); strings.Contains(view, "No traffic logs found") {
		t.Errorf("Traffic claims to have no logs while its fetch is in flight:\n%s", view)
	}
}

// Every key that dispatches a fetch has to say so. Without it the round trip
// is invisible, which is what makes a second m easy to press.
func TestLogs_EveryFetchPathMarksItsTabLoading(t *testing.T) {
	paths := []struct {
		name string
		run  func(LogsModel) (LogsModel, tea.Cmd)
	}{
		{"t", func(m LogsModel) (LogsModel, tea.Cmd) { return m.Update(logKey('t')) }},
		{"T", func(m LogsModel) (LogsModel, tea.Cmd) { return m.Update(logKey('T')) }},
		{"m", func(m LogsModel) (LogsModel, tea.Cmd) { return m.Update(logKey('m')) }},
		{"f commit", func(m LogsModel) (LogsModel, tea.Cmd) {
			m, _ = m.Update(logKey('f'))
			m = typeString(m, "action eq deny")
			return m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		}},
		{"refresh", func(m LogsModel) (LogsModel, tea.Cmd) { return m.RefreshActiveTab() }},
		{"tab switch", func(m LogsModel) (LogsModel, tea.Cmd) { return m.Update(logKey(']')) }},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			m, cmd := p.run(seedSystemPage(t))
			mustFetchReq(t, cmd)
			if !m.Loading {
				t.Error("dispatched a fetch without marking the tab loading")
			}
		})
	}
}

// Every page of one pagination carries the bound that pagination started
// with. Recomputing it per page walks the window forward over a newest-first
// list that is also growing at the head, so a skip counted from the rows
// already on screen stops pointing past them and rows repeat.
func TestLogs_PagingReusesTheBoundThePaginationStartedWith(t *testing.T) {
	m := seedSystemPage(t)

	m, cmd := m.Update(logKey('t')) // 15m: a preset with a real bound
	first := mustFetchReq(t, cmd)
	if first.Since.IsZero() {
		t.Fatal("the 15m preset produced no lower bound")
	}
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{Req: first, HasMore: true}, nil)

	m, cmd = m.Update(logKey('m'))
	next := mustFetchReq(t, cmd)
	if !next.Since.Equal(first.Since) {
		t.Errorf("page 2 bound = %v, want page 1's bound %v", next.Since, first.Since)
	}
}

// A new pagination gets a fresh bound: reuse is per pagination, not forever.
func TestLogs_RestartingAPaginationTakesAFreshBound(t *testing.T) {
	m := seedSystemPage(t)

	m, cmd := m.Update(logKey('t'))
	first := mustFetchReq(t, cmd)
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{Req: first, HasMore: true}, nil)

	_, cmd = m.RefreshActiveTab()
	if again := mustFetchReq(t, cmd); !again.Since.After(first.Since) {
		t.Errorf("refresh bound = %v, want one later than %v", again.Since, first.Since)
	}
}

// The spec is explicit that a rejected query is never stored. It used to be
// written the moment enter was pressed, so a refused expression stuck: every
// other tab read stale against it, r kept retrying it, and the model reported
// it as one the device had accepted.
func TestLogs_RejectedQueryIsNotStored(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM", Description: "keep me"}}, LogPageMeta{}, nil)
	m = m.SetTrafficLogs([]models.TrafficLogEntry{{Action: "allow"}}, LogPageMeta{}, nil)

	m, _ = m.Update(logKey('f'))
	m = typeString(m, "bogus_field eq x")
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	req := mustFetchReq(t, cmd)
	if req.Query != "bogus_field eq x" {
		t.Fatalf("the fetch carried %q, want the typed expression", req.Query)
	}

	m = m.SetSystemLogs(nil, LogPageMeta{Req: req}, errors.New("Invalid operator eq for field bogus_field"))

	if got := m.Query(); got != "" {
		t.Errorf("Query() = %q, want empty: the device refused this expression", got)
	}
	if m.tabStale(models.LogTypeTraffic) {
		t.Error("the traffic tab reads stale against a query the device refused")
	}

	_, cmd = m.RefreshActiveTab()
	if again := mustFetchReq(t, cmd); again.Query != "" {
		t.Errorf("refresh re-sent the refused query %q", again.Query)
	}
}

// The operator has to be able to see the message and the expression that
// produced it together: "syntax error at 14:50:57" does not say which query,
// and the refused expression is deliberately stored nowhere else.
func TestLogs_RejectedQueryIsShownWithTheExpression(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM", Description: "keep me"}}, LogPageMeta{}, nil)

	m, _ = m.Update(logKey('f'))
	m = typeString(m, "bogus_field eq x")
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	req := mustFetchReq(t, cmd)
	m = m.SetSystemLogs(nil, LogPageMeta{Req: req}, errors.New("Invalid operator eq for field bogus_field"))

	view := m.View()
	if !strings.Contains(view, "Invalid operator eq for field bogus_field") {
		t.Errorf("view does not show the device's message:\n%s", view)
	}
	if !strings.Contains(view, "bogus_field eq x") {
		t.Errorf("view does not show the expression the message is about:\n%s", view)
	}
}

// An accepted query is stored, so this is not just "never store anything".
func TestLogs_AcceptedQueryIsStored(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m, _ = m.Update(logKey('f'))
	m = typeString(m, "action eq deny")
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	req := mustFetchReq(t, cmd)

	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{Req: req}, nil)

	if got := m.Query(); got != "action eq deny" {
		t.Errorf("Query() = %q, want the accepted expression", got)
	}
}

// The rows survived a failed fetch in the model already, but the view drew
// the error instead of the table, so the operator lost the results they were
// reading the moment a query was refused. The whole point of keeping them is
// that they stay on screen.
func TestLogs_ErrorAccompaniesTheRowsRatherThanReplacingThem(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{
		{Severity: "high", Type: "general", Description: "system-entry-marker"},
	}, LogPageMeta{}, nil)

	m = m.SetSystemLogs(nil, LogPageMeta{}, errors.New("syntax error at 14:50:57"))

	view := m.View()
	if !strings.Contains(view, "syntax error at 14:50:57") {
		t.Errorf("view does not show the device's message:\n%s", view)
	}
	if !strings.Contains(view, "system-entry-marker") {
		t.Errorf("the rows the operator was reading were replaced by the error:\n%s", view)
	}
}

// With nothing loaded there is no table worth drawing, so the error stands on
// its own rather than being paired with "No system logs found".
func TestLogs_ErrorWithNoRowsStandsAlone(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m = m.SetSystemLogs(nil, LogPageMeta{}, errors.New("connection refused"))

	view := m.View()
	if !strings.Contains(view, "connection refused") {
		t.Errorf("view does not show the error:\n%s", view)
	}
	if strings.Contains(view, "No system logs found") {
		t.Errorf("a failed fetch reads as an empty result:\n%s", view)
	}
}

// The error and the rows together must still fit the 60-column floor.
func TestLogs_ErrorWithRowsFitsTheFloor(t *testing.T) {
	const floor = 60
	m := NewLogsModel().SetSize(floor, 30)
	m = m.SetSystemLogs([]models.SystemLogEntry{
		{Severity: "high", Type: "general", Description: "an entry worth keeping on screen"},
	}, LogPageMeta{}, nil)

	m, _ = m.Update(logKey('f'))
	m = typeString(m, "receive_time geq '2025/13/45 99:99:99' and addr.src in 203.0.113.5")
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	req := mustFetchReq(t, cmd)
	m = m.SetSystemLogs(nil, LogPageMeta{Req: req},
		errors.New("Invalid value not-a-time for field receive_time at line 1 column 42"))

	for _, line := range splitLines(m.View()) {
		if got := lipgloss.Width(line); got > floor {
			t.Errorf("width %d: line is %d cells, %d over\n  %q", floor, got, got-floor, line)
		}
	}
}

// A time bound written without knowing the device's UTC offset produces a
// plausible-looking table for the wrong window, so the guess has to reach the
// screen rather than the log.
func TestLogs_ClockWarningIsShownAndFitsTheFloor(t *testing.T) {
	const warning = "device clock unknown: the time bound was written in this computer's zone, so the window may be off by the firewall's UTC offset"
	const floor = 60

	m := NewLogsModel().SetSize(floor, 30)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}},
		LogPageMeta{Warning: warning}, nil)

	view := m.View()
	if !strings.Contains(view, "device clock unknown") {
		t.Errorf("the view hides a bound written in a guessed zone:\n%s", view)
	}
	for _, line := range splitLines(view) {
		if got := lipgloss.Width(line); got > floor {
			t.Errorf("width %d: line is %d cells, %d over\n  %q", floor, got, got-floor, line)
		}
	}
}

// A clean fetch says nothing extra.
func TestLogs_NoWarningWhenTheClockWasKnown(t *testing.T) {
	m := NewLogsModel().SetSize(120, 40)
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)

	if view := m.View(); strings.Contains(view, "device clock unknown") {
		t.Errorf("a clean fetch reported a clock problem:\n%s", view)
	}
}
