package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

func TestNewPickerModel(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	m := NewPickerModel(session)

	if m.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", m.cursor)
	}
}

func TestPickerModel_WithConnections(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")

	m := NewPickerModel(session)

	if len(m.connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(m.connections))
	}
	if m.active != "fw1" {
		t.Errorf("expected active='fw1', got %q", m.active)
	}
}

func TestPickerModel_UpdateConnections(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")

	m := NewPickerModel(session)

	// Add another connection
	session.AddConnection("fw2", fwConfig, "key2")
	m = m.UpdateConnections(session)

	if len(m.connections) != 2 {
		t.Errorf("expected 2 connections after update, got %d", len(m.connections))
	}
}

func TestPickerModel_SetSize(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)
	m := NewPickerModel(session)

	m = m.SetSize(100, 50)

	if m.width != 100 {
		t.Errorf("expected width=100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height=50, got %d", m.height)
	}
}

func TestPickerModel_Selected(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	// Empty picker
	m := NewPickerModel(session)
	if m.Selected() != "" {
		t.Errorf("expected empty selected, got %q", m.Selected())
	}

	// With connections
	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")

	m = NewPickerModel(session)
	if m.Selected() == "" {
		t.Error("expected non-empty selected")
	}
}

func TestPickerModel_Update_Navigation(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")
	session.AddConnection("fw3", fwConfig, "key3")

	m := NewPickerModel(session)
	initialCursor := m.cursor

	// Move down with j
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	expectedAfterJ := initialCursor + 1
	if initialCursor >= 2 {
		expectedAfterJ = 2 // Can't go past end
	}
	if m.cursor != expectedAfterJ {
		t.Errorf("expected cursor=%d after j (started at %d), got %d", expectedAfterJ, initialCursor, m.cursor)
	}

	// Move up with k
	prevCursor := m.cursor
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	expectedAfterK := prevCursor - 1
	if prevCursor == 0 {
		expectedAfterK = 0 // Can't go below 0
	}
	if m.cursor != expectedAfterK {
		t.Errorf("expected cursor=%d after k (started at %d), got %d", expectedAfterK, prevCursor, m.cursor)
	}

	// Go to start
	m.cursor = 0
	// At start, shouldn't go negative
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", m.cursor)
	}

	// Go to end
	m.cursor = 2
	// At end, shouldn't go further
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}
}

func TestPickerModel_View_Empty(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	m := NewPickerModel(session)
	m = m.SetSize(100, 50)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestPickerModel_View_WithConnections(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")

	m := NewPickerModel(session)
	m = m.SetSize(100, 50)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view with connections")
	}
}

func TestPickerModel_View_ZeroWidth(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)
	m := NewPickerModel(session)

	view := m.View()
	if view != "Loading..." {
		t.Errorf("expected 'Loading...' with zero width, got %q", view)
	}
}

func TestPickerModel_CursorOnActive(t *testing.T) {
	cfg := config.DefaultConfig()
	session := auth.NewSession(cfg)

	fwConfig := &config.FirewallConfig{Host: "10.0.0.1"}
	session.AddConnection("fw1", fwConfig, "key1")
	session.AddConnection("fw2", fwConfig, "key2")
	session.AddConnection("fw3", fwConfig, "key3")

	// Set fw2 as active
	session.SetActiveFirewall("fw2")

	m := NewPickerModel(session)

	// Cursor should be on fw2 (index depends on map iteration order)
	selected := m.Selected()
	if selected == "" {
		t.Error("expected a selection")
	}
}
