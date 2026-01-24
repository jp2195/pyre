package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/troubleshoot"
	"github.com/jp2195/pyre/internal/tui/views"
)

type ViewState int

const (
	ViewLogin ViewState = iota
	ViewDashboard
	ViewPolicies
	ViewNATPolicies
	ViewSessions
	ViewInterfaces
	ViewTroubleshoot
	ViewLogs
	ViewPicker
	ViewDevicePicker
	ViewCommandPalette
)

type Model struct {
	ctx    context.Context
	cancel context.CancelFunc

	config  *config.Config
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
	troubleshoot      views.TroubleshootModel
	logs              views.LogsModel
	picker            views.PickerModel
	devicePicker      views.DevicePickerModel
	commandPalette    views.CommandPaletteModel
	previousView      ViewState // Track previous view for Esc to return

	// Troubleshooting
	tsRegistry *troubleshoot.Registry
	tsEngine   *troubleshoot.Engine
}

func NewModel(cfg *config.Config, creds *auth.Credentials) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	session := auth.NewSession(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		ctx:         ctx,
		cancel:      cancel,
		config:      cfg,
		session:     session,
		keys:        DefaultKeyMap(),
		help:        help.New(),
		spinner:     s,
		currentView: ViewLogin,
	}

	if creds.HasAPIKey() && creds.HasHost() {
		// Look up full firewall config by host to get SSH settings
		var fwConfig *config.FirewallConfig
		var connName string
		for name, fw := range cfg.Firewalls {
			if fw.Host == creds.Host {
				fwCopy := fw
				fwConfig = &fwCopy
				connName = name
				break
			}
		}
		if fwConfig == nil {
			fwConfig = &config.FirewallConfig{
				Host:     creds.Host,
				Insecure: creds.Insecure,
			}
			connName = "default"
		}
		session.AddConnection(connName, fwConfig, creds.APIKey)
		m.currentView = ViewDashboard
	}

	m.navbar = views.NewNavbarModel()
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
	m.troubleshoot = views.NewTroubleshootModel()
	m.logs = views.NewLogsModel()
	m.picker = views.NewPickerModel(session)
	m.devicePicker = views.NewDevicePickerModel()
	m.commandPalette = views.NewCommandPaletteModel()

	// Initialize troubleshooting registry
	m.tsRegistry = troubleshoot.NewRegistry()
	m.tsRegistry.LoadEmbedded()
	m.troubleshoot = m.troubleshoot.SetRunbooks(m.tsRegistry.List())

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}

	if m.currentView == ViewDashboard {
		cmds = append(cmds, m.fetchCurrentDashboardData())
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

		// Calculate content height (minus header and footer)
		// Header = main row + sub-tab row + border = 3 lines
		// Footer = 1 line
		contentHeight := msg.Height - 4

		m.navbar = m.navbar.SetSize(msg.Width)
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
		m.troubleshoot = m.troubleshoot.SetSize(msg.Width, contentHeight)
		m.logs = m.logs.SetSize(msg.Width, contentHeight)
		m.picker = m.picker.SetSize(msg.Width, contentHeight)
		m.devicePicker = m.devicePicker.SetSize(msg.Width, contentHeight)
		m.commandPalette = m.commandPalette.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		if m.currentView == ViewLogin {
			return m.handleLoginKeys(msg)
		}

		if m.currentView == ViewPicker {
			return m.handlePickerKeys(msg)
		}

		if m.currentView == ViewDevicePicker {
			return m.handleDevicePickerKeys(msg)
		}

		if m.currentView == ViewCommandPalette {
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
			// ":" opens command palette filtered to connections
			m.previousView = m.currentView
			m.currentView = ViewCommandPalette
			m.commandPalette = m.commandPalette.SetCommands(m.buildCommandRegistry())
			m.commandPalette = m.commandPalette.FocusWithFilter("Connections")
			return m, nil

		case key.Matches(msg, m.keys.DevicePicker):
			conn := m.session.GetActiveConnection()
			if conn != nil && conn.IsPanorama {
				m.currentView = ViewDevicePicker
				m.devicePicker = m.devicePicker.SetDevices(
					conn.ManagedDevices,
					conn.TargetSerial,
					conn.Name,
				)
				return m, nil
			}
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			// Set loading state for views that support it
			switch m.currentView {
			case ViewPolicies:
				m.policies = m.policies.SetLoading(true)
			case ViewNATPolicies:
				m.natPolicies = m.natPolicies.SetLoading(true)
			case ViewSessions:
				m.sessions = m.sessions.SetLoading(true)
			case ViewInterfaces:
				m.interfaces = m.interfaces.SetLoading(true)
			case ViewLogs:
				m.logs = m.logs.SetLoading(true)
			}
			return m, m.refreshCurrentView()

		// Navigation group keys
		case key.Matches(msg, m.keys.NavGroup1):
			return m.handleNavGroupKey(0)
		case key.Matches(msg, m.keys.NavGroup2):
			return m.handleNavGroupKey(1)
		case key.Matches(msg, m.keys.NavGroup3):
			return m.handleNavGroupKey(2)
		case key.Matches(msg, m.keys.NavGroup4):
			return m.handleNavGroupKey(3)

		// Tab cycles through items in current group
		case key.Matches(msg, m.keys.Tab):
			group := m.navbar.ActiveGroup()
			if group != nil && len(group.Items) > 0 {
				nextItem := (m.navbar.ActiveItemIndex() + 1) % len(group.Items)
				m.navbar = m.navbar.SetActiveItem(nextItem)
				return m.navigateToCurrentItem()
			}
		}

		return m.handleViewKeys(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case LoginSuccessMsg:
		m.loading = false
		// Look up full firewall config by host to get SSH settings
		var fwConfig *config.FirewallConfig
		var connName string
		loginHost := m.login.Host()
		for name, fw := range m.config.Firewalls {
			if fw.Host == loginHost {
				fwCopy := fw
				fwConfig = &fwCopy
				connName = name
				break
			}
		}
		if fwConfig == nil {
			fwConfig = &config.FirewallConfig{
				Host:     loginHost,
				Insecure: true,
			}
			connName = msg.Name
		}
		// Pass login credentials for SSH reuse
		conn := m.session.AddConnectionWithSSH(connName, fwConfig, msg.APIKey, msg.Username, msg.Password)
		m.currentView = ViewDashboard
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

	case TroubleshootResultMsg:
		m.loading = false
		m.troubleshoot = m.troubleshoot.SetResult(msg.Result, msg.Err)

	case TroubleshootStepMsg:
		m.troubleshoot = m.troubleshoot.UpdateStepProgress(msg.StepIndex, msg.Status, msg.Output)

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
		}

	case SSHConnectedMsg:
		m.troubleshoot = m.troubleshoot.SetSSHConnecting(false)
		m.troubleshoot = m.troubleshoot.SetSSHAvailable(true)
		m.troubleshoot = m.troubleshoot.SetSSHError(nil)

	case SSHErrorMsg:
		m.troubleshoot = m.troubleshoot.SetSSHConnecting(false)
		m.troubleshoot = m.troubleshoot.SetSSHAvailable(false)
		m.troubleshoot = m.troubleshoot.SetSSHError(msg.Err)

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
		m.currentView = msg.View
		m.syncNavbarToCurrentView() // Sync navbar state
		switch msg.View {
		case ViewDashboard:
			return m, m.fetchCurrentDashboardData()
		case ViewPolicies:
			m.policies = m.policies.SetLoading(true)
			return m, m.fetchPolicies()
		case ViewNATPolicies:
			m.natPolicies = m.natPolicies.SetLoading(true)
			return m, m.fetchNATPolicies()
		case ViewSessions:
			m.sessions = m.sessions.SetLoading(true)
			return m, m.fetchSessions()
		case ViewInterfaces:
			return m, m.fetchInterfaces()
		case ViewTroubleshoot:
			conn := m.session.GetActiveConnection()
			sshConfigured := conn != nil && conn.HasSSH()
			m.troubleshoot = m.troubleshoot.SetSSHConfigured(sshConfigured)
			if sshConfigured && !conn.SSHEnabled {
				m.troubleshoot = m.troubleshoot.SetSSHConnecting(true)
				return m, m.connectSSH(conn)
			}
			return m, m.updateTroubleshootSSH()
		case ViewLogs:
			m.logs = m.logs.SetLoading(true)
			return m, m.fetchLogs()
		}
		return m, nil

	case SwitchDashboardMsg:
		m.currentDashboard = msg.Dashboard
		m.currentView = ViewDashboard
		m.syncNavbarToCurrentView() // Sync navbar state
		return m, m.fetchCurrentDashboardData()

	case ShowPickerMsg:
		m.currentView = ViewPicker
		m.picker = m.picker.UpdateConnections(m.session)
		return m, nil

	case RefreshMsg:
		switch m.currentView {
		case ViewPolicies:
			m.policies = m.policies.SetLoading(true)
		case ViewNATPolicies:
			m.natPolicies = m.natPolicies.SetLoading(true)
		case ViewSessions:
			m.sessions = m.sessions.SetLoading(true)
		case ViewInterfaces:
			m.interfaces = m.interfaces.SetLoading(true)
		case ViewLogs:
			m.logs = m.logs.SetLoading(true)
		}
		return m, m.refreshCurrentView()

	case ShowHelpMsg:
		m.showHelp = !m.showHelp
		return m, nil

	case DiskUsageMsg:
		m.dashboard = m.dashboard.SetDiskUsage(msg.Disks, msg.Err)

	case EnvironmentalsMsg:
		m.dashboard = m.dashboard.SetEnvironmentals(msg.Environmentals, msg.Err)

	case CertificatesMsg:
		m.dashboard = m.dashboard.SetCertificates(msg.Certificates, msg.Err)

	case ARPTableMsg:
		m.networkDashboard = m.networkDashboard.SetARPTable(msg.Entries, msg.Err)

	case RoutingTableMsg:
		m.networkDashboard = m.networkDashboard.SetRoutingTable(msg.Routes, msg.Err)

	case IPSecTunnelsMsg:
		m.vpnDashboard = m.vpnDashboard.SetIPSecTunnels(msg.Tunnels, msg.Err)

	case GlobalProtectUsersMsg:
		m.vpnDashboard = m.vpnDashboard.SetGlobalProtectUsers(msg.Users, msg.Err)

	case PendingChangesMsg:
		m.configDashboard = m.configDashboard.SetPendingChanges(msg.Changes, msg.Err)

	case NATPoolMsg:
		m.dashboard = m.dashboard.SetNATPoolInfo(msg.Pools, msg.Err)

	case RefreshTickMsg:
		return m, m.refreshCurrentView()

	case ErrorMsg:
		m.err = msg.Err
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	switch m.currentView {
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

	case ViewTroubleshoot:
		content = m.troubleshoot.View()

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
