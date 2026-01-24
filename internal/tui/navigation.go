package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jp2195/pyre/internal/tui/views"
)

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

	var cmd tea.Cmd

	switch item.ID {
	// Monitor group
	case "overview":
		m.currentView = ViewDashboard
		m.currentDashboard = views.DashboardMain
		cmd = m.fetchDashboardData()
	case "network":
		m.currentView = ViewDashboard
		m.currentDashboard = views.DashboardNetwork
		cmd = m.fetchNetworkDashboardData()
	case "security":
		m.currentView = ViewDashboard
		m.currentDashboard = views.DashboardSecurity
		cmd = m.fetchSecurityDashboardData()
	case "vpn":
		m.currentView = ViewDashboard
		m.currentDashboard = views.DashboardVPN
		cmd = m.fetchVPNDashboardData()

	// Analyze group
	case "policies":
		m.currentView = ViewPolicies
		m.policies = m.policies.SetLoading(true)
		cmd = m.fetchPolicies()
	case "nat":
		m.currentView = ViewNATPolicies
		m.natPolicies = m.natPolicies.SetLoading(true)
		cmd = m.fetchNATPolicies()
	case "sessions":
		m.currentView = ViewSessions
		m.sessions = m.sessions.SetLoading(true)
		cmd = m.fetchSessions()
	case "interfaces":
		m.currentView = ViewInterfaces
		m.interfaces = m.interfaces.SetLoading(true)
		cmd = m.fetchInterfaces()
	case "logs":
		m.currentView = ViewLogs
		m.logs = m.logs.SetLoading(true)
		cmd = m.fetchLogs()

	// Tools group
	case "troubleshoot":
		m.currentView = ViewTroubleshoot
		m.updateTroubleshootSSH()
		conn := m.session.GetActiveConnection()
		if conn != nil && conn.HasSSH() && conn.SSHClient == nil {
			cmd = m.connectSSH(conn)
		}
	case "config":
		m.currentView = ViewDashboard
		m.currentDashboard = views.DashboardConfig
		cmd = m.fetchConfigDashboardData()

	// Connections group
	case "picker":
		m.currentView = ViewPicker
	}

	return m, cmd
}

// syncNavbarToCurrentView syncs the navbar state to match the current view
func (m *Model) syncNavbarToCurrentView() {
	switch m.currentView {
	case ViewDashboard:
		switch m.currentDashboard {
		case views.DashboardMain:
			m.navbar = m.navbar.SetActiveByID("monitor", "overview")
		case views.DashboardNetwork:
			m.navbar = m.navbar.SetActiveByID("monitor", "network")
		case views.DashboardSecurity:
			m.navbar = m.navbar.SetActiveByID("monitor", "security")
		case views.DashboardVPN:
			m.navbar = m.navbar.SetActiveByID("monitor", "vpn")
		case views.DashboardConfig:
			m.navbar = m.navbar.SetActiveByID("tools", "config")
		}
	case ViewPolicies:
		m.navbar = m.navbar.SetActiveByID("analyze", "policies")
	case ViewNATPolicies:
		m.navbar = m.navbar.SetActiveByID("analyze", "nat")
	case ViewSessions:
		m.navbar = m.navbar.SetActiveByID("analyze", "sessions")
	case ViewInterfaces:
		m.navbar = m.navbar.SetActiveByID("analyze", "interfaces")
	case ViewLogs:
		m.navbar = m.navbar.SetActiveByID("analyze", "logs")
	case ViewTroubleshoot:
		m.navbar = m.navbar.SetActiveByID("tools", "troubleshoot")
	case ViewPicker:
		m.navbar = m.navbar.SetActiveByID("connections", "picker")
	}
}
