package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewInterfacesModel(t *testing.T) {
	m := NewInterfacesModel()

	if m.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.Cursor)
	}
	if m.sortBy != InterfaceSortName {
		t.Errorf("expected default sort by Name, got %d", m.sortBy)
	}
	if !m.SortAsc {
		t.Error("expected SortAsc=true by default")
	}
}

func TestInterfacesModel_SetSize(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(100, 50)

	if m.Width != 100 {
		t.Errorf("expected Width=100, got %d", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("expected Height=50, got %d", m.Height)
	}
}

func TestInterfacesModel_SetLoading(t *testing.T) {
	m := NewInterfacesModel()

	m = m.SetLoading(true)
	if !m.Loading {
		t.Error("expected Loading=true")
	}

	m = m.SetLoading(false)
	if m.Loading {
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

	if len(m.interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(m.interfaces))
	}
	if m.Loading {
		t.Error("expected Loading=false after SetInterfaces")
	}
}

func TestInterfacesModel_SetInterfaces_WithError(t *testing.T) {
	m := NewInterfacesModel()

	err := errors.New("API error")
	m = m.SetInterfaces(nil, err)

	if m.Err != err {
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
	if len(m.filtered) != 3 {
		t.Errorf("expected 3 filtered interfaces, got %d", len(m.filtered))
	}

	// Apply filter for zone
	m.Filter.SetValue("trust")
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered interfaces for 'trust', got %d", len(m.filtered))
	}

	// Filter for specific interface
	m.Filter.SetValue("loopback")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered interface for 'loopback', got %d", len(m.filtered))
	}

	// Clear filter
	m.Filter.SetValue("")
	m.applyFilter()

	if len(m.filtered) != 3 {
		t.Errorf("expected 3 filtered interfaces after clear, got %d", len(m.filtered))
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

	// Move down (table uses down arrow or j)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.TableCursor() != 1 {
		t.Errorf("expected TableCursor=1 after down, got %d", m.TableCursor())
	}

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.TableCursor() != 0 {
		t.Errorf("expected TableCursor=0 after up, got %d", m.TableCursor())
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
	if view != "Loading..." {
		t.Errorf("expected 'Loading...' with zero width, got %q", view)
	}
}

func TestInterfaceSortField_Constants(t *testing.T) {
	if InterfaceSortName != 0 {
		t.Errorf("expected InterfaceSortName=0, got %d", InterfaceSortName)
	}
	if InterfaceSortZone != 1 {
		t.Errorf("expected InterfaceSortZone=1, got %d", InterfaceSortZone)
	}
	if InterfaceSortState != 2 {
		t.Errorf("expected InterfaceSortState=2, got %d", InterfaceSortState)
	}
	if InterfaceSortIP != 3 {
		t.Errorf("expected InterfaceSortIP=3, got %d", InterfaceSortIP)
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
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.TableCursor() != 4 {
		t.Errorf("expected cursor at 4, got %d", m.TableCursor())
	}

	// Apply filter that reduces items
	m.Filter.SetValue("ethernet1/1")
	m.applyFilter()
	m.updateTableRows()

	// Now resize - cursor should be clamped
	m = m.SetSize(100, 50)

	// Cursor should be clamped to valid range (0 since only 1 item matches)
	if m.TableCursor() >= len(m.filtered) {
		t.Errorf("cursor %d should be less than filtered count %d after resize", m.TableCursor(), len(m.filtered))
	}
}
