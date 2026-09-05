package tui

import (
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

// batchCmds runs cmd and returns the commands it batched, or the single
// command itself when it was not a batch.
func batchCmds(t *testing.T, cmd tea.Cmd) []tea.Cmd {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		return []tea.Cmd{cmd}
	}
	return batch
}

// A view-issued log fetch sets that tab's loading state, which renders a
// spinner. Without a tick batched alongside the fetch the banner sits on one
// frozen frame for the whole one-to-four-second round trip, so it reads as a
// hung screen rather than as work in progress.
func TestDispatch_FetchLogsCmdBatchesASpinnerTick(t *testing.T) {
	m := newTestModel(t, ViewLogs)

	_, cmd := m.handleDataMsg(views.FetchLogsCmd{Type: models.LogTypeSystem})

	var sawTick bool
	for _, c := range batchCmds(t, cmd) {
		if c == nil {
			continue
		}
		if _, ok := c().(spinner.TickMsg); ok {
			sawTick = true
		}
	}
	if !sawTick {
		t.Error("a log fetch was dispatched without a spinner tick, so its banner never animates")
	}
}

// Entering the logs view with no data must animate its spinner too.
func TestDispatch_EnteringLogsBatchesASpinnerTick(t *testing.T) {
	m := newTestModel(t, ViewLogs)

	updated, cmd := m.handleSwitchView(SwitchViewMsg{View: ViewLogs})
	if !updated.(Model).logs.Loading {
		t.Fatal("entering the logs view did not mark the tab loading")
	}

	var sawTick bool
	for _, c := range batchCmds(t, cmd) {
		if c == nil {
			continue
		}
		if _, ok := c().(spinner.TickMsg); ok {
			sawTick = true
		}
	}
	if !sawTick {
		t.Error("entering the logs view did not start the spinner")
	}
}

// The request has to survive the round trip through the parent: the view can
// only tell a page it still wants from one it has moved on from if the
// identity it issued comes back on the message.
func TestDispatch_LogMessagesCarryTheRequestBack(t *testing.T) {
	m := newTestModel(t, ViewLogs)
	req := views.FetchLogsCmd{Type: models.LogTypeSystem, Query: "action eq deny", ReqID: 7}

	updated, _ := m.handleViewDataMsg(SystemLogsMsg{
		Logs: []models.SystemLogEntry{{Type: "SYSTEM"}},
		Req:  req,
	})

	// ReqID 7 was never issued by this model, so the page is not one it is
	// waiting for and must be dropped rather than applied.
	if got := updated.(Model).logs.HasData(); got {
		t.Error("a page answering a request this view never issued was applied")
	}
}
