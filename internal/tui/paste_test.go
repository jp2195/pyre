package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

// TestPaste_ReachesTheFocusedTextField checks that pasted text lands in the
// field under the cursor.
//
// Bubble Tea enables bracketed paste unless a view opts out, so a terminal
// delivers pasted text as a single PasteMsg rather than as a run of key
// presses. Nothing in pyre handled that message, so it fell through to the
// data dispatcher and was dropped: pasting into the login form, the
// connection form or a filter box did nothing at all. Credentials and long
// hostnames are exactly what people paste.
func TestPaste_ReachesTheFocusedTextField(t *testing.T) {
	t.Run("login host", func(t *testing.T) {
		m := newTestModel(t, ViewLogin)
		next, _ := m.Update(tea.PasteMsg{Content: "10.0.104.50"})
		if got := next.(Model).login.Host(); got != "10.0.104.50" {
			t.Errorf("host after pasting = %q, want 10.0.104.50", got)
		}
	})

	t.Run("login password", func(t *testing.T) {
		m := fillLogin(t, "10.0.104.50", "testuser", "")
		next, _ := m.Update(tea.PasteMsg{Content: "test-password"})
		if got := next.(Model).login.Password(); got != "test-password" {
			t.Errorf("password after pasting = %q, want test-password", got)
		}
	})

	t.Run("connection form host", func(t *testing.T) {
		m := newTestModel(t, ViewConnectionForm)
		m.connectionForm = views.NewAddConnectionForm()
		next, _ := m.Update(tea.PasteMsg{Content: "firewall.example.com"})
		if got := next.(Model).connectionForm.Host(); got != "firewall.example.com" {
			t.Errorf("host after pasting = %q, want firewall.example.com", got)
		}
	})

	// A filter box is a text field like any other. Compare pasting against
	// typing the same characters: if the paste is routed correctly the two
	// leave the view in the same state.
	t.Run("view filter", func(t *testing.T) {
		start := func() Model {
			m := newTestModel(t, ViewPolicies)
			m.policies = m.policies.SetPolicies([]models.SecurityRule{
				{Name: "allow-outbound-web", Position: 1, Action: "allow"},
			}, nil)
			next, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			m = next.(Model)
			if !m.currentViewFiltering() {
				t.Fatal("the policies view did not enter filter mode on /")
			}
			return m
		}

		const text = "allow-outbound"

		pasted, _ := start().Update(tea.PasteMsg{Content: text})

		typed := start()
		for _, r := range text {
			next, _ := typed.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
			typed = next.(Model)
		}

		if pasted.(Model).renderContent() != typed.renderContent() {
			t.Errorf("pasting %q left the view in a different state than typing it", text)
		}
	})

	// The device query bar (f) is a separate text field from the / filter,
	// and needs the same paste routing. A long PAN-OS expression is an
	// obvious thing to paste rather than type.
	t.Run("logs device query", func(t *testing.T) {
		m := newTestModel(t, ViewLogs)
		next, _ := m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
		m = next.(Model)
		if !m.currentViewFiltering() {
			t.Fatal("the logs view did not enter query mode on f")
		}

		const text = "addr.src in 203.0.113.5"
		next, _ = m.Update(tea.PasteMsg{Content: text})
		m = next.(Model)

		if got := m.logs.QueryValue(); got != text {
			t.Errorf("query bar after pasting = %q, want %q", got, text)
		}
	})
}

// TestPaste_IsIgnoredWhereThereIsNoTextField checks a stray paste on a
// listing screen does nothing rather than being treated as data.
func TestPaste_IsIgnoredWhereThereIsNoTextField(t *testing.T) {
	m := newTestModel(t, ViewPolicies)
	before := m.renderContent()
	next, cmd := m.Update(tea.PasteMsg{Content: "nonsense"})
	if cmd != nil {
		t.Error("a paste with nowhere to go returned a command")
	}
	if next.(Model).renderContent() != before {
		t.Error("a paste with nowhere to go changed the screen")
	}
}
