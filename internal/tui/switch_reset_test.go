package tui

// switch_reset_test.go – regression tests for connection / Panorama-target
// switching. Both failures these cover put one device's data on screen while
// the header named a different device, which is the worst failure mode this
// tool has: the operator reads the wrong firewall's rules and believes them.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
)

// seedViewData fills a representative spread of sub-views with data so a
// reset is observable: one dashboard, one generic list view, and the logs
// view (which keeps its own non-RuleList shell).
func seedViewData(m *Model) {
	m.dashboard = m.dashboard.SetSystemInfo(&models.SystemInfo{Hostname: "old-fw"}, nil)
	m.policies = m.policies.SetPolicies([]models.SecurityRule{{Name: "old-rule"}}, nil)
	m.sessions = m.sessions.SetSessions([]models.Session{{ID: 1}}, nil)
	m.logs = m.logs.SetSystemLogs([]models.SystemLogEntry{{Description: "old"}}, nil)
	m.objects = m.objects.SetAddresses([]models.AddressObject{{Name: "old-addr"}}, nil)
}

// assertViewDataCleared fails if any seeded view still reports data.
func assertViewDataCleared(t *testing.T, m Model) {
	t.Helper()
	for _, tc := range []struct {
		name    string
		hasData bool
	}{
		{"dashboard", m.dashboard.HasData()},
		{"policies", m.policies.HasData()},
		{"sessions", m.sessions.HasData()},
		{"logs", m.logs.HasData()},
		{"objects", m.objects.HasData()},
	} {
		if tc.hasData {
			t.Errorf("%s still reports data after switch; previous device's data would render under the new device's name", tc.name)
		}
	}
}

// newPanoramaTestModel builds a model with one active Panorama connection.
func newPanoramaTestModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t, ViewDashboard)
	conn, err := m.session.AddConnection("pano.example.com", &config.ConnectionConfig{}, "key")
	if err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	conn.SetPanoramaInfo(true)
	return m
}

func TestDevicePickerSelect_ClearsPreviousDeviceData(t *testing.T) {
	m := newPanoramaTestModel(t)
	devices := []models.ManagedDevice{
		{Serial: "001122334455", Hostname: "fw-a"},
		{Serial: "556677889900", Hostname: "fw-b"},
	}
	m.currentView = ViewDevicePicker
	// Cursor lands on the device matching the current target.
	m.devicePicker = m.devicePicker.SetDevices(devices, "001122334455", "pano.example.com")
	seedViewData(&m)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := updated.(Model)

	if got := nm.session.GetActiveConnection().Target(); got != "001122334455" {
		t.Fatalf("target = %q, want the selected device serial", got)
	}
	assertViewDataCleared(t, nm)
}

// TestDevicePickerSelect_PreservesViewSizes guards the obvious wrong fix:
// replacing the sub-models with fresh zero-valued ones drops the terminal
// dimensions, so every view renders its zero-width "Loading..." placeholder
// until the next resize event that may never come.
func TestDevicePickerSelect_PreservesViewSizes(t *testing.T) {
	m := newPanoramaTestModel(t)
	m.currentView = ViewDevicePicker
	m.devicePicker = m.devicePicker.SetDevices(
		[]models.ManagedDevice{{Serial: "001122334455", Hostname: "fw-a"}},
		"001122334455", "pano.example.com",
	)
	// Establish sizes the way the runtime does.
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	m = sized.(Model)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := updated.(Model)

	if nm.dashboard.Width != 200 {
		t.Errorf("dashboard width = %d after switch, want 200 (sizes must survive the reset)", nm.dashboard.Width)
	}
	if nm.dashboard.Height != 50-4 {
		t.Errorf("dashboard height = %d after switch, want %d", nm.dashboard.Height, 50-4)
	}
	if nm.networkDashboard.Width != 200 {
		t.Errorf("networkDashboard width = %d after switch, want 200", nm.networkDashboard.Width)
	}
}

func TestPickerSelect_ClearsPreviousConnectionData(t *testing.T) {
	m := newTestModel(t, ViewPicker)
	if _, err := m.session.AddConnection("10.0.0.1", &config.ConnectionConfig{}, "k1"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	if _, err := m.session.AddConnection("10.0.0.2", &config.ConnectionConfig{}, "k2"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	m.currentView = ViewPicker
	m.picker = m.picker.UpdateConnections(m.session)
	seedViewData(&m)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	nm := updated.(Model)

	assertViewDataCleared(t, nm)
}

// TestLoginSuccess_SecondHostBecomesActive covers the second half of the same
// failure: AddConnection only claimed the active slot when it was empty, so
// logging into a second firewall left every view pointed at the first one.
func TestLoginSuccess_SecondHostBecomesActive(t *testing.T) {
	m := newTestModel(t, ViewLogin)
	if _, err := m.session.AddConnection("10.0.0.1", &config.ConnectionConfig{}, "k1"); err != nil {
		t.Fatalf("AddConnection: %v", err)
	}
	seedViewData(&m)

	updated, _ := m.Update(LoginSuccessMsg{Host: "10.0.0.2", APIKey: "k2", Username: "admin"})
	nm := updated.(Model)

	conn := nm.session.GetActiveConnection()
	if conn == nil {
		t.Fatal("expected an active connection after login")
	}
	if conn.Host != "10.0.0.2" {
		t.Errorf("active host = %q, want 10.0.0.2 (the host just logged into)", conn.Host)
	}
	assertViewDataCleared(t, nm)
}

// TestView_DoesNotEnableMouseMode pins a deliberate choice. Mouse reporting
// was switched on while nothing in the program handles a mouse message, so it
// bought nothing and cost two things an operator notices: the terminal's own
// click-drag text selection stops working without holding Shift, which is how
// you copy an IP out of a table, and every wheel and motion event falls
// through to the unhandled-message warning. Re-enable it only alongside real
// mouse handling.
func TestView_DoesNotEnableMouseMode(t *testing.T) {
	m := newTestModel(t, ViewDashboard)
	if got := m.View().MouseMode; got != tea.MouseModeNone {
		t.Errorf("MouseMode = %v, want MouseModeNone while no mouse messages are handled", got)
	}
}
