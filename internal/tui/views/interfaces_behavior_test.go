package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func pressInterfaces(m InterfacesModel, keys string) InterfacesModel {
	for _, r := range keys {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestInterfaces_Behavior_FilterKeystrokesNarrowRows(t *testing.T) {
	InitStyles()
	m := NewInterfacesModel()
	m = m.SetSize(120, 30)
	m = m.SetInterfaces([]models.Interface{
		{Name: "ethernet1/1", Zone: "trust", State: "up"},
		{Name: "loopback.1", Zone: "mgmt", State: "up"},
	}, nil)

	m = pressInterfaces(m, "/loopback")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	out := m.View()
	if !strings.Contains(out, "loopback.1") {
		t.Errorf("expected loopback.1 after filter:\n%s", out)
	}
	if strings.Contains(out, "ethernet1/1") {
		t.Errorf("expected ethernet1/1 filtered out:\n%s", out)
	}
}

func TestInterfaces_Behavior_DefaultSortNameAscending(t *testing.T) {
	InitStyles()
	m := NewInterfacesModel()
	m = m.SetSize(120, 30)
	m = m.SetInterfaces([]models.Interface{
		{Name: "zz-iface", State: "up"},
		{Name: "aa-iface", State: "up"},
	}, nil)

	out := m.View()
	if strings.Index(out, "aa-iface") > strings.Index(out, "zz-iface") {
		t.Errorf("expected aa-iface first with default ascending name sort:\n%s", out)
	}
}

func TestInterfaces_Behavior_EscCollapsesThenClearsFilter(t *testing.T) {
	InitStyles()
	m := NewInterfacesModel()
	m = m.SetSize(120, 40)
	m = m.SetInterfaces([]models.Interface{
		{Name: "ethernet1/1", State: "up"},
		{Name: "loopback.1", State: "up"},
	}, nil)

	// Filter down to ethernet, then expand detail.
	m = pressInterfaces(m, "/ethernet")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // commit filter
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // expand detail
	out := m.View()
	if !strings.Contains(out, "Interface Details") {
		t.Fatalf("expected expanded detail panel:\n%s", out)
	}

	// First esc collapses the detail; filter still applied.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	out = m.View()
	if strings.Contains(out, "Interface Details") {
		t.Errorf("expected detail collapsed after first esc:\n%s", out)
	}
	if strings.Contains(out, "loopback.1") {
		t.Errorf("expected filter still active after first esc:\n%s", out)
	}

	// Second esc clears the filter.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	out = m.View()
	if !strings.Contains(out, "loopback.1") {
		t.Errorf("expected filter cleared after second esc:\n%s", out)
	}
}

func TestInterfaces_Behavior_DetailShowsARPEntries(t *testing.T) {
	InitStyles()
	m := NewInterfacesModel()
	m = m.SetSize(120, 40)
	m = m.SetInterfaces([]models.Interface{
		{Name: "ethernet1/1", State: "up", IP: "10.0.0.1/24"},
	}, nil)
	m = m.SetARPTable([]models.ARPEntry{
		{Interface: "ethernet1/1", IP: "10.0.0.50", MAC: "aa:bb:cc:00:11:22", Status: "complete"},
		{Interface: "ethernet1/9", IP: "10.9.9.9", MAC: "ff:ff:ff:00:00:00", Status: "complete"},
	})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	out := m.View()
	if !strings.Contains(out, "ARP Entries") {
		t.Errorf("expected ARP Entries section in detail:\n%s", out)
	}
	if !strings.Contains(out, "10.0.0.50") {
		t.Errorf("expected ARP entry for this interface:\n%s", out)
	}
	if strings.Contains(out, "10.9.9.9") {
		t.Errorf("expected other interface's ARP entry excluded:\n%s", out)
	}
}
