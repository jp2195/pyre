package tui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/tui/views"
)

// handleDataMsg routes async data messages to categorized sub-handlers.
func (m Model) handleDataMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case LoginSuccessMsg, LoginErrorMsg, PanoramaDetectedMsg, ManagedDevicesMsg:
		return m.handleAuthMsg(msg)

	case SystemInfoMsg, ResourcesMsg, SessionInfoMsg, HAStatusMsg,
		GlobalProtectMsg, LoggedInAdminsMsg, LicensesMsg, JobsMsg,
		DiskUsageMsg, EnvironmentalsMsg, CertificatesMsg, NATPoolMsg:
		return m.handleDashboardDataMsg(msg)

	case InterfacesMsg, ThreatSummaryMsg, PoliciesMsg, NATPoliciesMsg,
		SessionsMsg, SessionDetailMsg, SystemLogsMsg, TrafficLogsMsg,
		ThreatLogsMsg, ARPTableMsg, RoutingTableMsg, BGPNeighborsMsg,
		OSPFNeighborsMsg, IPSecTunnelsMsg, GlobalProtectUsersMsg,
		PendingChangesMsg:
		return m.handleViewDataMsg(msg)

	case DashboardSelectedMsg, SwitchViewMsg, SwitchDashboardMsg,
		ShowPickerMsg, ShowConnectionHubMsg, ShowConnectionFormMsg,
		ConnectionSelectedMsg, ConnectionFormSubmitMsg,
		ConnectionDeletedMsg, RefreshMsg, ShowHelpMsg, RefreshTickMsg:
		return m.handleNavigationMsg(msg)

	case ConfigSavedMsg, StateSavedMsg, ErrorMsg, ErrorDismissMsg:
		return m.handleStatusMsg(msg)

	case views.FetchDetailCmd:
		msg := msg.(views.FetchDetailCmd)
		return m, m.fetchSessionDetail(msg.SessionID)

	default:
		return m, nil
	}
}

// handleAuthMsg processes login, panorama detection, and device discovery messages.
func (m Model) handleAuthMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case LoginSuccessMsg:
		m.loading = false
		m.login = m.login.ClearPassword()

		var connConfig *config.ConnectionConfig
		host := msg.Host

		if m.selectedConnection != "" {
			host = m.selectedConnection
			connConfig = &m.selectedConnectionConfig
		} else {
			if conn, ok := m.config.Connections[host]; ok {
				connCopy := conn
				connConfig = &connCopy
			} else {
				connConfig = &config.ConnectionConfig{
					Insecure: m.login.Insecure(),
				}
			}
		}

		conn, err := m.session.AddConnection(host, connConfig, msg.APIKey)
		if err != nil {
			m.login = m.login.SetError(err)
			m.selectedConnection = ""
			m.selectedConnectionConfig = config.ConnectionConfig{}
			return m, nil
		}

		// Persist the API key to the OS keychain so subsequent launches
		// skip the password prompt. Best-effort: a keychain failure must
		// not block the user from completing login.
		if host != "" && msg.APIKey != "" {
			if err := config.SetAPIKey(host, msg.APIKey); err != nil {
				log.Printf("warning: failed to persist API key to keychain for %s: %v", host, err)
			}
		}

		m.currentView = ViewDashboard

		if m.state != nil && host != "" {
			m.state.UpdateConnection(host, msg.Username)
			cmds = append(cmds, m.saveState())
		}

		m.selectedConnection = ""
		m.selectedConnectionConfig = config.ConnectionConfig{}
		cmds = append(cmds, m.fetchCurrentDashboardData(), m.detectPanorama(conn))

	case LoginErrorMsg:
		m.loading = false
		m.login = m.login.SetError(msg.Err)

	case PanoramaDetectedMsg:
		conn := m.session.GetActiveConnection()
		if conn != nil {
			conn.IsPanorama = msg.IsPanorama
			if msg.IsPanorama {
				cmds = append(cmds, m.fetchManagedDevices(conn))
			}
		}

	case ManagedDevicesMsg:
		conn := m.session.GetActiveConnection()
		if conn != nil && msg.Err == nil {
			conn.ManagedDevices = msg.Devices
			if m.currentView == ViewDashboard && conn.IsPanorama && conn.TargetSerial == "" {
				m.currentView = ViewDevicePicker
				m.devicePicker = m.devicePicker.SetDevices(msg.Devices, conn.TargetSerial, conn.Host)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// handleDashboardDataMsg processes data messages for dashboard panels.
func (m Model) handleDashboardDataMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SystemInfoMsg:
		m.dashboard = m.dashboard.SetSystemInfo(msg.Info, msg.Err)
	case ResourcesMsg:
		m.dashboard = m.dashboard.SetResources(msg.Resources, msg.Err)
	case SessionInfoMsg:
		m.dashboard = m.dashboard.SetSessionInfo(msg.Info, msg.Err)
	case HAStatusMsg:
		m.dashboard = m.dashboard.SetHAStatus(msg.Status, msg.Err)
	case GlobalProtectMsg:
		m.dashboard = m.dashboard.SetGlobalProtectInfo(msg.Info, msg.Err)
	case LoggedInAdminsMsg:
		m.dashboard = m.dashboard.SetLoggedInAdmins(msg.Admins, msg.Err)
	case LicensesMsg:
		m.dashboard = m.dashboard.SetLicenses(msg.Licenses, msg.Err)
	case JobsMsg:
		m.dashboard = m.dashboard.SetJobs(msg.Jobs, msg.Err)
	case DiskUsageMsg:
		m.dashboard = m.dashboard.SetDiskUsage(msg.Disks, msg.Err)
	case EnvironmentalsMsg:
		m.dashboard = m.dashboard.SetEnvironmentals(msg.Environmentals, msg.Err) //nolint:misspell // "environmentals" is the PAN-OS XML API tag name
	case CertificatesMsg:
		m.dashboard = m.dashboard.SetCertificates(msg.Certificates, msg.Err)
	case NATPoolMsg:
		m.dashboard = m.dashboard.SetNATPoolInfo(msg.Pools, msg.Err)
	}

	return m, nil
}

// handleViewDataMsg processes data messages for detail views (policies, logs, network, etc.).
func (m Model) handleViewDataMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case InterfacesMsg:
		m.dashboard = m.dashboard.SetInterfaces(msg.Interfaces, msg.Err)
		m.interfaces = m.interfaces.SetInterfaces(msg.Interfaces, msg.Err)
		m.networkDashboard = m.networkDashboard.SetInterfaces(msg.Interfaces, msg.Err)
	case ThreatSummaryMsg:
		m.dashboard = m.dashboard.SetThreatSummary(msg.Summary, msg.Err)
		m.securityDashboard = m.securityDashboard.SetThreatSummary(msg.Summary, msg.Err)
	case PoliciesMsg:
		m.policies = m.policies.SetPolicies(msg.Policies, msg.Err)
		m.securityDashboard = m.securityDashboard.SetPolicies(msg.Policies, msg.Err)
		m.configDashboard = m.configDashboard.SetPolicies(msg.Policies, msg.Err)
	case NATPoliciesMsg:
		m.natPolicies = m.natPolicies.SetRules(msg.Rules, msg.Err)
	case SessionsMsg:
		m.sessions = m.sessions.SetSessions(msg.Sessions, msg.Err)
	case SessionDetailMsg:
		m.sessions = m.sessions.SetDetail(msg.Detail, msg.Err)
	case SystemLogsMsg:
		m.logs = m.logs.SetSystemLogs(msg.Logs, msg.Err)
	case TrafficLogsMsg:
		m.logs = m.logs.SetTrafficLogs(msg.Logs, msg.Err)
	case ThreatLogsMsg:
		m.logs = m.logs.SetThreatLogs(msg.Logs, msg.Err)
	case ARPTableMsg:
		m.networkDashboard = m.networkDashboard.SetARPTable(msg.Entries, msg.Err)
		if msg.Err == nil {
			m.interfaces = m.interfaces.SetARPTable(msg.Entries)
		}
	case RoutingTableMsg:
		m.networkDashboard = m.networkDashboard.SetRoutingTable(msg.Routes, msg.Err)
		m.routes = m.routes.SetRoutes(msg.Routes, msg.Err)
	case BGPNeighborsMsg:
		m.networkDashboard = m.networkDashboard.SetBGPNeighbors(msg.Neighbors, msg.Err)
		m.routes = m.routes.SetBGPNeighbors(msg.Neighbors, msg.Err)
	case OSPFNeighborsMsg:
		m.networkDashboard = m.networkDashboard.SetOSPFNeighbors(msg.Neighbors, msg.Err)
		m.routes = m.routes.SetOSPFNeighbors(msg.Neighbors, msg.Err)
	case IPSecTunnelsMsg:
		m.vpnDashboard = m.vpnDashboard.SetIPSecTunnels(msg.Tunnels, msg.Err)
		m.ipsecTunnels = m.ipsecTunnels.SetTunnels(msg.Tunnels, msg.Err)
	case GlobalProtectUsersMsg:
		m.vpnDashboard = m.vpnDashboard.SetGlobalProtectUsers(msg.Users, msg.Err)
		m.gpUsers = m.gpUsers.SetUsers(msg.Users, msg.Err)
	case PendingChangesMsg:
		m.configDashboard = m.configDashboard.SetPendingChanges(msg.Changes, msg.Err)
	}

	return m, nil
}

// handleNavigationMsg processes view transitions and UI navigation messages.
func (m Model) handleNavigationMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DashboardSelectedMsg:
		m.currentDashboard = msg.Dashboard
		m.currentView = ViewDashboard
		return m, m.fetchCurrentDashboardData()

	case SwitchViewMsg:
		return m.handleSwitchView(msg)

	case SwitchDashboardMsg:
		m.currentDashboard = msg.Dashboard
		m.currentView = ViewDashboard
		m.syncNavbarToCurrentView()
		return m, m.fetchCurrentDashboardData()

	case ShowPickerMsg:
		m.currentView = ViewPicker
		m.picker = m.picker.UpdateConnections(m.session)
		return m, nil

	case ShowConnectionHubMsg:
		m.connectionHub = m.connectionHub.SetConnections(m.config, m.state)
		m.currentView = ViewConnectionHub
		return m, nil

	case ShowConnectionFormMsg:
		return m.handleShowConnectionForm(msg)

	case ConnectionSelectedMsg:
		m.selectedConnection = msg.Host
		m.selectedConnectionConfig = msg.Config
		m.login = views.NewLoginModel(&auth.Credentials{
			Host:     msg.Host,
			Username: msg.Config.Username,
			Insecure: msg.Config.Insecure,
		})
		m.login = m.login.SetSize(m.width, m.height)
		m.currentView = ViewLogin
		return m, nil

	case ConnectionFormSubmitMsg:
		return m.handleConnectionFormSubmit(msg)

	case ConnectionDeletedMsg:
		return m.handleConnectionDeleted(msg)

	case RefreshMsg:
		return m.handleRefresh()

	case ShowHelpMsg:
		m.showHelp = !m.showHelp
		return m, nil

	case RefreshTickMsg:
		return m, m.refreshCurrentView()
	}

	return m, nil
}

// handleStatusMsg processes config/state save results and error lifecycle messages.
func (m Model) handleStatusMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ConfigSavedMsg:
		if msg.Err != nil {
			cmds = append(cmds, m.setError(msg.Err))
		}
	case StateSavedMsg:
		if msg.Err != nil {
			cmds = append(cmds, m.setError(msg.Err))
		}
	case ErrorMsg:
		cmds = append(cmds, m.setError(msg.Err))
	case ErrorDismissMsg:
		m.err = nil
	}

	return m, tea.Batch(cmds...)
}

// handleSwitchView switches to a new view and fetches data if needed.
func (m Model) handleSwitchView(msg SwitchViewMsg) (tea.Model, tea.Cmd) {
	m.currentView = msg.View
	m.syncNavbarToCurrentView()
	switch msg.View {
	case ViewDashboard:
		return m, m.fetchCurrentDashboardData()
	case ViewPolicies:
		if !m.policies.HasData() {
			m.policies = m.policies.SetLoading(true)
			return m, m.fetchPolicies()
		}
	case ViewNATPolicies:
		if !m.natPolicies.HasData() {
			m.natPolicies = m.natPolicies.SetLoading(true)
			return m, m.fetchNATPolicies()
		}
	case ViewSessions:
		if !m.sessions.HasData() {
			m.sessions = m.sessions.SetLoading(true)
			return m, m.fetchSessions()
		}
	case ViewInterfaces:
		if !m.interfaces.HasData() {
			m.interfaces = m.interfaces.SetLoading(true)
			conn := m.session.GetActiveConnection()
			if conn != nil {
				return m, tea.Batch(m.fetchInterfaces(), m.fetchARPTable(conn))
			}
			return m, m.fetchInterfaces()
		}
	case ViewRoutes:
		if !m.routes.HasData() {
			m.routes = m.routes.SetLoading(true)
			return m, m.fetchRoutesData()
		}
	case ViewIPSecTunnels:
		if !m.ipsecTunnels.HasData() {
			m.ipsecTunnels = m.ipsecTunnels.SetLoading(true)
			conn := m.session.GetActiveConnection()
			if conn != nil {
				return m, m.fetchIPSecTunnels(conn)
			}
		}
	case ViewGPUsers:
		if !m.gpUsers.HasData() {
			m.gpUsers = m.gpUsers.SetLoading(true)
			conn := m.session.GetActiveConnection()
			if conn != nil {
				return m, m.fetchGlobalProtectUsers(conn)
			}
		}
	case ViewLogs:
		if !m.logs.HasData() {
			m.logs = m.logs.SetLoading(true)
			return m, m.fetchLogs()
		}
	}
	return m, nil
}

// handleShowConnectionForm opens the connection form in the appropriate mode.
func (m Model) handleShowConnectionForm(msg ShowConnectionFormMsg) (tea.Model, tea.Cmd) {
	switch msg.Mode {
	case views.FormModeQuickConnect:
		m.connectionForm = views.NewQuickConnectForm()
	case views.FormModeAdd:
		m.connectionForm = views.NewAddConnectionForm()
	case views.FormModeEdit:
		m.connectionForm = views.NewEditConnectionForm(msg.Host, msg.Config)
	}
	m.connectionForm = m.connectionForm.SetSize(m.width, m.height)
	m.currentView = ViewConnectionForm
	return m, nil
}

// handleConnectionFormSubmit processes form submission, saving to config if requested.
func (m Model) handleConnectionFormSubmit(msg ConnectionFormSubmitMsg) (tea.Model, tea.Cmd) {
	var saveCmd tea.Cmd
	if msg.SaveToConfig {
		if msg.Mode == views.FormModeEdit {
			_ = m.config.UpdateConnection(msg.Host, msg.Config) //nolint:errcheck // UI flow continues regardless
		} else {
			_ = m.config.AddConnection(msg.Host, msg.Config) //nolint:errcheck // UI flow continues regardless
		}
		saveCmd = m.saveConfig()
		m.connectionHub = m.connectionHub.SetConnections(m.config, m.state)
	}

	m.selectedConnection = msg.Host
	m.selectedConnectionConfig = msg.Config
	m.login = views.NewLoginModel(&auth.Credentials{
		Host:     msg.Host,
		Username: msg.Config.Username,
		Insecure: msg.Config.Insecure,
	})
	m.login = m.login.SetSize(m.width, m.height)
	m.currentView = ViewLogin
	return m, saveCmd
}

// handleConnectionDeleted removes a connection from config and state.
func (m Model) handleConnectionDeleted(msg ConnectionDeletedMsg) (tea.Model, tea.Cmd) {
	_ = m.config.DeleteConnection(msg.Host) //nolint:errcheck // UI flow continues regardless
	var saveCmds []tea.Cmd
	saveCmds = append(saveCmds, m.saveConfig())
	if m.state != nil {
		m.state.DeleteConnection(msg.Host)
		saveCmds = append(saveCmds, m.saveState())
	}
	m.connectionHub = m.connectionHub.SetConnections(m.config, m.state)
	return m, tea.Batch(saveCmds...)
}
