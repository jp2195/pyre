package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/models"
)

func TestLoginView_EscapeReturnsToConnectionHub(t *testing.T) {
	m := newTestModel(t, ViewLogin)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	got := next.(Model).currentView
	if got != ViewConnectionHub {
		t.Fatalf("esc from login: currentView = %v, want ViewConnectionHub", got)
	}
}

// TestDevicePicker_DKeyIgnoredOnStandalone verifies that pressing 'd' on a
// non-Panorama (standalone firewall) connection does NOT open the device picker.
// The 'd' key should instead fall through to the view-level handler.
func TestDevicePicker_DKeyIgnoredOnStandalone(t *testing.T) {
	m := newTestModel(t, ViewSessions)
	// Register a non-Panorama connection directly into the session so the
	// global 'd' handler sees an active connection that is not Panorama.
	m.session.Connections["fw.example"] = &auth.Connection{
		Host:       "fw.example",
		Connected:  true,
		IsPanorama: false,
	}
	m.session.ActiveFirewall = "fw.example"

	// Prime the sessions view so its own 'd' handler has an observable effect:
	// one row available and the detail panel already expanded. In that state
	// the view-level 'd' key triggers detail fetch (sets detailLoading=true
	// and detailID). If the global handler incorrectly intercepts 'd', the
	// fall-through won't happen and detailLoading stays false — giving us a
	// positive assertion that 'd' reached the view.
	m.sessions = m.sessions.SetSessions([]models.Session{{ID: 42, Application: "ssh"}}, nil)
	m.sessions = m.sessions.SetExpanded(true)

	if m.sessions.IsDetailLoading() {
		t.Fatalf("precondition: detail should not be loading before dispatch")
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	nm := next.(Model)

	if nm.currentView == ViewDevicePicker {
		t.Fatalf("d on standalone connection should NOT open device picker (got %v)", nm.currentView)
	}
	if !nm.sessions.IsDetailLoading() {
		t.Fatalf("expected sessions view to handle 'd' (detailLoading=true) after fall-through; got detailLoading=false")
	}
	if got := nm.sessions.GetDetailID(); got != 42 {
		t.Fatalf("expected sessions detailID=42 after 'd' fall-through, got %d", got)
	}
}

// TestLoginView_EscapeClearsFormState verifies that pressing esc on the login
// form returns to the connection hub AND wipes any typed credentials from the
// login model's textinput buffers. This matches the "Credential Resolution"
// policy of keeping secret in-memory lifetime as short as possible.
func TestLoginView_EscapeClearsFormState(t *testing.T) {
	m := newTestModel(t, ViewLogin)

	// Populate host (initial focus), tab to username, type, tab to password,
	// type. Dispatch via Model.Update so the keys flow through the real
	// handleLoginKeys path (tab cases + fall-through to login.Update).
	typeString := func(mod Model, s string) Model {
		for _, r := range s {
			next, _ := mod.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
			mod = next.(Model)
		}
		return mod
	}

	m = typeString(m, "fw.example")
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(Model)
	m = typeString(m, "admin")
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(Model)
	m = typeString(m, "hunter2")

	if got := m.login.Host(); got != "fw.example" {
		t.Fatalf("precondition: host buffer = %q, want %q", got, "fw.example")
	}
	if got := m.login.Username(); got != "admin" {
		t.Fatalf("precondition: username buffer = %q, want %q", got, "admin")
	}
	if got := m.login.Password(); got != "hunter2" {
		t.Fatalf("precondition: password buffer = %q, want %q", got, "hunter2")
	}

	// Press esc — should reset login model and switch view.
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = next.(Model)

	if m.currentView != ViewConnectionHub {
		t.Fatalf("esc from login: currentView = %v, want ViewConnectionHub", m.currentView)
	}
	if got := m.login.Host(); got != "" {
		t.Errorf("host buffer not cleared after esc: %q", got)
	}
	if got := m.login.Username(); got != "" {
		t.Errorf("username buffer not cleared after esc: %q", got)
	}
	if got := m.login.Password(); got != "" {
		t.Errorf("password buffer not cleared after esc: %q", got)
	}
}

// TestCommandPaletteQuit_ReturnsQuitMsg verifies that the Quit entry's
// Action yields a tea.QuitMsg when invoked (not at registry build time).
func TestCommandPaletteQuit_ReturnsQuitMsg(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	reg := m.buildCommandRegistry()

	var quitAction func() tea.Msg
	for _, e := range reg {
		if e.ID == "quit" {
			quitAction = e.Action
			break
		}
	}
	if quitAction == nil {
		t.Fatal("no Quit entry in registry")
	}

	msg := quitAction()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("Quit action returned %T, want tea.QuitMsg", msg)
	}
}
