package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// typeInto sends each rune of s to the login handler as a key press, the
// same way a user filling the form would.
func typeInto(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(Model)
	}
	return m
}

// fillLogin returns a login-view Model with every field populated so
// CanSubmit reports true.
func fillLogin(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t, ViewLogin)

	m = typeInto(t, m, "fw.example.com")
	updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	m = typeInto(t, m, "admin")
	updated, _ = m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	m = typeInto(t, m, "hunter2")

	if !m.login.CanSubmit() {
		t.Fatalf("precondition: form should be submittable (host=%q user=%q)",
			m.login.Host(), m.login.Username())
	}
	return m
}

// TestLoginEnterIsIgnoredWhileInFlight guards the MFA lockout bug: while a
// keygen is already in flight, further Enter presses must NOT fire another
// auth attempt. PAN-OS counts each keygen as a failed login while the MFA
// push is pending, so re-submitting locks the account out.
func TestLoginEnterIsIgnoredWhileInFlight(t *testing.T) {
	m := fillLogin(t)

	updated, cmd := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("first Enter should start a login")
	}
	m = updated.(Model)
	if !m.loading {
		t.Fatal("first Enter should set loading")
	}

	for i := range 4 {
		updated, cmd = m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			t.Fatalf("Enter #%d while in flight fired another auth attempt; "+
				"this is what locks the account out during MFA", i+2)
		}
	}
}

// TestLoginViewShowsInFlightFeedback ensures the user gets visible feedback
// that authentication is underway. Without it they assume Enter did nothing
// and keep pressing it.
func TestLoginViewShowsInFlightFeedback(t *testing.T) {
	m := fillLogin(t)

	before := m.login.View()
	if strings.Contains(strings.ToLower(before), "authenticating") {
		t.Fatal("idle form should not claim to be authenticating")
	}

	updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	got := m.login.View()
	if !strings.Contains(strings.ToLower(got), "authenticating") {
		t.Errorf("in-flight login view gives the user no feedback; got:\n%s", got)
	}
}

// TestLoginEscCancelsInFlight lets the user back out of a stalled login
// rather than being stuck on a frozen form.
func TestLoginEscCancelsInFlight(t *testing.T) {
	m := fillLogin(t)

	updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	updated, _ = m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)

	if m.loading {
		t.Error("esc should clear the in-flight loading state")
	}
	if m.currentView != ViewConnectionHub {
		t.Errorf("esc should leave the login view, got %v", m.currentView)
	}
}

// TestLoginErrorClearsInFlight ensures a failed login re-arms the form so the
// user can correct credentials and retry.
func TestLoginErrorClearsInFlight(t *testing.T) {
	m := fillLogin(t)

	updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	res, _ := m.Update(LoginErrorMsg{Err: errFake{}})
	m = res.(Model)

	if m.login.Submitting() {
		t.Error("login error should clear the submitting flag")
	}

	// The form must accept Enter again after the failure.
	_, cmd := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("after a login error the form should accept a retry")
	}
}

type errFake struct{}

func (errFake) Error() string { return "invalid credentials" }

// TestPasteIntoLoginFields guards bracketed-paste support. tea.PasteMsg was
// never routed anywhere — the top-level Update only matched KeyPressMsg, so
// paste fell through to handleDataMsg and was logged as an unhandled message
// and dropped. Users could not paste a password (or host) into the login form.
func TestPasteIntoLoginFields(t *testing.T) {
	m := newTestModel(t, ViewLogin)

	// Focus starts on the host field.
	res, _ := m.Update(tea.PasteMsg{Content: "fw.example.com"})
	m = res.(Model)
	if got := m.login.Host(); got != "fw.example.com" {
		t.Errorf("paste into host field: got %q, want %q", got, "fw.example.com")
	}

	updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	res, _ = m.Update(tea.PasteMsg{Content: "admin"})
	m = res.(Model)
	if got := m.login.Username(); got != "admin" {
		t.Errorf("paste into username field: got %q, want %q", got, "admin")
	}

	updated, _ = m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)
	res, _ = m.Update(tea.PasteMsg{Content: "s3cr3t-p@ss"})
	m = res.(Model)
	if got := m.login.Password(); got != "s3cr3t-p@ss" {
		t.Errorf("paste into password field: got %q, want %q", got, "s3cr3t-p@ss")
	}
}

// TestPasteIgnoredWhileAuthenticating keeps paste consistent with the other
// input handling: the fields are frozen while a keygen is in flight.
func TestPasteIgnoredWhileAuthenticating(t *testing.T) {
	m := fillLogin(t)
	updated, _ := m.handleLoginKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)

	before := m.login.Password()
	res, _ := m.Update(tea.PasteMsg{Content: "extra"})
	m = res.(Model)
	if got := m.login.Password(); got != before {
		t.Errorf("paste mutated a frozen field: %q -> %q", before, got)
	}
}
