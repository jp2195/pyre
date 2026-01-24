package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/tui/views"
)

func (m Model) handleLoginKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case msg.String() == "ctrl+c":
		return m, tea.Quit

	case msg.String() == "enter":
		if m.login.CanSubmit() {
			m.loading = true
			return m, m.doLogin()
		}

	case msg.String() == "tab":
		m.login = m.login.NextField()
		return m, nil
	}

	m.login, cmd = m.login.Update(msg)
	return m, cmd
}

func (m Model) handlePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pickerKeys := DefaultPickerKeyMap()

	switch {
	case key.Matches(msg, pickerKeys.Back):
		if m.session.GetActiveConnection() != nil {
			m.currentView = ViewDashboard
		}
		return m, nil

	case key.Matches(msg, pickerKeys.Select):
		selected := m.picker.Selected()
		if selected != "" {
			m.session.SetActiveFirewall(selected)
			m.currentView = ViewDashboard
			return m, m.fetchDashboardData()
		}
		return m, nil

	case key.Matches(msg, pickerKeys.Add):
		m.login = views.NewLoginModel(&auth.Credentials{})
		m.currentView = ViewLogin
		return m, nil
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

func (m Model) handleDevicePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	devicePickerKeys := DefaultDevicePickerKeyMap()

	switch {
	case key.Matches(msg, devicePickerKeys.Back):
		m.currentView = ViewDashboard
		return m, nil

	case key.Matches(msg, devicePickerKeys.Select):
		conn := m.session.GetActiveConnection()
		if conn != nil {
			device := m.devicePicker.SelectedDevice()
			conn.SetTarget(device)
			m.currentView = ViewDashboard
			return m, m.fetchCurrentDashboardData()
		}
		return m, nil

	case key.Matches(msg, devicePickerKeys.Refresh):
		conn := m.session.GetActiveConnection()
		if conn != nil {
			return m, m.fetchManagedDevices(conn)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.devicePicker, cmd = m.devicePicker.Update(msg)
	return m, cmd
}

func (m Model) handleCommandPaletteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = m.previousView
		m.commandPalette = m.commandPalette.Blur()
		return m, nil

	case "enter":
		if cmd := m.commandPalette.SelectedCommand(); cmd != nil && cmd.Action != nil {
			m.currentView = m.previousView
			m.commandPalette = m.commandPalette.Blur()
			return m, func() tea.Msg { return cmd.Action() }
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.commandPalette, cmd = m.commandPalette.Update(msg)
	return m, cmd
}

// buildCommandRegistry creates the command registry for the command palette
func (m Model) buildCommandRegistry() []views.Command {
	commands := []views.Command{
		// Monitor - dashboards for at-a-glance status
		{
			ID:          "monitor-overview",
			Label:       "Overview",
			Description: "System health, resources",
			Category:    "Monitor",
			Action:      func() tea.Msg { return SwitchDashboardMsg{views.DashboardMain} },
		},
		{
			ID:          "monitor-network",
			Label:       "Network",
			Description: "Interfaces, ARP, routing",
			Category:    "Monitor",
			Action:      func() tea.Msg { return SwitchDashboardMsg{views.DashboardNetwork} },
		},
		{
			ID:          "monitor-security",
			Label:       "Security",
			Description: "Threats, blocked apps",
			Category:    "Monitor",
			Action:      func() tea.Msg { return SwitchDashboardMsg{views.DashboardSecurity} },
		},
		{
			ID:          "monitor-vpn",
			Label:       "VPN",
			Description: "IPSec, GlobalProtect",
			Category:    "Monitor",
			Action:      func() tea.Msg { return SwitchDashboardMsg{views.DashboardVPN} },
		},

		// Analyze - detailed data views
		{
			ID:          "analyze-policies",
			Label:       "Security Policies",
			Description: "Security rules",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewPolicies} },
		},
		{
			ID:          "analyze-nat",
			Label:       "NAT Policies",
			Description: "NAT translation rules",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewNATPolicies} },
		},
		{
			ID:          "analyze-sessions",
			Label:       "Sessions",
			Description: "Active connections",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewSessions} },
		},
		{
			ID:          "analyze-interfaces",
			Label:       "Interfaces",
			Description: "Network interfaces",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewInterfaces} },
		},
		{
			ID:          "analyze-logs",
			Label:       "Logs",
			Description: "Traffic & threat logs",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewLogs} },
		},

		// Tools - diagnostic and config
		{
			ID:          "tools-troubleshoot",
			Label:       "Troubleshoot",
			Description: "Diagnostic runbooks",
			Category:    "Tools",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewTroubleshoot} },
		},
		{
			ID:          "tools-config",
			Label:       "Config",
			Description: "Rules, pending changes",
			Category:    "Tools",
			Action:      func() tea.Msg { return SwitchDashboardMsg{views.DashboardConfig} },
		},

		// Connections
		{
			ID:          "connect-new",
			Label:       "Connect to firewall...",
			Description: "Switch device",
			Category:    "Connections",
			Action:      func() tea.Msg { return ShowPickerMsg{} },
		},

		// Actions
		{
			ID:          "refresh",
			Label:       "Refresh",
			Description: "Reload current view",
			Category:    "Actions",
			Shortcut:    "r",
			Action:      func() tea.Msg { return RefreshMsg{} },
		},

		// System
		{
			ID:          "help",
			Label:       "Help",
			Description: "Keyboard shortcuts",
			Category:    "System",
			Shortcut:    "?",
			Action:      func() tea.Msg { return ShowHelpMsg{} },
		},
		{
			ID:          "quit",
			Label:       "Quit",
			Description: "Exit application",
			Category:    "System",
			Shortcut:    "q",
			Action:      func() tea.Msg { return tea.Quit() },
		},
	}

	// Add dynamic connection entries
	conn := m.session.GetActiveConnection()
	if conn != nil {
		commands = append(commands, views.Command{
			ID:          "conn-current",
			Label:       conn.Name + " (current)",
			Description: "Connected",
			Category:    "Connections",
		})
	}

	return commands
}

func (m Model) handleViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentView {
	case ViewDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case ViewPolicies:
		m.policies, cmd = m.policies.Update(msg)
	case ViewNATPolicies:
		m.natPolicies, cmd = m.natPolicies.Update(msg)
	case ViewSessions:
		m.sessions, cmd = m.sessions.Update(msg)
	case ViewInterfaces:
		m.interfaces, cmd = m.interfaces.Update(msg)
	case ViewTroubleshoot:
		// Handle 'R' to retry SSH connection
		if msg.String() == "R" && m.troubleshoot.Mode() == views.TroubleshootModeList {
			conn := m.session.GetActiveConnection()
			if conn != nil && conn.HasSSH() {
				conn.DisconnectSSH()
				m.troubleshoot = m.troubleshoot.SetSSHConnecting(true)
				m.troubleshoot = m.troubleshoot.SetSSHError(nil)
				return m, m.connectSSH(conn)
			}
		}
		// Handle Enter to run runbook
		if msg.String() == "enter" && m.troubleshoot.Mode() == views.TroubleshootModeList {
			runbook := m.troubleshoot.Selected()
			if runbook != nil {
				m.loading = true
				m.troubleshoot = m.troubleshoot.SetRunning(runbook)
				return m, m.runTroubleshoot(runbook)
			}
		}
		m.troubleshoot, cmd = m.troubleshoot.Update(msg)
	case ViewLogs:
		m.logs, cmd = m.logs.Update(msg)
	}

	return m, cmd
}
