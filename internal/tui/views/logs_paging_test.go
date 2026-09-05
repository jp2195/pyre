package views

import (
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
