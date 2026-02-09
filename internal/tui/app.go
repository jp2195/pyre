package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/tui/views"
)

const errorDismissTimeout = 5 * time.Second

type ViewState int

const (
	ViewConnectionHub  ViewState = iota // Entry point when config has connections
	ViewConnectionForm                  // Add/Edit/QuickConnect form
	ViewLogin
	ViewDashboard
	ViewPolicies
	ViewNATPolicies
	ViewSessions
	ViewInterfaces
	ViewRoutes
	ViewIPSecTunnels
	ViewGPUsers
	ViewLogs
	ViewPicker
	ViewDevicePicker
	ViewCommandPalette
)

type Model struct {
	ctx    context.Context
	cancel context.CancelFunc

	config  *config.Config
	state   *config.State
	session *auth.Session
	keys    KeyMap
	help    help.Model
	spinner spinner.Model

	width  int
	height int

	currentView      ViewState
	currentDashboard views.DashboardType
	showHelp         bool
	loading          bool
	err              error

	navbar            views.NavbarModel
	connectionHub     views.ConnectionHubModel
	connectionForm    views.ConnectionFormModel
	login             views.LoginModel
	dashboard         views.DashboardModel
	networkDashboard  views.NetworkDashboardModel
	securityDashboard views.SecurityDashboardModel
	vpnDashboard      views.VPNDashboardModel
	configDashboard   views.ConfigDashboardModel
	policies          views.PoliciesModel
	natPolicies       views.NATPoliciesModel
	sessions          views.SessionsModel
	interfaces        views.InterfacesModel
	routes            views.RoutesModel
	ipsecTunnels      views.IPSecTunnelsModel
	gpUsers           views.GPUsersModel
	logs              views.LogsModel
	picker            views.PickerModel
	devicePicker      views.DevicePickerModel
	commandPalette    views.CommandPaletteModel
	previousView      ViewState // Track previous view for Esc to return

	// selectedConnection stores the connection selected from hub before login
	selectedConnection       string
	selectedConnectionConfig config.ConnectionConfig
}

func NewModel(cfg *config.Config, state *config.State, creds *auth.Credentials, startView ViewState) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	session := auth.NewSession(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		ctx:         ctx,
		cancel:      cancel,
		config:      cfg,
		state:       state,
		session:     session,
		keys:        DefaultKeyMap(),
		help:        help.New(),
		spinner:     s,
		currentView: startView,
	}

	// If we have full credentials (API key + host), go straight to dashboard
	if creds.HasAPIKey() && creds.HasHost() {
		// Look up full connection config by host
		var connConfig *config.ConnectionConfig
		if conn, ok := cfg.Connections[creds.Host]; ok {
			connCopy := conn
			connConfig = &connCopy
		} else {
			connConfig = &config.ConnectionConfig{
				Insecure: creds.Insecure,
			}
		}
		session.AddConnection(creds.Host, connConfig, creds.APIKey)
		m.currentView = ViewDashboard
	}

	// Initialize all view models
	m.navbar = views.NewNavbarModel()
	m.connectionHub = views.NewConnectionHubModel().SetConnections(cfg, state)
	m.connectionForm = views.NewQuickConnectForm()
	m.login = views.NewLoginModel(creds)
	m.dashboard = views.NewDashboardModel()
	m.networkDashboard = views.NewNetworkDashboardModel()
	m.securityDashboard = views.NewSecurityDashboardModel()
	m.vpnDashboard = views.NewVPNDashboardModel()
	m.configDashboard = views.NewConfigDashboardModel()
	m.policies = views.NewPoliciesModel()
	m.natPolicies = views.NewNATPoliciesModel()
	m.sessions = views.NewSessionsModel()
	m.interfaces = views.NewInterfacesModel()
	m.routes = views.NewRoutesModel()
	m.ipsecTunnels = views.NewIPSecTunnelsModel()
	m.gpUsers = views.NewGPUsersModel()
	m.logs = views.NewLogsModel()
	m.picker = views.NewPickerModel(session)
	m.devicePicker = views.NewDevicePickerModel()
	m.commandPalette = views.NewCommandPaletteModel()

	return m
}

// setError sets an error on the model and returns a command to auto-dismiss it.
func (m *Model) setError(err error) tea.Cmd {
	m.err = err
	return tea.Tick(errorDismissTimeout, func(time.Time) tea.Msg {
		return ErrorDismissMsg{}
	})
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}

	if m.currentView == ViewDashboard {
		cmds = append(cmds, m.fetchCurrentDashboardData())
		// Also detect panorama when starting with API key credentials
		if conn := m.session.GetActiveConnection(); conn != nil {
			cmds = append(cmds, m.detectPanorama(conn))
		}
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	default:
		return m.handleDataMsg(msg)
	}
}

// handleWindowSize propagates resize events to all sub-views.
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.help.Width = msg.Width

	// Calculate content height (minus header and footer)
	// Header = main row + sub-tab row + border = 3 lines
	// Footer = 1 line
	contentHeight := msg.Height - 4

	m.navbar = m.navbar.SetSize(msg.Width)
	m.connectionHub = m.connectionHub.SetSize(msg.Width, msg.Height)
	m.connectionForm = m.connectionForm.SetSize(msg.Width, msg.Height)
	m.login = m.login.SetSize(msg.Width, msg.Height)
	m.dashboard = m.dashboard.SetSize(msg.Width, contentHeight)
	m.networkDashboard = m.networkDashboard.SetSize(msg.Width, contentHeight)
	m.securityDashboard = m.securityDashboard.SetSize(msg.Width, contentHeight)
	m.vpnDashboard = m.vpnDashboard.SetSize(msg.Width, contentHeight)
	m.configDashboard = m.configDashboard.SetSize(msg.Width, contentHeight)
	m.policies = m.policies.SetSize(msg.Width, contentHeight)
	m.natPolicies = m.natPolicies.SetSize(msg.Width, contentHeight)
	m.sessions = m.sessions.SetSize(msg.Width, contentHeight)
	m.interfaces = m.interfaces.SetSize(msg.Width, contentHeight)
	m.routes = m.routes.SetSize(msg.Width, contentHeight)
	m.ipsecTunnels = m.ipsecTunnels.SetSize(msg.Width, contentHeight)
	m.gpUsers = m.gpUsers.SetSize(msg.Width, contentHeight)
	m.logs = m.logs.SetSize(msg.Width, contentHeight)
	m.picker = m.picker.SetSize(msg.Width, contentHeight)
	m.devicePicker = m.devicePicker.SetSize(msg.Width, contentHeight)
	m.commandPalette = m.commandPalette.SetSize(msg.Width, msg.Height)

	return m, nil
}

// handleKeyMsg routes keyboard input to the appropriate view or global handler.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// Delegate to view-specific key handlers
	switch m.currentView {
	case ViewConnectionHub:
		return m.handleConnectionHubKeys(msg)
	case ViewConnectionForm:
		return m.handleConnectionFormKeys(msg)
	case ViewLogin:
		return m.handleLoginKeys(msg)
	case ViewPicker:
		return m.handlePickerKeys(msg)
	case ViewDevicePicker:
		return m.handleDevicePickerKeys(msg)
	case ViewCommandPalette:
		return m.handleCommandPaletteKeys(msg)
	}

	// If logs view is in filter mode, pass keys through to the view
	// (except ctrl+c for emergency quit)
	if m.currentView == ViewLogs && m.logs.IsFilterMode() {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleViewKeys(msg)
	}

	// Global key bindings (active when in main views)
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.OpenPalette):
		m.previousView = m.currentView
		m.currentView = ViewCommandPalette
		m.commandPalette = m.commandPalette.SetCommands(m.buildCommandRegistry())
		m.commandPalette = m.commandPalette.Focus()
		return m, nil

	case msg.String() == ":":
		// ":" opens connection hub directly
		m.connectionHub = m.connectionHub.SetConnections(m.config, m.state)
		m.currentView = ViewConnectionHub
		return m, nil

	case key.Matches(msg, m.keys.DevicePicker):
		conn := m.session.GetActiveConnection()
		if conn != nil && conn.IsPanorama {
			m.currentView = ViewDevicePicker
			m.devicePicker = m.devicePicker.SetDevices(
				conn.ManagedDevices,
				conn.TargetSerial,
				conn.Host,
			)
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m.handleRefresh()

	// Navigation group keys
	case key.Matches(msg, m.keys.NavGroup1):
		return m.handleNavGroupKey(0)
	case key.Matches(msg, m.keys.NavGroup2):
		return m.handleNavGroupKey(1)
	case key.Matches(msg, m.keys.NavGroup3):
		return m.handleNavGroupKey(2)

	// Tab cycles forward through items in current group
	case key.Matches(msg, m.keys.Tab):
		group := m.navbar.ActiveGroup()
		if group != nil && len(group.Items) > 0 {
			nextItem := (m.navbar.ActiveItemIndex() + 1) % len(group.Items)
			m.navbar = m.navbar.SetActiveItem(nextItem)
			return m.navigateToCurrentItem()
		}

	// Shift+Tab cycles backward through items in current group
	case key.Matches(msg, m.keys.ShiftTab):
		group := m.navbar.ActiveGroup()
		if group != nil && len(group.Items) > 0 {
			prevItem := m.navbar.ActiveItemIndex() - 1
			if prevItem < 0 {
				prevItem = len(group.Items) - 1 // Wrap to end
			}
			m.navbar = m.navbar.SetActiveItem(prevItem)
			return m.navigateToCurrentItem()
		}
	}

	return m.handleViewKeys(msg)
}

// handleRefresh sets loading state and refreshes the current view.
func (m Model) handleRefresh() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewPolicies:
		m.policies = m.policies.SetLoading(true)
	case ViewNATPolicies:
		m.natPolicies = m.natPolicies.SetLoading(true)
	case ViewSessions:
		m.sessions = m.sessions.SetLoading(true)
	case ViewInterfaces:
		m.interfaces = m.interfaces.SetLoading(true)
	case ViewRoutes:
		m.routes = m.routes.SetLoading(true)
	case ViewIPSecTunnels:
		m.ipsecTunnels = m.ipsecTunnels.SetLoading(true)
	case ViewGPUsers:
		m.gpUsers = m.gpUsers.SetLoading(true)
	case ViewLogs:
		m.logs = m.logs.SetLoading(true)
	}
	return m, m.refreshCurrentView()
}

// handleSpinnerTick updates the spinner and shares its frame with all views.
func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	// Share spinner frame with all views
	frame := m.spinner.View()
	m.policies.SpinnerFrame = frame
	m.natPolicies.SpinnerFrame = frame
	m.sessions.SpinnerFrame = frame
	m.interfaces.SpinnerFrame = frame
	m.routes.SpinnerFrame = frame
	m.ipsecTunnels.SpinnerFrame = frame
	m.gpUsers.SpinnerFrame = frame
	m.logs.SpinnerFrame = frame

	// Share spinner frame with dashboard views
	m.dashboard = m.dashboard.SetSpinnerFrame(frame)
	m.networkDashboard = m.networkDashboard.SetSpinnerFrame(frame)
	m.securityDashboard = m.securityDashboard.SetSpinnerFrame(frame)
	m.vpnDashboard = m.vpnDashboard.SetSpinnerFrame(frame)
	m.configDashboard = m.configDashboard.SetSpinnerFrame(frame)

	return m, cmd
}

// handleDataMsg processes async data messages (API responses, navigation, connections).
func (m Model) handleDataMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case LoginSuccessMsg:
		m.loading = false
		// Clear password from login model immediately after success
		m.login = m.login.ClearPassword()

		// Use selected connection if available, otherwise look up by host
		var connConfig *config.ConnectionConfig
		host := msg.Host

		if m.selectedConnection != "" {
			// Use the connection selected from hub
			host = m.selectedConnection
			connConfig = &m.selectedConnectionConfig
		} else {
			// Look up full connection config by host
			if conn, ok := m.config.Connections[host]; ok {
				connCopy := conn
				connConfig = &connCopy
			} else {
				connConfig = &config.ConnectionConfig{
					Insecure: m.login.Insecure(),
				}
			}
		}

		conn := m.session.AddConnection(host, connConfig, msg.APIKey)
		m.currentView = ViewDashboard

		// Update state with connection info
		if m.state != nil && host != "" {
			m.state.UpdateConnection(host, msg.Username)
			cmds = append(cmds, m.saveState())
		}

		// Clear selected connection
		m.selectedConnection = ""
		m.selectedConnectionConfig = config.ConnectionConfig{}

		cmds = append(cmds, m.fetchCurrentDashboardData(), m.detectPanorama(conn))

	case LoginErrorMsg:
		m.loading = false
		m.login = m.login.SetError(msg.Err)

	case SystemInfoMsg:
		m.dashboard = m.dashboard.SetSystemInfo(msg.Info, msg.Err)

	case ResourcesMsg:
		m.dashboard = m.dashboard.SetResources(msg.Resources, msg.Err)

	case SessionInfoMsg:
		m.dashboard = m.dashboard.SetSessionInfo(msg.Info, msg.Err)

	case HAStatusMsg:
		m.dashboard = m.dashboard.SetHAStatus(msg.Status, msg.Err)

	case InterfacesMsg:
		m.dashboard = m.dashboard.SetInterfaces(msg.Interfaces, msg.Err)
		m.interfaces = m.interfaces.SetInterfaces(msg.Interfaces, msg.Err)
		m.networkDashboard = m.networkDashboard.SetInterfaces(msg.Interfaces, msg.Err)

	case ThreatSummaryMsg:
		m.dashboard = m.dashboard.SetThreatSummary(msg.Summary, msg.Err)
		m.securityDashboard = m.securityDashboard.SetThreatSummary(msg.Summary, msg.Err)

	case GlobalProtectMsg:
		m.dashboard = m.dashboard.SetGlobalProtectInfo(msg.Info, msg.Err)

	case LoggedInAdminsMsg:
		m.dashboard = m.dashboard.SetLoggedInAdmins(msg.Admins, msg.Err)

	case LicensesMsg:
		m.dashboard = m.dashboard.SetLicenses(msg.Licenses, msg.Err)

	case JobsMsg:
		m.dashboard = m.dashboard.SetJobs(msg.Jobs, msg.Err)

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

	case views.FetchDetailCmd:
		return m, m.fetchSessionDetail(msg.SessionID)

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
			// If we're on dashboard after initial panorama connection (no target selected yet),
			// automatically show the device picker so user can select a target device
			if m.currentView == ViewDashboard && conn.IsPanorama && conn.TargetSerial == "" {
				m.currentView = ViewDevicePicker
				m.devicePicker = m.devicePicker.SetDevices(msg.Devices, conn.TargetSerial, conn.Host)
			}
		}

	case SystemLogsMsg:
		m.logs = m.logs.SetSystemLogs(msg.Logs, msg.Err)

	case TrafficLogsMsg:
		m.logs = m.logs.SetTrafficLogs(msg.Logs, msg.Err)

	case ThreatLogsMsg:
		m.logs = m.logs.SetThreatLogs(msg.Logs, msg.Err)

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
		// Store selected connection info and transition to login
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

	case DiskUsageMsg:
		m.dashboard = m.dashboard.SetDiskUsage(msg.Disks, msg.Err)

	case EnvironmentalsMsg:
		m.dashboard = m.dashboard.SetEnvironmentals(msg.Environmentals, msg.Err) //nolint:misspell // "environmentals" is the PAN-OS XML API tag name

	case CertificatesMsg:
		m.dashboard = m.dashboard.SetCertificates(msg.Certificates, msg.Err)

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

	case NATPoolMsg:
		m.dashboard = m.dashboard.SetNATPoolInfo(msg.Pools, msg.Err)

	case RefreshTickMsg:
		return m, m.refreshCurrentView()

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

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	switch m.currentView {
	case ViewConnectionHub:
		return m.connectionHub.View()

	case ViewConnectionForm:
		return m.connectionForm.View()

	case ViewLogin:
		return m.login.View()

	case ViewPicker:
		content = m.picker.View()

	case ViewDevicePicker:
		content = m.devicePicker.View()

	case ViewCommandPalette:
		// Command palette is rendered as an overlay
		return m.commandPalette.View()

	case ViewDashboard:
		switch m.currentDashboard {
		case views.DashboardNetwork:
			content = m.networkDashboard.View()
		case views.DashboardSecurity:
			content = m.securityDashboard.View()
		case views.DashboardVPN:
			content = m.vpnDashboard.View()
		case views.DashboardConfig:
			content = m.configDashboard.View()
		default:
			content = m.dashboard.View()
		}

	case ViewPolicies:
		content = m.policies.View()

	case ViewNATPolicies:
		content = m.natPolicies.View()

	case ViewSessions:
		content = m.sessions.View()

	case ViewInterfaces:
		content = m.interfaces.View()

	case ViewRoutes:
		content = m.routes.View()

	case ViewIPSecTunnels:
		content = m.ipsecTunnels.View()

	case ViewGPUsers:
		content = m.gpUsers.View()

	case ViewLogs:
		content = m.logs.View()
	}

	if m.showHelp {
		content = m.renderHelp()
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}
