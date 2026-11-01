package tui

import (
	"bytes"
	"log"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// captureLog points the standard logger at a buffer for the rest of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// TestTerminalReports_AreNotUnhandled covers the reports Bubble Tea sends on
// its own at startup. ColorProfileMsg and EnvMsg arrive on every run; the
// rest arrive whenever the terminal answers a startup probe. Each used to
// reach handleDataMsg and log an unhandled-message warning.
func TestTerminalReports_AreNotUnhandled(t *testing.T) {
	buf := captureLog(t)
	m := newTestModel(t, ViewPolicies)

	for _, msg := range []tea.Msg{
		tea.ColorProfileMsg{},
		tea.EnvMsg{},
		tea.CapabilityMsg{Content: "RGB"},
		tea.ModeReportMsg{},
		tea.KeyboardEnhancementsMsg{},
		tea.TerminalVersionMsg{Name: "xterm"},
	} {
		next, cmd := m.Update(msg)
		if cmd != nil {
			t.Errorf("%T returned a command; a report should be a no-op", msg)
		}
		m = next.(Model)
	}

	if strings.Contains(buf.String(), "unhandled") {
		t.Errorf("terminal reports logged as unhandled:\n%s", buf)
	}
}

// TestTerminalReports_MouseStillWarns pins the other side of the line: mouse
// reporting is off on purpose, so a mouse message means someone turned it on
// without adding handling, and the warning is how that gets noticed.
func TestTerminalReports_MouseStillWarns(t *testing.T) {
	buf := captureLog(t)
	m := newTestModel(t, ViewPolicies)

	m.Update(tea.MouseClickMsg{})

	if !strings.Contains(buf.String(), "unhandled message type tea.MouseClickMsg") {
		t.Errorf("mouse click did not reach the unhandled warning; log:\n%q", buf)
	}
}
