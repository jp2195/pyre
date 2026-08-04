package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

func TestDispatch_AddressesMsg_RoutesToObjectsModel(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	msg := AddressesMsg{Items: []models.AddressObject{
		{Name: "web-servers", Type: "ip-netmask", Value: "10.0.0.0/24"},
	}}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	if got := len(model.objects.Addresses()); got != 1 {
		t.Errorf("expected 1 address routed to ObjectsModel, got %d", got)
	}
}

func TestDispatch_ServicesMsg_RoutesToObjectsModel(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	msg := ServicesMsg{Items: []models.ServiceObject{
		{Name: "tcp-443", Protocol: "tcp", DestPort: "443"},
	}}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	if got := len(model.objects.Services()); got != 1 {
		t.Errorf("expected 1 service routed to ObjectsModel, got %d", got)
	}
}

// TestDispatch_TabOnObjectsView_NavigatesAway pins that Objects no longer
// swallows Tab. It used to cycle the Address/Service sub-tabs, which made
// Objects the one view you could not Tab out of even though the footer
// advertises "Tab/S-Tab next/prev". a/s still select sub-tabs directly.
func TestDispatch_TabOnObjectsView_NavigatesAway(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	// Navigate to Objects view via SwitchViewMsg, then send Tab through Model.Update.
	updated, _ := m.Update(SwitchViewMsg{View: ViewObjects})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(Model)

	if m.objects.ActiveTab() != views.ObjectsTabAddress {
		t.Errorf("Tab must not cycle Objects sub-tabs; got tab=%v", m.objects.ActiveTab())
	}
	if m.currentView == ViewObjects {
		t.Error("Tab on Objects view should navigate to the next Analyze item")
	}
}
