package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// Switching to a tab that has never been fetched must ask for it, because the
// view no longer fetches all three types up front.
func TestLogs_TabSwitchRequestsUnfetchedTab(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSystemLogs([]models.SystemLogEntry{{Type: "SYSTEM"}}, LogPageMeta{}, nil)

	m, cmd := m.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	if m.ActiveLogType() != models.LogTypeTraffic {
		t.Fatalf("active tab = %v, want traffic", m.ActiveLogType())
	}
	if cmd == nil {
		t.Fatal("switching to an unfetched tab returned no fetch command")
	}
	req, ok := cmd().(FetchLogsCmd)
	if !ok {
		t.Fatalf("expected FetchLogsCmd, got %T", cmd())
	}
	if req.Type != models.LogTypeTraffic {
		t.Errorf("request type = %v, want traffic", req.Type)
	}
	if req.Skip != 0 || req.Append {
		t.Errorf("first page should be Skip=0 Append=false, got Skip=%d Append=%v", req.Skip, req.Append)
	}
}

// A tab already holding rows for the current range must not refetch every
// time it is shown.
func TestLogs_TabSwitchDoesNotRefetchFreshTab(t *testing.T) {
	m := NewLogsModel()
	m = m.SetTrafficLogs([]models.TrafficLogEntry{{Action: "allow"}}, LogPageMeta{}, nil)
	m = m.SetActiveLogType(models.LogTypeSystem)

	// System -> Traffic, which already holds rows fetched under this range.
	m, cmd := m.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	if m.ActiveLogType() != models.LogTypeTraffic {
		t.Fatalf("active tab = %v, want traffic", m.ActiveLogType())
	}
	if cmd != nil {
		if _, isFetch := cmd().(FetchLogsCmd); isFetch {
			t.Error("a tab holding current rows was refetched on revisit")
		}
	}
}

// m asks for the next page, starting where the loaded rows end.
func TestLogs_MRequestsNextPage(t *testing.T) {
	m := NewLogsModel()
	rows := make([]models.SystemLogEntry, 500)
	m = m.SetSystemLogs(rows, LogPageMeta{HasMore: true}, nil)
	m.Cursor = 7

	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd == nil {
		t.Fatal("m returned no command while more rows were available")
	}
	req, ok := cmd().(FetchLogsCmd)
	if !ok {
		t.Fatalf("expected FetchLogsCmd, got %T", cmd())
	}
	if req.Skip != 500 {
		t.Errorf("Skip = %d, want 500", req.Skip)
	}
	if !req.Append {
		t.Error("the next page must append, not replace")
	}
	if m2.Cursor != 7 {
		t.Errorf("cursor moved to %d; paging must leave it alone", m2.Cursor)
	}
}

// A short page is the end of the results, so m must stop offering more.
func TestLogs_MDoesNothingAtEnd(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 12), LogPageMeta{HasMore: false}, nil)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd != nil {
		if _, isFetch := cmd().(FetchLogsCmd); isFetch {
			t.Error("m asked for another page after the results ended")
		}
	}
}

// The appended page extends the rows rather than replacing them.
func TestLogs_AppendedPageExtendsRows(t *testing.T) {
	m := NewLogsModel()
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{HasMore: true}, nil)
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 300), LogPageMeta{HasMore: false, Append: true}, nil)

	if got := m.rowCount(models.LogTypeSystem); got != 800 {
		t.Errorf("rows = %d, want 800", got)
	}
	if s := m.tabState(models.LogTypeSystem); s.fetched != 800 || s.hasMore {
		t.Errorf("tab state = {fetched:%d hasMore:%v}, want {800 false}", s.fetched, s.hasMore)
	}
}

func TestLogs_StatusLineOffersMoreAndStopsAtEnd(t *testing.T) {
	m := NewLogsModel().SetSize(100, 40)
	m = m.SetSystemLogs(make([]models.SystemLogEntry, 500), LogPageMeta{HasMore: true}, nil)
	if got := m.statusLine(); !strings.Contains(got, "more available") {
		t.Errorf("status line %q does not offer more", got)
	}

	m = m.SetSystemLogs(make([]models.SystemLogEntry, 10), LogPageMeta{HasMore: false, Append: true}, nil)
	if got := m.statusLine(); strings.Contains(got, "more available") {
		t.Errorf("status line %q still offers more after the end", got)
	}
}
