package views

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// typeQuery opens the device query bar, types expr, and commits it, returning
// the fetch the commit issued for the tab on screen.
func typeQuery(t *testing.T, m LogsModel, expr string) (LogsModel, FetchLogsCmd) {
	t.Helper()
	m, _ = m.Update(logKey('f'))
	// The bar opens prefilled with the last committed text so a refused
	// expression can be corrected; clear it to type a fresh one.
	for range len(m.committed) {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	for _, r := range expr {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return m, mustFetchReq(t, cmd)
}

// TestLogs_ActiveTabIsNotStrandedByARevertedQuery reproduces a false empty
// state. A query the device refuses reverts the view's expression to the last
// accepted one. That revert also orphans any fetch still in flight for the
// refused expression -- including the one belonging to the tab the operator is
// now looking at. Dropping that page is right; leaving the tab idle afterwards
// is not, because nothing re-issues it. The tab rendered "No traffic logs
// found" with no error and no spinner until the operator pressed r.
func TestLogs_ActiveTabIsNotStrandedByARevertedQuery(t *testing.T) {
	InitStyles()
	m := NewLogsModel().SetSize(120, 40)

	// System is the tab on screen. Commit an expression the device will
	// refuse; this issues System's fetch for it.
	m, sysReq := typeQuery(t, m, "bad expression")

	// Move to Traffic before the refusal lands. The switch issues Traffic's
	// own fetch for the same expression.
	m, cmd := m.Update(logKey(']'))
	trafReq := mustFetchReq(t, cmd)
	if m.ActiveLogType() != models.LogTypeTraffic {
		t.Fatalf("precondition: expected Traffic on screen, got %v", m.ActiveLogType())
	}

	// The refusal arrives for System and reverts the view's query.
	m, _ = m.SetSystemLogs(nil, LogPageMeta{Req: sysReq}, errors.New("invalid query"))

	// Traffic's page now carries an expression the view has moved off, so it
	// is dropped. Traffic is the tab on screen and holds nothing.
	m, refetch := m.SetTrafficLogs(make([]models.TrafficLogEntry, 3), LogPageMeta{Req: trafReq}, nil)

	// The dropped page has to be replaced, not merely not-applied.
	req := mustFetchReq(t, refetch)
	if req.Type != models.LogTypeTraffic {
		t.Errorf("refetched %v, want the tab on screen (Traffic)", req.Type)
	}
	if req.Query != "" {
		t.Errorf("refetched under %q, want the reverted expression", req.Query)
	}
	if req.Skip != 0 || req.Append {
		t.Errorf("refetch should start a fresh pagination, got skip=%d append=%v",
			req.Skip, req.Append)
	}

	// And until it lands the tab must read as busy, not as answered-empty.
	out := m.View()
	if strings.Contains(out, "No traffic logs found") {
		t.Errorf("the tab on screen claims the device returned nothing, but its "+
			"page was dropped and a replacement is still in flight:\n%s", out)
	}
}

// TestLogs_BackgroundTabIsNotRefetchedWhenItsPageIsDropped is the other half of
// the rule. A tab nobody is looking at is refetched when next shown, so
// replacing its dropped page immediately would spend a device job and a couple
// of megabytes on rows that may never be displayed.
func TestLogs_BackgroundTabIsNotRefetchedWhenItsPageIsDropped(t *testing.T) {
	InitStyles()
	m := NewLogsModel().SetSize(120, 40)

	m, sysReq := typeQuery(t, m, "bad expression")

	m, cmd := m.Update(logKey(']'))
	trafReq := mustFetchReq(t, cmd)

	// Traffic -- the tab on screen -- is refused, which reverts the query and
	// so orphans System's in-flight fetch for the old expression.
	m, _ = m.SetTrafficLogs(nil, LogPageMeta{Req: trafReq}, errors.New("invalid query"))

	// System is the background tab now, and its page is dropped. It must be
	// left to the tab switch to refetch, not asked again here.
	m, sysCmd := m.SetSystemLogs(make([]models.SystemLogEntry, 3),
		LogPageMeta{Req: sysReq}, nil)
	noFetch(t, sysCmd, "a background tab whose page was dropped")

	if got := m.rowCount(models.LogTypeSystem); got != 0 {
		t.Errorf("the dropped page was applied anyway: System holds %d rows", got)
	}
}
