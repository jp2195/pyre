package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TableBase provides common state and navigation handling for table-style views.
// Views should embed this struct and delegate navigation to its methods.
type TableBase struct {
	Cursor       int
	Offset       int
	FilterMode   bool
	Filter       textinput.Model
	Expanded     bool
	Width        int
	Height       int
	SortAsc      bool
	Loading      bool
	Err          error
	SpinnerFrame string // Current spinner frame from app
}

// NewTableBase creates a new TableBase with default settings.
func NewTableBase(placeholder string) TableBase {
	f := textinput.New()
	f.Placeholder = placeholder
	f.CharLimit = 100
	f.Width = 40

	return TableBase{
		Filter: f,
	}
}

// SetSize updates the dimensions.
func (t TableBase) SetSize(width, height int) TableBase {
	t.Width = width
	t.Height = height
	return t
}

// SetLoading updates the loading state.
func (t TableBase) SetLoading(loading bool) TableBase {
	t.Loading = loading
	return t
}

// SetError updates the error state.
func (t TableBase) SetError(err error) TableBase {
	t.Err = err
	t.Loading = false
	return t
}

// SetSpinnerFrame updates the spinner frame for display.
func (t TableBase) SetSpinnerFrame(frame string) TableBase {
	t.SpinnerFrame = frame
	return t
}

// VisibleRows calculates how many rows can be displayed, accounting for overhead.
// overhead is the number of lines used by headers, footers, and other UI elements.
// expandedOverhead is additional space needed when the detail panel is expanded.
func (t TableBase) VisibleRows(overhead, expandedOverhead int) int {
	rows := t.Height - overhead
	if t.Expanded {
		rows -= expandedOverhead
	}
	if rows < 1 {
		rows = 1
	}
	return rows
}

// EnsureVisible adjusts the offset so the cursor is visible.
func (t *TableBase) EnsureVisible(visibleRows int) {
	if t.Cursor < t.Offset {
		t.Offset = t.Cursor
	}
	if t.Cursor >= t.Offset+visibleRows {
		t.Offset = t.Cursor - visibleRows + 1
	}
}

// EnsureCursorValid constrains the cursor within bounds.
func (t *TableBase) EnsureCursorValid(itemCount int) {
	if t.Cursor >= itemCount && itemCount > 0 {
		t.Cursor = itemCount - 1
	}
	if t.Cursor < 0 {
		t.Cursor = 0
	}
}

// HandleNavigation processes common navigation keys.
// Returns the updated TableBase, whether the key was handled, and any command.
// itemCount is the total number of items in the current (filtered) list.
// visibleRows is the number of visible rows for page up/down calculations.
func (t TableBase) HandleNavigation(msg tea.KeyMsg, itemCount, visibleRows int) (TableBase, bool, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if t.Cursor < itemCount-1 {
			t.Cursor++
			t.EnsureVisible(visibleRows)
		}
		return t, true, nil

	case "k", "up":
		if t.Cursor > 0 {
			t.Cursor--
			t.EnsureVisible(visibleRows)
		}
		return t, true, nil

	case "g", "home":
		t.Cursor = 0
		t.Offset = 0
		return t, true, nil

	case "G", "end":
		if itemCount > 0 {
			t.Cursor = itemCount - 1
			t.EnsureVisible(visibleRows)
		}
		return t, true, nil

	case "ctrl+d", "pgdown":
		pageSize := visibleRows
		if pageSize < 10 {
			pageSize = 10
		}
		t.Cursor += pageSize
		if t.Cursor >= itemCount {
			t.Cursor = itemCount - 1
		}
		if t.Cursor < 0 {
			t.Cursor = 0
		}
		t.EnsureVisible(visibleRows)
		return t, true, nil

	case "ctrl+u", "pgup":
		pageSize := visibleRows
		if pageSize < 10 {
			pageSize = 10
		}
		t.Cursor -= pageSize
		if t.Cursor < 0 {
			t.Cursor = 0
		}
		t.EnsureVisible(visibleRows)
		return t, true, nil

	case "/":
		t.FilterMode = true
		t.Filter.Focus()
		return t, true, textinput.Blink

	case "enter":
		t.Expanded = !t.Expanded
		return t, true, nil
	}

	return t, false, nil
}

// HandleFilterMode processes keys while in filter input mode.
// Returns the updated TableBase, whether filter mode was exited, and any command.
func (t TableBase) HandleFilterMode(msg tea.Msg) (TableBase, bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			t.FilterMode = false
			t.Filter.Blur()
			// Reset cursor on filter change
			if msg.String() == "enter" {
				t.Cursor = 0
				t.Offset = 0
			}
			return t, true, nil
		}
	}

	var cmd tea.Cmd
	t.Filter, cmd = t.Filter.Update(msg)
	return t, false, cmd
}

// HandleClearFilter clears the filter when esc is pressed outside filter mode.
// Returns true if the filter was cleared.
func (t *TableBase) HandleClearFilter() bool {
	if t.Filter.Value() != "" {
		t.Filter.SetValue("")
		return true
	}
	return false
}

// HandleToggleExpanded toggles the expanded state when the detail is expanded.
// Returns true if expansion was toggled off.
func (t *TableBase) HandleCollapseIfExpanded() bool {
	if t.Expanded {
		t.Expanded = false
		return true
	}
	return false
}

// ResetPosition resets cursor and offset to the beginning.
func (t *TableBase) ResetPosition() {
	t.Cursor = 0
	t.Offset = 0
}

// FilterValue returns the current filter value.
func (t TableBase) FilterValue() string {
	return t.Filter.Value()
}

// IsFiltered returns true if a filter is active.
func (t TableBase) IsFiltered() bool {
	return t.Filter.Value() != ""
}
