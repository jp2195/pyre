package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestTerminalStatus_IdleReportsNothing guards the clear path: an idle view
// must set neither field, or the renderer never sends the reset and the tab
// keeps showing a fetch that finished long ago.
func TestTerminalStatus_IdleReportsNothing(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	v := m.View()
	if v.ProgressBar != nil {
		t.Errorf("idle view set ProgressBar = %+v, want nil", *v.ProgressBar)
	}
	if v.ProgramStatus != nil {
		t.Errorf("idle view set ProgramStatus = %+v, want nil", *v.ProgramStatus)
	}
}

func TestTerminalStatus_LoadingReportsWorking(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	m.policies = m.policies.SetLoading(true)
	v := m.View()

	if v.ProgressBar == nil || v.ProgressBar.State != tea.ProgressBarIndeterminate {
		t.Errorf("ProgressBar = %+v, want indeterminate", v.ProgressBar)
	}
	if v.ProgramStatus == nil || v.ProgramStatus.State != tea.ProgramStateWorking {
		t.Fatalf("ProgramStatus = %+v, want working", v.ProgramStatus)
	}
	if v.ProgramStatus.App != statusApp {
		t.Errorf("ProgramStatus.App = %q, want %q", v.ProgramStatus.App, statusApp)
	}
}

// TestTerminalStatus_ErrorStaysOutOfProgramStatus pins the split described
// on programStatus: Bubble Tea leaves the last program status in place after
// exit, so an error reported there would outlive a normal quit. Errors belong
// on the progress bar, which the renderer resets.
func TestTerminalStatus_ErrorStaysOutOfProgramStatus(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	m.policies = m.policies.SetLoading(true)
	m.err = errors.New("kaboom")
	v := m.View()

	if v.ProgressBar == nil || v.ProgressBar.State != tea.ProgressBarError {
		t.Errorf("ProgressBar = %+v, want error to win over loading", v.ProgressBar)
	}
	if v.ProgramStatus != nil && v.ProgramStatus.State == tea.ProgramStateError {
		t.Error("ProgramStatus reported an error; it would persist after quit")
	}

	m.policies = m.policies.SetLoading(false)
	if s := m.View().ProgramStatus; s != nil {
		t.Errorf("idle view with an error set ProgramStatus = %+v, want nil", *s)
	}
}

func TestTerminalStatus_LoginWaitsForAuth(t *testing.T) {
	m := newTestModel(t, ViewLogin)
	s := m.View().ProgramStatus
	if s == nil || s.State != tea.ProgramStateBlocked || s.Kind != tea.ProgramStatusKindAuth {
		t.Fatalf("login ProgramStatus = %+v, want blocked on auth", s)
	}

	// Once credentials are submitted pyre is no longer waiting on the user.
	m.loading = true
	m.login = m.login.SetSubmitting(true)
	if s := m.View().ProgramStatus; s == nil || s.State != tea.ProgramStateWorking {
		t.Errorf("submitting login ProgramStatus = %+v, want working", s)
	}
}

// TestTerminalStatus_TextIsGeneric keeps device detail out of the terminal's
// own UI, which pyre does not control. The error text here stands in for any
// message that names a host.
func TestTerminalStatus_TextIsGeneric(t *testing.T) {
	const host = "10.0.104.50"
	for _, view := range []ViewState{ViewPolicies, ViewLogin} {
		m := newTestModel(t, view)
		m.loading = true
		m.err = errors.New("dial tcp " + host + ":443: connect: connection refused")
		s := m.View().ProgramStatus
		if s == nil {
			t.Fatalf("view %d: ProgramStatus = nil, want working", view)
		}
		for _, field := range []string{s.ID, s.Title, s.Message} {
			if strings.Contains(field, host) || strings.Contains(field, "refused") {
				t.Errorf("view %d: ProgramStatus leaked device detail: %+v", view, *s)
			}
		}
	}
}
