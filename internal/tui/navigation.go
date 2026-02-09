package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/tui/views"
)

// navTarget describes the view state and data fetch for a navigation item.
type navTarget struct {
	view      ViewState
	dashboard views.DashboardType // Only used when view == ViewDashboard
	hasData   func(m *Model) bool
	fetch     func(m *Model) tea.Cmd
}

// navTargets maps navbar item IDs to their navigation targets.
var navTargets = map[string]navTarget{
	// Monitor group (dashboard views)
	"overview": {
		view:      ViewDashboard,
		dashboard: views.DashboardMain,
		hasData:   func(m *Model) bool { return m.dashboard.HasData() },
		fetch:     func(m *Model) tea.Cmd { return m.fetchDashboardData() },
	},
	"network": {
		view:      ViewDashboard,
		dashboard: views.DashboardNetwork,
		hasData:   func(m *Model) bool { return m.networkDashboard.HasData() },
		fetch:     func(m *Model) tea.Cmd { return m.fetchNetworkDashboardData() },
	},
	"security": {
		view:      ViewDashboard,
		dashboard: views.DashboardSecurity,
		hasData:   func(m *Model) bool { return m.securityDashboard.HasData() },
		fetch:     func(m *Model) tea.Cmd { return m.fetchSecurityDashboardData() },
	},
	"vpn": {
		view:      ViewDashboard,
		dashboard: views.DashboardVPN,
		hasData:   func(m *Model) bool { return m.vpnDashboard.HasData() },
		fetch:     func(m *Model) tea.Cmd { return m.fetchVPNDashboardData() },
	},
	"config": {
		view:      ViewDashboard,
		dashboard: views.DashboardConfig,
		hasData:   func(m *Model) bool { return m.configDashboard.HasData() },
		fetch:     func(m *Model) tea.Cmd { return m.fetchConfigDashboardData() },
	},

	// Analyze group (detail views)
	"policies": {
		view:    ViewPolicies,
		hasData: func(m *Model) bool { return m.policies.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.policies = m.policies.SetLoading(true)
			return m.fetchPolicies()
		},
	},
	"nat": {
		view:    ViewNATPolicies,
		hasData: func(m *Model) bool { return m.natPolicies.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.natPolicies = m.natPolicies.SetLoading(true)
			return m.fetchNATPolicies()
		},
	},
	"sessions": {
		view:    ViewSessions,
		hasData: func(m *Model) bool { return m.sessions.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.sessions = m.sessions.SetLoading(true)
			return m.fetchSessions()
		},
	},
	"interfaces": {
		view:    ViewInterfaces,
		hasData: func(m *Model) bool { return m.interfaces.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.interfaces = m.interfaces.SetLoading(true)
			return m.fetchInterfaces()
		},
	},
	"routes": {
		view:    ViewRoutes,
		hasData: func(m *Model) bool { return m.routes.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.routes = m.routes.SetLoading(true)
			return m.fetchRoutesData()
		},
	},
	"ipsec": {
		view:    ViewIPSecTunnels,
		hasData: func(m *Model) bool { return m.ipsecTunnels.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.ipsecTunnels = m.ipsecTunnels.SetLoading(true)
			conn := m.session.GetActiveConnection()
			if conn != nil {
				return m.fetchIPSecTunnels(conn)
			}
			return nil
		},
	},
	"gpusers": {
		view:    ViewGPUsers,
		hasData: func(m *Model) bool { return m.gpUsers.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.gpUsers = m.gpUsers.SetLoading(true)
			conn := m.session.GetActiveConnection()
			if conn != nil {
				return m.fetchGlobalProtectUsers(conn)
			}
			return nil
		},
	},
	"logs": {
		view:    ViewLogs,
		hasData: func(m *Model) bool { return m.logs.HasData() },
		fetch: func(m *Model) tea.Cmd {
			m.logs = m.logs.SetLoading(true)
			return m.fetchLogs()
		},
	},
}

// navbarMapping maps (view, dashboard) pairs to (group, item) IDs for navbar sync.
type navbarID struct {
	group string
	item  string
}

var viewToNavbar = map[ViewState][]struct {
	dashboard views.DashboardType
	id        navbarID
}{
	ViewDashboard: {
		{views.DashboardMain, navbarID{"monitor", "overview"}},
		{views.DashboardNetwork, navbarID{"monitor", "network"}},
		{views.DashboardSecurity, navbarID{"monitor", "security"}},
		{views.DashboardVPN, navbarID{"monitor", "vpn"}},
		{views.DashboardConfig, navbarID{"tools", "config"}},
	},
	ViewPolicies:     {{0, navbarID{"analyze", "policies"}}},
	ViewNATPolicies:  {{0, navbarID{"analyze", "nat"}}},
	ViewSessions:     {{0, navbarID{"analyze", "sessions"}}},
	ViewInterfaces:   {{0, navbarID{"analyze", "interfaces"}}},
	ViewRoutes:       {{0, navbarID{"analyze", "routes"}}},
	ViewIPSecTunnels: {{0, navbarID{"analyze", "ipsec"}}},
	ViewGPUsers:      {{0, navbarID{"analyze", "gpusers"}}},
	ViewLogs:         {{0, navbarID{"analyze", "logs"}}},
}

// handleNavGroupKey handles number key presses for navigation
func (m Model) handleNavGroupKey(groupIndex int) (tea.Model, tea.Cmd) {
	currentGroup := m.navbar.ActiveGroupIndex()

	if currentGroup == groupIndex {
		// Already in this group - cycle to next item
		group := m.navbar.ActiveGroup()
		if group != nil {
			nextItem := (m.navbar.ActiveItemIndex() + 1) % len(group.Items)
			m.navbar = m.navbar.SetActiveItem(nextItem)
		}
	} else {
		// Switch to the new group (first item)
		m.navbar = m.navbar.SetActiveGroup(groupIndex)
	}

	// Navigate to the selected item
	return m.navigateToCurrentItem()
}

// navigateToCurrentItem switches to the view corresponding to the current navbar item
func (m Model) navigateToCurrentItem() (tea.Model, tea.Cmd) {
	item := m.navbar.ActiveItem()
	if item == nil {
		return m, nil
	}

	target, ok := navTargets[item.ID]
	if !ok {
		return m, nil
	}

	m.currentView = target.view
	if target.view == ViewDashboard {
		m.currentDashboard = target.dashboard
	}

	var cmd tea.Cmd
	if target.fetch != nil && !target.hasData(&m) {
		cmd = target.fetch(&m)
	}

	return m, cmd
}

// syncNavbarToCurrentView syncs the navbar state to match the current view
func (m *Model) syncNavbarToCurrentView() {
	entries, ok := viewToNavbar[m.currentView]
	if !ok {
		return
	}
	for _, e := range entries {
		if m.currentView != ViewDashboard || e.dashboard == m.currentDashboard {
			m.navbar = m.navbar.SetActiveByID(e.id.group, e.id.item)
			return
		}
	}
}
