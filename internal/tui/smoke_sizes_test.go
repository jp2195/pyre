package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

// allViews enumerates every screen the model can render.
func allViews() []struct {
	name string
	set  func(*Model)
} {
	return []struct {
		name string
		set  func(*Model)
	}{
		{"overview", func(m *Model) { m.currentView = ViewDashboard; m.currentDashboard = views.DashboardMain }},
		{"network", func(m *Model) { m.currentView = ViewDashboard; m.currentDashboard = views.DashboardNetwork }},
		{"security", func(m *Model) { m.currentView = ViewDashboard; m.currentDashboard = views.DashboardSecurity }},
		{"vpn", func(m *Model) { m.currentView = ViewDashboard; m.currentDashboard = views.DashboardVPN }},
		{"config", func(m *Model) { m.currentView = ViewDashboard; m.currentDashboard = views.DashboardConfig }},
		{"policies", func(m *Model) { m.currentView = ViewPolicies }},
		{"nat", func(m *Model) { m.currentView = ViewNATPolicies }},
		{"objects", func(m *Model) { m.currentView = ViewObjects }},
		{"sessions", func(m *Model) { m.currentView = ViewSessions }},
		{"interfaces", func(m *Model) { m.currentView = ViewInterfaces }},
		{"routes", func(m *Model) { m.currentView = ViewRoutes }},
		{"ipsec", func(m *Model) { m.currentView = ViewIPSecTunnels }},
		{"gpusers", func(m *Model) { m.currentView = ViewGPUsers }},
		{"logs/system", func(m *Model) { m.currentView = ViewLogs; m.logs = m.logs.SetActiveLogType(models.LogTypeSystem) }},
		{"logs/traffic", func(m *Model) { m.currentView = ViewLogs; m.logs = m.logs.SetActiveLogType(models.LogTypeTraffic) }},
		{"logs/threat", func(m *Model) { m.currentView = ViewLogs; m.logs = m.logs.SetActiveLogType(models.LogTypeThreat) }},
		{"hub", func(m *Model) { m.currentView = ViewConnectionHub }},
		{"form", func(m *Model) { m.currentView = ViewConnectionForm }},
		{"login", func(m *Model) { m.currentView = ViewLogin }},
		{"picker", func(m *Model) { m.currentView = ViewPicker }},
		{"devicepicker", func(m *Model) { m.currentView = ViewDevicePicker }},
		{"palette", func(m *Model) { m.currentView = ViewCommandPalette }},
	}
}

// seedEverything fills every view with one row of data.
func seedEverything(m *Model) {
	*m = func() Model {
		mm := *m
		mm.dashboard = mm.dashboard.SetSystemInfo(&models.SystemInfo{Hostname: "fw"}, nil)
		mm.dashboard = mm.dashboard.SetResources(&models.Resources{CPUPercent: 5}, nil)
		mm.dashboard = mm.dashboard.SetSessionInfo(&models.SessionInfo{ActiveCount: 1, MaxCount: 10}, nil)
		mm.dashboard = mm.dashboard.SetDiskUsage([]models.DiskUsage{{MountPoint: "/", Percent: 30}}, nil)
		mm.dashboard = mm.dashboard.SetLicenses([]models.LicenseInfo{{Feature: "Threat", DaysLeft: 5}}, nil)
		mm.dashboard = mm.dashboard.SetJobs([]models.Job{{ID: 1, Type: "Commit", Status: "FIN"}}, nil)
		mm.dashboard = mm.dashboard.SetCertificates([]models.Certificate{{Name: "c", Status: "expiring", DaysLeft: 3}}, nil)
		mm.dashboard = mm.dashboard.SetNATPoolInfo([]models.NATPoolInfo{{RuleName: "r", Percent: 50}}, nil)
		mm.dashboard = mm.dashboard.SetEnvironmentals([]models.Environmental{{Component: "Fan", Value: "1"}}, nil)
		mm.dashboard = mm.dashboard.SetLoggedInAdmins([]models.LoggedInAdmin{{Username: "admin"}}, nil)
		ts := &models.ThreatSummary{TotalThreats: 3, MediumCount: 3, BlockedCount: 1, AlertedCount: 2, SampleLimit: 100}
		mm.dashboard = mm.dashboard.SetThreatSummary(ts, nil)
		mm.securityDashboard = mm.securityDashboard.SetThreatSummary(ts, nil)
		rules := []models.SecurityRule{{Name: "r1", Position: 1, Action: "allow", HitCount: 5}}
		mm.policies = mm.policies.SetPolicies(rules, nil)
		mm.securityDashboard = mm.securityDashboard.SetPolicies(rules, nil)
		mm.configDashboard = mm.configDashboard.SetPolicies(rules, nil)
		mm.configDashboard = mm.configDashboard.SetPendingChanges([]models.PendingChange{{User: "admin", Type: "edit"}}, nil)
		mm.natPolicies = mm.natPolicies.SetRules([]models.NATRule{{Name: "n1", Position: 1}}, nil)
		ifaces := []models.Interface{{Name: "ethernet1/1", State: "up", Zone: "z", IP: "10.0.0.1/24"}}
		mm.interfaces = mm.interfaces.SetInterfaces(ifaces, nil)
		mm.networkDashboard = mm.networkDashboard.SetInterfaces(ifaces, nil)
		mm.networkDashboard = mm.networkDashboard.SetARPTable([]models.ARPEntry{{IP: "10.0.0.2"}}, nil)
		routes := []models.RouteEntry{{Destination: "0.0.0.0/0", Nexthop: "10.0.0.1", Protocol: "static"}}
		mm.networkDashboard = mm.networkDashboard.SetRoutingTable(routes, nil)
		mm.routes = mm.routes.SetRoutes(routes, nil)
		mm.routes = mm.routes.SetBGPNeighbors([]models.BGPNeighbor{{PeerAddress: "10.0.0.2", State: "Established"}}, nil)
		tunnels := []models.IPSecTunnel{{Name: "t1", State: "up"}}
		mm.ipsecTunnels = mm.ipsecTunnels.SetTunnels(tunnels, nil)
		mm.vpnDashboard = mm.vpnDashboard.SetIPSecTunnels(tunnels, nil)
		gp := []models.GlobalProtectUser{{Username: "u1", VirtualIP: "10.1.1.1"}}
		mm.gpUsers = mm.gpUsers.SetUsers(gp, nil)
		mm.vpnDashboard = mm.vpnDashboard.SetGlobalProtectUsers(gp, nil)
		mm.sessions = mm.sessions.SetSessions([]models.Session{{ID: 1, SourceIP: "10.0.0.5", Protocol: "tcp"}}, nil)
		mm.objects = mm.objects.SetAddresses([]models.AddressObject{{Name: "a", Type: "ip-netmask", Value: "1.1.1.1"}}, nil)
		mm.objects = mm.objects.SetServices([]models.ServiceObject{{Name: "s", Protocol: "tcp", DestPort: "443"}}, nil)
		mm.logs, _ = mm.logs.SetSystemLogs([]models.SystemLogEntry{{Time: time.Now(), Severity: "high", Description: "d"}}, views.LogPageMeta{}, nil)
		mm.logs, _ = mm.logs.SetTrafficLogs([]models.TrafficLogEntry{{Time: time.Now(), Action: "allow", SourceIP: "1.1.1.1"}}, views.LogPageMeta{}, nil)
		mm.logs, _ = mm.logs.SetThreatLogs([]models.ThreatLogEntry{{Time: time.Now(), Severity: "high", ThreatName: "t"}}, views.LogPageMeta{}, nil)
		return mm
	}()
}

// TestRender_SurvivesPathologicalSizes renders every screen at sizes a user
// can actually produce by dragging a terminal edge. A width-derived value
// going negative anywhere turns into a panic that takes the whole program
// down, and a TUI is resized far more often than it is restarted.
func TestRender_SurvivesPathologicalSizes(t *testing.T) {
	sizes := []struct{ w, h int }{
		{1, 1}, {2, 3}, {5, 5}, {10, 8}, {20, 10}, {30, 12},
		{40, 15}, {60, 20}, {79, 24}, {80, 24}, {120, 30}, {300, 80},
	}
	for _, dim := range sizes {
		for _, v := range allViews() {
			m := newTestModel(t, ViewDashboard)
			seedEverything(&m)
			u, _ := m.Update(tea.WindowSizeMsg{Width: dim.w, Height: dim.h})
			m = u.(Model)
			v.set(&m)

			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s panicked at %dx%d: %v", v.name, dim.w, dim.h, r)
					}
				}()
				_ = m.renderContent()
			}()
		}
	}
}

// TestRender_DoesNotOverflowTerminalWidth checks that no screen emits a line
// wider than the terminal. Because renderContent joins lines vertically,
// lipgloss pads every line to the widest one, so a single overlong line
// shifts the whole interface off the right edge rather than just wrapping
// itself. That is how the fixed-width application footer broke every screen
// below 95 columns.
func TestRender_DoesNotOverflowTerminalWidth(t *testing.T) {
	// 60 columns is the narrowest terminal the application supports. The
	// panic sweep above covers everything below it; this one covers what it
	// should look like.
	sizes := []struct{ w, h int }{
		{60, 20}, {70, 22}, {80, 24}, {100, 30}, {120, 40}, {160, 45}, {200, 50},
	}
	for _, dim := range sizes {
		for _, v := range allViews() {
			m := newTestModel(t, ViewDashboard)
			seedEverything(&m)
			u, _ := m.Update(tea.WindowSizeMsg{Width: dim.w, Height: dim.h})
			m = u.(Model)
			v.set(&m)

			worst, worstLine := 0, ""
			for line := range strings.SplitSeq(m.renderContent(), "\n") {
				if w := lipgloss.Width(line); w > worst {
					worst, worstLine = w, line
				}
			}
			if worst > dim.w {
				t.Errorf("%s at %dx%d: widest line is %d cells, %d over\n  %q",
					v.name, dim.w, dim.h, worst, worst-dim.w, worstLine)
			}
		}
	}
}
