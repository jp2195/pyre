package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
)

// TestFilterMode_GlobalKeysBypassGlobalHandlers guards against the
// filter-mode key leak: while a view's filter input is focused, global
// bindings (q=quit, r=refresh, 1-3=navigate) must be routed to the view,
// not the global handler. Before the fix this guard existed only for
// ViewLogs.
func TestFilterMode_GlobalKeysBypassGlobalHandlers(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T) Model
	}{
		{"sessions", func(t *testing.T) Model {
			m := newTestModel(t, ViewSessions)
			m.sessions = m.sessions.SetSessions([]models.Session{{ID: 1, Application: "quic"}}, nil)
			m.sessions, _ = m.sessions.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.sessions.IsFilterMode() {
				t.Fatal("precondition: sessions filter mode")
			}
			return m
		}},
		{"policies", func(t *testing.T) Model {
			m := newTestModel(t, ViewPolicies)
			m.policies = m.policies.SetPolicies([]models.SecurityRule{{Name: "rule-q"}}, nil)
			m.policies, _ = m.policies.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.policies.IsFilterMode() {
				t.Fatal("precondition: policies filter mode")
			}
			return m
		}},
		{"routes", func(t *testing.T) Model {
			m := newTestModel(t, ViewRoutes)
			m.routes = m.routes.SetRoutes([]models.RouteEntry{{Destination: "10.0.0.0/8"}}, nil)
			m.routes, _ = m.routes.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.routes.IsFilterMode() {
				t.Fatal("precondition: routes filter mode")
			}
			return m
		}},
		{"nat_policies", func(t *testing.T) Model {
			m := newTestModel(t, ViewNATPolicies)
			m.natPolicies = m.natPolicies.SetRules([]models.NATRule{{Name: "nat-q"}}, nil)
			m.natPolicies, _ = m.natPolicies.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.natPolicies.IsFilterMode() {
				t.Fatal("precondition: nat_policies filter mode")
			}
			return m
		}},
		{"interfaces", func(t *testing.T) Model {
			m := newTestModel(t, ViewInterfaces)
			m.interfaces = m.interfaces.SetInterfaces([]models.Interface{{Name: "eth0"}}, nil)
			m.interfaces, _ = m.interfaces.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.interfaces.IsFilterMode() {
				t.Fatal("precondition: interfaces filter mode")
			}
			return m
		}},
		{"ipsec_tunnels", func(t *testing.T) Model {
			m := newTestModel(t, ViewIPSecTunnels)
			m.ipsecTunnels = m.ipsecTunnels.SetTunnels([]models.IPSecTunnel{{Name: "vpn-q"}}, nil)
			m.ipsecTunnels, _ = m.ipsecTunnels.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.ipsecTunnels.IsFilterMode() {
				t.Fatal("precondition: ipsec_tunnels filter mode")
			}
			return m
		}},
		{"gp_users", func(t *testing.T) Model {
			m := newTestModel(t, ViewGPUsers)
			m.gpUsers = m.gpUsers.SetUsers([]models.GlobalProtectUser{{Username: "alice"}}, nil)
			m.gpUsers, _ = m.gpUsers.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.gpUsers.IsFilterMode() {
				t.Fatal("precondition: gp_users filter mode")
			}
			return m
		}},
		{"logs", func(t *testing.T) Model {
			m := newTestModel(t, ViewLogs)
			m.logs = m.logs.SetSystemLogs([]models.SystemLogEntry{{Description: "test-event"}}, nil)
			m.logs, _ = m.logs.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.logs.IsFilterMode() {
				t.Fatal("precondition: logs filter mode")
			}
			return m
		}},
		{"objects", func(t *testing.T) Model {
			m := newTestModel(t, ViewObjects)
			m.objects = m.objects.SetAddresses([]models.AddressObject{{Name: "host-q", Type: "ip-netmask", Value: "10.0.0.1/32"}}, nil)
			m.objects, _ = m.objects.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			if !m.objects.IsFilterMode() {
				t.Fatal("precondition: objects filter mode")
			}
			return m
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.setup(t)

			// 'q' must not quit. A nil cmd is fine (key typed into the
			// filter); a non-nil cmd must not be Quit.
			updated, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
			if cmd != nil {
				if _, isQuit := cmd().(tea.QuitMsg); isQuit {
					t.Fatal("'q' during filter mode quit the app")
				}
			}
			um, ok := updated.(Model)
			if !ok {
				t.Fatalf("Update returned %T, want Model", updated)
			}

			// '1' must not navigate away.
			updated, _ = um.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
			um = updated.(Model)
			if um.currentView != m.currentView {
				t.Errorf("'1' during filter mode navigated from %v to %v", m.currentView, um.currentView)
			}

			// ctrl+c must STILL quit (emergency exit).
			_, cmd = um.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
			if cmd == nil {
				t.Fatal("expected ctrl+c to produce a command in filter mode")
			}
			if _, isQuit := cmd().(tea.QuitMsg); !isQuit {
				t.Error("expected ctrl+c to quit even in filter mode")
			}
		})
	}
}
