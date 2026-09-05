package tui

import (
	"testing"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

// TestHandleRefresh_KeepsTheModelTheRefreshMutated pins an evaluation-order
// hazard. refreshCurrentView takes a pointer receiver because refreshing the
// logs view is a state change: it restarts the pagination and returns the
// cursor to the top. Calling it inside the operand list of
// `return m, tea.Batch(m.refreshCurrentView(), ...)` leaves the order in which
// m is read unspecified by the Go spec, so the returned model may be the copy
// taken before the mutation.
//
// The tab's own bookkeeping survives either way because it lives in a map that
// every copy of the model shares. The cursor does not: it is a plain field, so
// the wrong order silently keeps the operator's old row selection over a page
// that has just been refetched from the top.
func TestHandleRefresh_KeepsTheModelTheRefreshMutated(t *testing.T) {
	m := newTestModel(t, ViewLogs)
	m.logs, _ = m.logs.SetSystemLogs(
		make([]models.SystemLogEntry, 50), views.LogPageMeta{}, nil)

	// The operator has scrolled away from the top.
	m.logs.Cursor = 12
	m.logs.Offset = 5

	next, _ := m.handleRefresh()
	nm := next.(Model)

	if nm.logs.Cursor != 0 || nm.logs.Offset != 0 {
		t.Errorf("refresh returned a model from before its own reset: cursor=%d offset=%d, want 0/0",
			nm.logs.Cursor, nm.logs.Offset)
	}
}
