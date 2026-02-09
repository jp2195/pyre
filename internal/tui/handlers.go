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

	case msg.String() == " ":
		// Space toggles insecure checkbox when focused
		if m.login.FocusedField() == views.FieldInsecure {
			m.login = m.login.ToggleInsecure()
			return m, nil
		}

	case msg.String() == "tab":
		m.login = m.login.NextField()
		return m, nil

	case msg.String() == "shift+tab":
		m.login = m.login.PrevField()
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
		} else if m.config.HasConnections() {
			m.connectionHub = m.connectionHub.SetConnections(m.config, m.state)
			m.currentView = ViewConnectionHub
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
			if err := conn.SetTarget(device); err != nil {
				return m, m.setError(err)
			}
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
			ID:          "analyze-routes",
			Label:       "Routes",
			Description: "Routing table & neighbors",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewRoutes} },
		},
		{
			ID:          "analyze-ipsec",
			Label:       "IPSec Tunnels",
			Description: "IPSec VPN tunnels",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewIPSecTunnels} },
		},
		{
			ID:          "analyze-gpusers",
			Label:       "GlobalProtect Users",
			Description: "GP VPN users",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewGPUsers} },
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
			ID:          "tools-config",
			Label:       "Config",
			Description: "Rules, pending changes",
			Category:    "Tools",
			Action:      func() tea.Msg { return SwitchDashboardMsg{views.DashboardConfig} },
		},

		// Connections
		{
			ID:          "connections",
			Label:       "Connections",
			Description: "Manage & switch devices",
			Category:    "Connections",
			Shortcut:    ":",
			Action:      func() tea.Msg { return ShowConnectionHubMsg{} },
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

	return commands
}

func (m Model) handleConnectionHubKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hubKeys := DefaultConnectionHubKeyMap()

	// Handle delete confirmation
	if m.connectionHub.IsConfirming() {
		switch msg.String() {
		case "y", "Y":
			target := m.connectionHub.ConfirmTarget()
			m.connectionHub = m.connectionHub.HideDeleteConfirm()
			return m, func() tea.Msg { return ConnectionDeletedMsg{Host: target} }
		case "n", "N", "esc":
			m.connectionHub = m.connectionHub.HideDeleteConfirm()
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, hubKeys.Quit):
		return m, tea.Quit

	case key.Matches(msg, hubKeys.Connect):
		if entry := m.connectionHub.Selected(); entry != nil {
			connConfig, ok := m.config.GetConnection(entry.Host)
			if ok {
				return m, func() tea.Msg {
					return ConnectionSelectedMsg{
						Host:   entry.Host,
						Config: connConfig,
					}
				}
			}
		}
		return m, nil

	case key.Matches(msg, hubKeys.New):
		return m, func() tea.Msg {
			return ShowConnectionFormMsg{Mode: views.FormModeAdd}
		}

	case key.Matches(msg, hubKeys.Edit):
		if entry := m.connectionHub.Selected(); entry != nil {
			connConfig, ok := m.config.GetConnection(entry.Host)
			if ok {
				return m, func() tea.Msg {
					return ShowConnectionFormMsg{
						Mode:   views.FormModeEdit,
						Host:   entry.Host,
						Config: connConfig,
					}
				}
			}
		}
		return m, nil

	case key.Matches(msg, hubKeys.Delete):
		if entry := m.connectionHub.Selected(); entry != nil {
			m.connectionHub = m.connectionHub.ShowDeleteConfirm(entry.Host)
		}
		return m, nil

	case key.Matches(msg, hubKeys.QuickConnect):
		return m, func() tea.Msg {
			return ShowConnectionFormMsg{Mode: views.FormModeQuickConnect}
		}
	}

	var cmd tea.Cmd
	m.connectionHub, cmd = m.connectionHub.Update(msg)
	return m, cmd
}

func (m Model) handleConnectionFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	formKeys := DefaultConnectionFormKeyMap()

	switch {
	case key.Matches(msg, formKeys.Quit):
		return m, tea.Quit

	case key.Matches(msg, formKeys.Cancel):
		m.connectionHub = m.connectionHub.SetConnections(m.config, m.state)
		m.currentView = ViewConnectionHub
		return m, nil

	case key.Matches(msg, formKeys.Submit):
		// Submit the form
		if m.connectionForm.CanSubmit() {
			return m, func() tea.Msg {
				return ConnectionFormSubmitMsg{
					Host:         m.connectionForm.Host(),
					Config:       m.connectionForm.GetConfig(),
					SaveToConfig: m.connectionForm.SaveToConfig(),
					Mode:         m.connectionForm.Mode(),
				}
			}
		}
		return m, nil

	case key.Matches(msg, formKeys.Tab):
		m.connectionForm = m.connectionForm.NextField()
		return m, nil

	case key.Matches(msg, formKeys.ShiftTab):
		m.connectionForm = m.connectionForm.PrevField()
		return m, nil

	case key.Matches(msg, formKeys.Space):
		// Toggle checkboxes
		switch m.connectionForm.FocusedField() {
		case views.FormFieldType:
			m.connectionForm = m.connectionForm.ToggleType()
		case views.FormFieldInsecure:
			m.connectionForm = m.connectionForm.ToggleInsecure()
		case views.FormFieldSave:
			m.connectionForm = m.connectionForm.ToggleSave()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.connectionForm, cmd = m.connectionForm.Update(msg)
	return m, cmd
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
	case ViewRoutes:
		m.routes, cmd = m.routes.Update(msg)
	case ViewIPSecTunnels:
		m.ipsecTunnels, cmd = m.ipsecTunnels.Update(msg)
	case ViewGPUsers:
		m.gpUsers, cmd = m.gpUsers.Update(msg)
	case ViewLogs:
		m.logs, cmd = m.logs.Update(msg)
	}

	return m, cmd
}
