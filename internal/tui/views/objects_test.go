package views

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewObjectsModel(t *testing.T) {
	m := NewObjectsModel()
	if m.ActiveTab() != ObjectsTabAddress {
		t.Errorf("expected default ActiveTab=ObjectsTabAddress, got %v", m.ActiveTab())
	}
	if m.HasData() {
		t.Error("expected HasData=false before any data is set")
	}
}

func TestObjectsModel_SetAddresses_HasDataTrue(t *testing.T) {
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{
		{Name: "web-servers", Type: "ip-netmask", Value: "10.0.0.0/24"},
	}, nil)
	if !m.HasData() {
		t.Error("expected HasData=true after SetAddresses")
	}
}

func TestObjectsModel_SetServices_DoesNotAffectAddressTab(t *testing.T) {
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{
		{Name: "web-servers", Type: "ip-netmask", Value: "10.0.0.0/24"},
	}, nil)
	m = m.SetServices([]models.ServiceObject{
		{Name: "tcp-443", Protocol: "tcp", DestPort: "443"},
	}, nil)

	if got := len(m.Addresses()); got != 1 {
		t.Errorf("expected 1 address still present, got %d", got)
	}
	if got := len(m.Services()); got != 1 {
		t.Errorf("expected 1 service present, got %d", got)
	}
}

func TestObjectsModel_TabSwitching(t *testing.T) {
	m := NewObjectsModel()
	m = m.SetSize(120, 30)

	// Tab key advances Address -> Service.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.ActiveTab() != ObjectsTabService {
		t.Errorf("after Tab, expected ObjectsTabService, got %v", m.ActiveTab())
	}

	// Tab again wraps Service -> Address.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.ActiveTab() != ObjectsTabAddress {
		t.Errorf("after second Tab, expected ObjectsTabAddress, got %v", m.ActiveTab())
	}

	// 's' jumps to Service tab from Address.
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if m.ActiveTab() != ObjectsTabService {
		t.Errorf("after 's', expected ObjectsTabService, got %v", m.ActiveTab())
	}

	// 'a' jumps back to Address.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if m.ActiveTab() != ObjectsTabAddress {
		t.Errorf("after 'a', expected ObjectsTabAddress, got %v", m.ActiveTab())
	}
}

func TestObjectsModel_FilterPerTab(t *testing.T) {
	InitStyles()
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{
		{Name: "web-servers", Type: "ip-netmask", Value: "10.0.0.0/24"},
		{Name: "db-cluster", Type: "ip-netmask", Value: "10.1.0.0/16"},
	}, nil)
	m = m.SetServices([]models.ServiceObject{
		{Name: "tcp-443", Protocol: "tcp", DestPort: "443"},
		{Name: "udp-dns", Protocol: "udp", DestPort: "53"},
	}, nil)

	// Apply filter on Address tab.
	m.addressTab.Filter.SetValue("web")
	m.addressTab.applyFilter()
	if got := len(m.addressTab.filtered); got != 1 {
		t.Errorf("expected 1 filtered address, got %d", got)
	}

	// Switch to Service tab; its filter is unset, so all services visible.
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if got := len(m.serviceTab.filtered); got != 2 {
		t.Errorf("expected 2 services (no filter), got %d", got)
	}

	// Apply filter on Service tab.
	m.serviceTab.Filter.SetValue("udp")
	m.serviceTab.applyFilter()
	if got := len(m.serviceTab.filtered); got != 1 {
		t.Errorf("expected 1 filtered service, got %d", got)
	}

	// Switch back to Address tab; original filter still applied.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if got := m.addressTab.Filter.Value(); got != "web" {
		t.Errorf("expected Address tab filter preserved as 'web', got %q", got)
	}
	if got := len(m.addressTab.filtered); got != 1 {
		t.Errorf("expected Address tab still filtered to 1, got %d", got)
	}
}

func TestObjectsModel_RenderEmitsValidUTF8(t *testing.T) {
	InitStyles()
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{
		{Name: "web-servers", Type: "ip-netmask", Value: "10.0.0.0/24", Tags: []string{"prod", "web"}},
		{Name: "partner-vpn", Type: "fqdn", Value: "vpn.partner.example.com"},
	}, nil)
	m = m.SetServices([]models.ServiceObject{
		{Name: "tcp-443", Protocol: "tcp", DestPort: "443"},
	}, nil)

	out := m.View()
	if !utf8.ValidString(out) {
		t.Fatalf("Objects view contains invalid UTF-8\n--- output ---\n%s\n--- end ---", out)
	}
	if !strings.Contains(out, "web-servers") {
		t.Errorf("expected 'web-servers' in Address tab render, got: %s", out)
	}
}

func TestObjectsModel_View_EmptyState_PerTab(t *testing.T) {
	InitStyles()
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{}, nil)
	m = m.SetServices([]models.ServiceObject{}, nil)

	out := m.View()
	if !strings.Contains(out, "No address objects") {
		t.Errorf("expected 'No address objects' message, got: %s", out)
	}

	// Switch to Service tab.
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	out = m.View()
	if !strings.Contains(out, "No service objects") {
		t.Errorf("expected 'No service objects' message on Service tab, got: %s", out)
	}
}

func TestObjectsModel_View_ZeroWidth(t *testing.T) {
	m := NewObjectsModel()
	out := m.View()
	if !strings.Contains(out, "Loading") {
		t.Errorf("expected 'Loading' when width=0, got: %s", out)
	}
}

func TestObjectsModel_SortCycle_PerTab(t *testing.T) {
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{
		{Name: "b", Type: "fqdn", Value: "z"},
		{Name: "a", Type: "ip-netmask", Value: "y"},
	}, nil)

	// Default: sort by Name ascending — first row is "a".
	if got := m.addressTab.filtered[0].Name; got != "a" {
		t.Fatalf("default sort: first row Name=%q, want %q", got, "a")
	}

	// Capital S cycles to Type sort.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if m.addressTab.sortBy != AddressSortType {
		t.Errorf("after S, expected sortBy=AddressSortType, got %v", m.addressTab.sortBy)
	}
	// After Type sort, "fqdn" < "ip-netmask" so "b" should be first.
	if got := m.addressTab.filtered[0].Name; got != "b" {
		t.Errorf("after Type sort: first row Name=%q, want %q (fqdn sorts before ip-netmask)", got, "b")
	}
	if !m.addressTab.SortAsc {
		t.Error("after S, expected SortAsc=true")
	}
	if m.addressTab.Cursor != 0 || m.addressTab.Offset != 0 {
		t.Errorf("after S, expected Cursor=0 Offset=0, got Cursor=%d Offset=%d",
			m.addressTab.Cursor, m.addressTab.Offset)
	}
}

func TestObjectsModel_DetailPanel_EnterOpensEscCloses(t *testing.T) {
	InitStyles()
	m := NewObjectsModel()
	m = m.SetSize(120, 30)
	m = m.SetAddresses([]models.AddressObject{
		{Name: "web-servers", Type: "ip-netmask", Value: "10.0.0.0/24", Description: "Prod web", Tags: []string{"prod"}},
	}, nil)

	// Enter expands.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.addressTab.Expanded {
		t.Error("expected addressTab.Expanded=true after Enter")
	}

	// View now contains the description.
	out := m.View()
	if !strings.Contains(out, "Prod web") {
		t.Errorf("expected detail panel to show description, got: %s", out)
	}

	// Esc collapses.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.addressTab.Expanded {
		t.Error("expected addressTab.Expanded=false after Esc")
	}
}
