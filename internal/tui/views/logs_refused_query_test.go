package views

import (
	"errors"
	"strings"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

// TestLogs_ARefusedQueryIsNotResentToTheSameTab covers the per-log-type nature
// of PAN-OS field validity. An expression naming a field that only traffic logs
// carry is accepted on Traffic and refused on System. Because the view keeps a
// single accepted expression, the refusal on System does not revert anything --
// correctly, since Traffic is still answering it. But nothing recorded that
// System had already refused this exact expression, so every return to that tab
// spent another device job re-asking a question whose answer cannot change
// until the expression does.
func TestLogs_ARefusedQueryIsNotResentToTheSameTab(t *testing.T) {
	InitStyles()
	m := NewLogsModel().SetSize(120, 40)

	// Move to Traffic and let its first fetch settle.
	m, cmd := m.Update(logKey(']'))
	m, _ = m.SetTrafficLogs(make([]models.TrafficLogEntry, 2),
		LogPageMeta{Req: mustFetchReq(t, cmd)}, nil)

	// Traffic accepts the expression, so it becomes the accepted one.
	m, trafReq := typeQuery(t, m, "(addr.src in 10.0.0.1)")
	m, _ = m.SetTrafficLogs(make([]models.TrafficLogEntry, 2),
		LogPageMeta{Req: trafReq}, nil)

	// System refuses it.
	m, cmd = m.Update(logKey('['))
	if m.ActiveLogType() != models.LogTypeSystem {
		t.Fatalf("precondition: expected System, got %v", m.ActiveLogType())
	}
	m, _ = m.SetSystemLogs(nil, LogPageMeta{Req: mustFetchReq(t, cmd)},
		errors.New("invalid query field"))

	// Leave and come back. The expression has not changed, so the device has
	// nothing new to say about it.
	m, cmd = m.Update(logKey(']'))
	noFetch(t, cmd, "returning to Traffic, which already holds rows for this query")
	m, cmd = m.Update(logKey('['))
	noFetch(t, cmd, "returning to a tab that already refused this exact expression")

	// The operator still has to be told why the tab is empty.
	if out := m.View(); !strings.Contains(out, "invalid query field") {
		t.Errorf("the remembered refusal is not shown on return:\n%s", out)
	}
}

// TestLogs_ARefusedQueryIsRetriedOnceItChanges is the other half: the refusal is
// remembered against one expression, not against the tab forever.
func TestLogs_ARefusedQueryIsRetriedOnceItChanges(t *testing.T) {
	InitStyles()
	m := NewLogsModel().SetSize(120, 40)

	m, sysReq := typeQuery(t, m, "(addr.src in 10.0.0.1)")
	m, _ = m.SetSystemLogs(nil, LogPageMeta{Req: sysReq},
		errors.New("invalid query field"))

	// A new expression is a new question.
	_, retry := typeQuery(t, m, "(severity eq high)")
	if retry.Query != "(severity eq high)" {
		t.Errorf("retried with %q, want the new expression", retry.Query)
	}
}
