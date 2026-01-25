package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewTableBase(t *testing.T) {
	tb := NewTableBase("Search...")

	if tb.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", tb.Cursor)
	}
	if tb.Offset != 0 {
		t.Errorf("expected Offset=0, got %d", tb.Offset)
	}
	if tb.FilterMode {
		t.Error("expected FilterMode=false")
	}
	if tb.Expanded {
		t.Error("expected Expanded=false")
	}
	if tb.Loading {
		t.Error("expected Loading=false")
	}
	if tb.Filter.Placeholder != "Search..." {
		t.Errorf("expected placeholder 'Search...', got '%s'", tb.Filter.Placeholder)
	}
}

func TestTableBase_SetSize(t *testing.T) {
	tb := NewTableBase("")
	tb = tb.SetSize(100, 50)

	if tb.Width != 100 {
		t.Errorf("expected Width=100, got %d", tb.Width)
	}
	if tb.Height != 50 {
		t.Errorf("expected Height=50, got %d", tb.Height)
	}
}

func TestTableBase_SetLoading(t *testing.T) {
	tb := NewTableBase("")

	tb = tb.SetLoading(true)
	if !tb.Loading {
		t.Error("expected Loading=true")
	}

	tb = tb.SetLoading(false)
	if tb.Loading {
		t.Error("expected Loading=false")
	}
}

func TestTableBase_SetError(t *testing.T) {
	tb := NewTableBase("")
	tb.Loading = true

	err := errors.New("test error")
	tb = tb.SetError(err)

	if tb.Err != err {
		t.Errorf("expected error to be set")
	}
	if tb.Loading {
		t.Error("expected Loading=false after SetError")
	}
}

func TestTableBase_VisibleRows(t *testing.T) {
	tests := []struct {
		name             string
		height           int
		overhead         int
		expandedOverhead int
		expanded         bool
		want             int
	}{
		{"basic", 30, 5, 10, false, 25},
		{"expanded", 30, 5, 10, true, 15},
		{"minimum", 5, 10, 5, false, 1},
		{"zero height", 0, 5, 5, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTableBase("")
			tb.Height = tt.height
			tb.Expanded = tt.expanded

			got := tb.VisibleRows(tt.overhead, tt.expandedOverhead)
			if got != tt.want {
				t.Errorf("VisibleRows() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTableBase_EnsureVisible(t *testing.T) {
	tests := []struct {
		name        string
		cursor      int
		offset      int
		visibleRows int
		wantOffset  int
	}{
		{"cursor above view", 2, 5, 10, 2},
		{"cursor below view", 15, 0, 10, 6},
		{"cursor in view", 5, 0, 10, 0},
		{"cursor at bottom edge", 9, 0, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTableBase("")
			tb.Cursor = tt.cursor
			tb.Offset = tt.offset

			tb.EnsureVisible(tt.visibleRows)

			if tb.Offset != tt.wantOffset {
				t.Errorf("EnsureVisible() offset = %d, want %d", tb.Offset, tt.wantOffset)
			}
		})
	}
}

func TestTableBase_EnsureCursorValid(t *testing.T) {
	tests := []struct {
		name       string
		cursor     int
		itemCount  int
		wantCursor int
	}{
		{"cursor too high", 10, 5, 4},
		{"cursor negative", -1, 5, 0},
		{"cursor valid", 3, 5, 3},
		{"empty list preserves cursor", 5, 0, 5}, // Empty list doesn't reset cursor
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTableBase("")
			tb.Cursor = tt.cursor

			tb.EnsureCursorValid(tt.itemCount)

			if tb.Cursor != tt.wantCursor {
				t.Errorf("EnsureCursorValid() cursor = %d, want %d", tb.Cursor, tt.wantCursor)
			}
		})
	}
}

func TestTableBase_HandleNavigation(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		cursor      int
		itemCount   int
		visibleRows int
		wantCursor  int
		wantHandled bool
	}{
		{"j moves down", "j", 0, 10, 5, 1, true},
		{"down moves down", "down", 0, 10, 5, 1, true},
		{"j at end stays", "j", 9, 10, 5, 9, true},
		{"k moves up", "k", 5, 10, 5, 4, true},
		{"up moves up", "up", 5, 10, 5, 4, true},
		{"k at start stays", "k", 0, 10, 5, 0, true},
		{"g goes to start", "g", 5, 10, 5, 0, true},
		{"home goes to start", "home", 5, 10, 5, 0, true},
		{"G goes to end", "G", 0, 10, 5, 9, true},
		{"end goes to end", "end", 0, 10, 5, 9, true},
		{"unknown key", "x", 5, 10, 5, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTableBase("")
			tb.Cursor = tt.cursor

			var msg tea.KeyMsg
			switch tt.key {
			case "down":
				msg = tea.KeyMsg{Type: tea.KeyDown}
			case "up":
				msg = tea.KeyMsg{Type: tea.KeyUp}
			case "home":
				msg = tea.KeyMsg{Type: tea.KeyHome}
			case "end":
				msg = tea.KeyMsg{Type: tea.KeyEnd}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			result, handled, _ := tb.HandleNavigation(msg, tt.itemCount, tt.visibleRows)

			if handled != tt.wantHandled {
				t.Errorf("HandleNavigation() handled = %v, want %v", handled, tt.wantHandled)
			}
			if result.Cursor != tt.wantCursor {
				t.Errorf("HandleNavigation() cursor = %d, want %d", result.Cursor, tt.wantCursor)
			}
		})
	}
}

func TestTableBase_HandleNavigation_FilterMode(t *testing.T) {
	tb := NewTableBase("")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	result, handled, cmd := tb.HandleNavigation(msg, 10, 5)

	if !handled {
		t.Error("expected / to be handled")
	}
	if !result.FilterMode {
		t.Error("expected FilterMode=true after /")
	}
	if cmd == nil {
		t.Error("expected blink command after entering filter mode")
	}
}

func TestTableBase_HandleNavigation_ToggleExpanded(t *testing.T) {
	tb := NewTableBase("")
	tb.Expanded = false

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, handled, _ := tb.HandleNavigation(msg, 10, 5)

	if !handled {
		t.Error("expected enter to be handled")
	}
	if !result.Expanded {
		t.Error("expected Expanded=true after enter")
	}

	// Toggle back
	result, _, _ = result.HandleNavigation(msg, 10, 5)
	if result.Expanded {
		t.Error("expected Expanded=false after second enter")
	}
}

func TestTableBase_HandleClearFilter(t *testing.T) {
	tb := NewTableBase("")
	tb.Filter.SetValue("test")

	cleared := tb.HandleClearFilter()

	if !cleared {
		t.Error("expected HandleClearFilter to return true")
	}
	if tb.Filter.Value() != "" {
		t.Error("expected filter to be cleared")
	}

	// Second call should return false
	cleared = tb.HandleClearFilter()
	if cleared {
		t.Error("expected HandleClearFilter to return false when already empty")
	}
}

func TestTableBase_HandleCollapseIfExpanded(t *testing.T) {
	tb := NewTableBase("")
	tb.Expanded = true

	collapsed := tb.HandleCollapseIfExpanded()

	if !collapsed {
		t.Error("expected HandleCollapseIfExpanded to return true")
	}
	if tb.Expanded {
		t.Error("expected Expanded=false")
	}

	// Second call should return false
	collapsed = tb.HandleCollapseIfExpanded()
	if collapsed {
		t.Error("expected HandleCollapseIfExpanded to return false when not expanded")
	}
}

func TestTableBase_ResetPosition(t *testing.T) {
	tb := NewTableBase("")
	tb.Cursor = 10
	tb.Offset = 5

	tb.ResetPosition()

	if tb.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", tb.Cursor)
	}
	if tb.Offset != 0 {
		t.Errorf("expected Offset=0, got %d", tb.Offset)
	}
}

func TestTableBase_FilterValue(t *testing.T) {
	tb := NewTableBase("")
	tb.Filter.SetValue("test filter")

	if tb.FilterValue() != "test filter" {
		t.Errorf("expected 'test filter', got '%s'", tb.FilterValue())
	}
}

func TestTableBase_IsFiltered(t *testing.T) {
	tb := NewTableBase("")

	if tb.IsFiltered() {
		t.Error("expected IsFiltered=false when empty")
	}

	tb.Filter.SetValue("test")
	if !tb.IsFiltered() {
		t.Error("expected IsFiltered=true when filter has value")
	}
}
