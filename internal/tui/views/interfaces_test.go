package views

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestInterfacesModel_RenderEmitsValidUTF8(t *testing.T) {
	InitStyles()
	m := NewInterfacesModel()
	m = m.SetSize(120, 20)
	m = m.SetInterfaces([]models.Interface{
		{Name: "ethernet1/1", State: "up", Type: "layer3", Zone: "trust", IP: "10.0.0.1/24", MAC: "aa:bb:cc:dd:ee:ff", VirtualRouter: "default"},
		{Name: "ethernet1/2", State: "down", Type: "layer3", Zone: "untrust", IP: "", MAC: "", VirtualRouter: ""},
	}, nil)

	out := m.View()
	if !utf8.ValidString(out) {
		t.Fatalf("View() output contains invalid UTF-8\n--- output ---\n%s\n--- end ---", out)
	}
	if !strings.Contains(out, "ethernet1/1") {
		t.Errorf("expected 'ethernet1/1' in output, got: %s", out)
	}
	if !strings.Contains(out, "ethernet1/2") {
		t.Errorf("expected 'ethernet1/2' in output, got: %s", out)
	}
}

func TestNewInterfacesModel(t *testing.T) {
	m := NewInterfacesModel()

	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.list.Cursor)
	}
	if m.list.sortBy != 0 {
		t.Errorf("expected default sort by Name (0), got %d", m.list.sortBy)
	}
	if !m.list.SortAsc {
		t.Error("expected SortAsc=true by default")
	}
}

func TestInterfacesModel_SetSize(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(100, 50)

	if m.list.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.list.Width)
	}
	if m.list.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.list.Height)
	}
}

func TestInterfacesModel_SetLoading(t *testing.T) {
	m := NewInterfacesModel()

	m = m.SetLoading(true)
	if !m.list.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.list.Loading {
		t.Error("expected Loading=false")
	}
}

func TestInterfacesModel_SetInterfaces(t *testing.T) {
	m := NewInterfacesModel()

	interfaces := []models.Interface{
		{Name: "ethernet1/1", Zone: "trust", State: "up", IP: "10.0.0.1/24"},
		{Name: "ethernet1/2", Zone: "untrust", State: "up", IP: "192.168.1.1/24"},
	}

	m = m.SetInterfaces(interfaces, nil)

	if len(m.list.Items()) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(m.list.Items()))
	}
	if m.list.Loading {
		t.Error("expected Loading=false after SetInterfaces")
	}
}

func TestInterfacesModel_SetInterfaces_WithError(t *testing.T) {
	m := NewInterfacesModel()

	err := errors.New("API error")
	m = m.SetInterfaces(nil, err)

	if m.list.Err != err {
		t.Error("expected error to be set")
	}
}

func TestInterfacesModel_Filtering(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(100, 50)

	interfaces := []models.Interface{
		{Name: "ethernet1/1", Zone: "trust", State: "up"},
		{Name: "ethernet1/2", Zone: "untrust", State: "down"},
		{Name: "loopback.1", Zone: "mgmt", State: "up"},
	}

	m = m.SetInterfaces(interfaces, nil)

	// Initially all interfaces should be visible
	if len(m.list.Filtered()) != 3 {
		t.Errorf("expected 3 filtered interfaces, got %d", len(m.list.Filtered()))
	}

	// Apply filter for zone
	m.list.Filter.SetValue("trust")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 2 {
		t.Errorf("expected 2 filtered interfaces for 'trust', got %d", len(m.list.Filtered()))
	}

	// Filter for specific interface
	m.list.Filter.SetValue("loopback")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 1 {
		t.Errorf("expected 1 filtered interface for 'loopback', got %d", len(m.list.Filtered()))
	}

	// Clear filter
	m.list.Filter.SetValue("")
	m.list.applyFilter()

	if len(m.list.Filtered()) != 3 {
		t.Errorf("expected 3 filtered interfaces after clear, got %d", len(m.list.Filtered()))
	}
}

func TestInterfacesModel_Update_Navigation(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(100, 50)

	interfaces := []models.Interface{
		{Name: "ethernet1/1"},
		{Name: "ethernet1/2"},
		{Name: "ethernet1/3"},
	}
	m = m.SetInterfaces(interfaces, nil)

	// Move down
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.list.Cursor != 1 {
		t.Errorf("expected Cursor=1 after down, got %d", m.list.Cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0 after up, got %d", m.list.Cursor)
	}
}

func TestInterfacesModel_View(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(100, 50)

	// View without interfaces
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View with interfaces
	interfaces := []models.Interface{
		{Name: "ethernet1/1", Zone: "trust", State: "up", IP: "10.0.0.1/24"},
	}
	m = m.SetInterfaces(interfaces, nil)

	view = m.View()
	if view == "" {
		t.Error("expected non-empty view with interfaces")
	}
}

func TestInterfacesModel_View_ZeroWidth(t *testing.T) {
	m := NewInterfacesModel()
	// Don't set size

	view := m.View()
	if !strings.Contains(view, "Loading...") {
		t.Errorf("expected view to contain 'Loading...' with zero width, got %q", view)
	}
}

func TestInterfacesModel_EscClearsFilterAndCollapsesDetail(t *testing.T) {
	InitStyles()
	m := NewInterfacesModel()
	m = m.SetSize(120, 20)
	m = m.SetInterfaces([]models.Interface{{Name: "ethernet1/1", State: "up"}}, nil)

	// Set initial state: filter applied AND detail expanded.
	m.list.Filter.SetValue("eth")
	m.list.applyFilter()
	m.list.Expanded = true

	// First esc: collapse the detail panel.
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if updated.list.Expanded {
		t.Error("expected detail panel collapsed after first esc")
	}
	if updated.list.Filter.Value() != "eth" {
		t.Errorf("expected filter still 'eth' after first esc, got %q", updated.list.Filter.Value())
	}

	// Second esc: clear the filter.
	updated, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if updated.list.Filter.Value() != "" {
		t.Errorf("expected filter cleared after second esc, got %q", updated.list.Filter.Value())
	}
}

func TestInterfacesModel_SetSize_ClampsCursor(t *testing.T) {
	m := NewInterfacesModel()

	// Set up interfaces and move cursor to end
	interfaces := []models.Interface{
		{Name: "ethernet1/1"},
		{Name: "ethernet1/2"},
		{Name: "ethernet1/3"},
		{Name: "ethernet1/4"},
		{Name: "ethernet1/5"},
	}
	m = m.SetSize(100, 50) // Large enough for all
	m = m.SetInterfaces(interfaces, nil)

	// Move cursor to end using table navigation
	for range 4 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if m.list.Cursor != 4 {
		t.Errorf("expected cursor at 4, got %d", m.list.Cursor)
	}

	// Apply filter that reduces items
	m.list.Filter.SetValue("ethernet1/1")
	m.list.applyFilter()

	// Now resize - cursor should be clamped
	m = m.SetSize(100, 50)

	// Cursor should be clamped to valid range (0 since only 1 item matches)
	if m.list.Cursor >= len(m.list.Filtered()) {
		t.Errorf("cursor %d should be less than filtered count %d after resize", m.list.Cursor, len(m.list.Filtered()))
	}
}

func TestInterfacesModel_SetSpinnerFrame_ReachesList(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSpinnerFrame("◢")
	if m.list.SpinnerFrame != "◢" {
		t.Errorf("list.SpinnerFrame = %q, want ◢", m.list.SpinnerFrame)
	}
}

// visibleWidth returns the rune width of a line with ANSI escapes removed.
func visibleWidth(s string) int {
	return len([]rune(stripANSI(s)))
}

// TestInterfaceDetail_FitsWithinGivenWidth pins that the detail panel sizes
// itself to the width it is handed. RuleListModel renders it *inside* a
// bordered, padded panel but used to pass the raw terminal width, so the
// detail's own border overflowed the container and wrapped — which is what
// made the borders look mangled and every row look double-spaced.
func TestInterfaceDetail_FitsWithinGivenWidth(t *testing.T) {
	InitStyles()
	iface := models.Interface{
		Name: "loopback.60", Type: "loopback", Zone: "dmz", State: "down",
		IP: "10.0.105.1/32", VirtualRouter: "lr:default", Vsys: "1",
	}

	for _, width := range []int{80, 100, 128, 160} {
		out := renderInterfaceDetail(iface, width, nil)
		for i, line := range strings.Split(out, "\n") {
			if got := visibleWidth(line); got > width {
				t.Errorf("width=%d: line %d is %d cols wide, overflows its container:\n%s",
					width, i, got, stripANSI(line))
			}
		}
	}
}

// TestInterfaceDetail_HeadingStaysWithItsFields guards the two-column split.
// It used to cut the flat line list at its midpoint, stranding the "Network"
// heading at the bottom of the left column while IP/MAC/Virtual Router began
// the right column — the heading rendered empty and the fields header-less.
func TestInterfaceDetail_HeadingStaysWithItsFields(t *testing.T) {
	InitStyles()
	iface := models.Interface{
		Name: "loopback.60", Type: "loopback", Zone: "dmz", State: "down",
		IP: "10.0.105.1/32", MAC: "3c:fa:30:f7:34:03", VirtualRouter: "lr:default", Vsys: "1",
	}

	// Wide enough to trigger the two-column layout.
	lines := strings.Split(stripANSI(renderInterfaceDetail(iface, 130, nil)), "\n")

	netLine := -1
	for i, l := range lines {
		if strings.Contains(l, "Network") {
			netLine = i
			break
		}
	}
	if netLine < 0 {
		t.Fatalf("no Network section rendered:\n%s", strings.Join(lines, "\n"))
	}
	if netLine == len(lines)-1 || !strings.Contains(lines[netLine+1], "IP Address") {
		t.Errorf("the Network heading is not followed by its own fields; "+
			"the column split separated them:\n%s", strings.Join(lines, "\n"))
	}
}

// TestSplitSectionsIntoColumns_KeepsSectionsIntact verifies sections are
// indivisible and the two columns come out close in height.
func TestSplitSectionsIntoColumns_KeepsSectionsIntact(t *testing.T) {
	sections := [][]string{
		{"Basic", "a", "b", "c", "d", "e"}, // 6
		{"Network", "f", "g", "h"},         // 4
		{"Physical", "i", "j"},             // 3
	}

	left, right := splitSectionsIntoColumns(sections)

	rebuilt := make([][]string, 0, len(left)+len(right))
	rebuilt = append(rebuilt, left...)
	rebuilt = append(rebuilt, right...)
	if len(rebuilt) != len(sections) {
		t.Fatalf("expected %d sections preserved, got %d", len(sections), len(rebuilt))
	}
	for i := range sections {
		if len(rebuilt[i]) != len(sections[i]) {
			t.Errorf("section %d was split: had %d lines, now %d",
				i, len(sections[i]), len(rebuilt[i]))
		}
	}

	height := func(cols [][]string) int {
		n := 0
		for _, c := range cols {
			n += len(c)
		}
		return n
	}
	if diff := height(left) - height(right); diff > 3 || diff < -3 {
		t.Errorf("columns badly unbalanced: left=%d right=%d", height(left), height(right))
	}
}
