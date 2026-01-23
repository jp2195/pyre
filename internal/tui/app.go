package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joshuamontgomery/pyre/internal/api"
	"github.com/joshuamontgomery/pyre/internal/auth"
	"github.com/joshuamontgomery/pyre/internal/config"
	"github.com/joshuamontgomery/pyre/internal/models"
	"github.com/joshuamontgomery/pyre/internal/ssh"
	"github.com/joshuamontgomery/pyre/internal/troubleshoot"
	"github.com/joshuamontgomery/pyre/internal/tui/views"
)

type ViewState int

const (
	ViewLogin ViewState = iota
	ViewDashboard
	ViewPolicies
	ViewSessions
	ViewInterfaces
	ViewTroubleshoot
	ViewLogs
	ViewPicker
	ViewDevicePicker
	ViewCommandPalette
)

type Model struct {
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

	login             views.LoginModel
	dashboard         views.DashboardModel
	networkDashboard  views.NetworkDashboardModel
	securityDashboard views.SecurityDashboardModel
	vpnDashboard      views.VPNDashboardModel
	configDashboard   views.ConfigDashboardModel
	policies          views.PoliciesModel
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

type SystemInfoMsg struct {
	Info *models.SystemInfo
	Err  error
}

type ResourcesMsg struct {
	Resources *models.Resources
	Err       error
}

type SessionInfoMsg struct {
	Info *models.SessionInfo
	Err  error
}

type SessionsMsg struct {
	Sessions []models.Session
	Err      error
}

type PoliciesMsg struct {
	Policies []models.SecurityRule
	Err      error
}

type HAStatusMsg struct {
	Status *models.HAStatus
	Err    error
}

type InterfacesMsg struct {
	Interfaces []models.Interface
	Err        error
}

type ThreatSummaryMsg struct {
	Summary *models.ThreatSummary
	Err     error
}

type GlobalProtectMsg struct {
	Info *models.GlobalProtectInfo
	Err  error
}

type LoggedInAdminsMsg struct {
	Admins []models.LoggedInAdmin
	Err    error
}

type LicensesMsg struct {
	Licenses []models.LicenseInfo
	Err      error
}

type JobsMsg struct {
	Jobs []models.Job
	Err  error
}

type LoginSuccessMsg struct {
	Name     string
	APIKey   string
	Username string
	Password string
}

type LoginErrorMsg struct {
	Err error
}

type RefreshTickMsg struct{}

type ErrorMsg struct {
	Err error
}

type TroubleshootResultMsg struct {
	Result *troubleshoot.RunbookResult
	Err    error
}

type TroubleshootStepMsg struct {
	StepIndex int
	Status    troubleshoot.StepStatus
	Output    string
}

type ManagedDevicesMsg struct {
	Devices []models.ManagedDevice
	Err     error
}

type PanoramaDetectedMsg struct {
	IsPanorama bool
	Model      string
}

type SSHConnectedMsg struct {
	ConnectionName string
}

type SSHErrorMsg struct {
	ConnectionName string
	Err            error
}

type SystemLogsMsg struct {
	Logs []models.SystemLogEntry
	Err  error
}

type TrafficLogsMsg struct {
	Logs []models.TrafficLogEntry
	Err  error
}

type ThreatLogsMsg struct {
	Logs []models.ThreatLogEntry
	Err  error
}

type DashboardSelectedMsg struct {
	Dashboard views.DashboardType
}

// SwitchViewMsg requests switching to a specific view
type SwitchViewMsg struct {
	View ViewState
}

// SwitchDashboardMsg requests switching to a specific dashboard
type SwitchDashboardMsg struct {
	Dashboard views.DashboardType
}

// ShowPickerMsg requests showing the firewall picker
type ShowPickerMsg struct{}

// RefreshMsg requests a refresh of the current view
type RefreshMsg struct{}

// ShowHelpMsg requests showing the help overlay
type ShowHelpMsg struct{}

type DiskUsageMsg struct {
	Disks []models.DiskUsage
	Err   error
}

type EnvironmentalsMsg struct {
	Environmentals []models.Environmental
	Err            error
}

type CertificatesMsg struct {
	Certificates []models.Certificate
	Err          error
}

type ARPTableMsg struct {
	Entries []models.ARPEntry
	Err     error
}

type RoutingTableMsg struct {
	Routes []models.RouteEntry
	Err    error
}

type IPSecTunnelsMsg struct {
	Tunnels []models.IPSecTunnel
	Err     error
}

type GlobalProtectUsersMsg struct {
	Users []models.GlobalProtectUser
	Err   error
}

type PendingChangesMsg struct {
	Changes []models.PendingChange
	Err     error
}

func NewModel(cfg *config.Config, creds *auth.Credentials) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	session := auth.NewSession(cfg)

	m := Model{
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

	m.login = views.NewLoginModel(creds)
	m.dashboard = views.NewDashboardModel()
	m.networkDashboard = views.NewNetworkDashboardModel()
	m.securityDashboard = views.NewSecurityDashboardModel()
	m.vpnDashboard = views.NewVPNDashboardModel()
	m.configDashboard = views.NewConfigDashboardModel()
	m.policies = views.NewPoliciesModel()
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

		m.login = m.login.SetSize(msg.Width, msg.Height)
		m.dashboard = m.dashboard.SetSize(msg.Width, msg.Height-4)
		m.networkDashboard = m.networkDashboard.SetSize(msg.Width, msg.Height-4)
		m.securityDashboard = m.securityDashboard.SetSize(msg.Width, msg.Height-4)
		m.vpnDashboard = m.vpnDashboard.SetSize(msg.Width, msg.Height-4)
		m.configDashboard = m.configDashboard.SetSize(msg.Width, msg.Height-4)
		m.policies = m.policies.SetSize(msg.Width, msg.Height-4)
		m.sessions = m.sessions.SetSize(msg.Width, msg.Height-4)
		m.interfaces = m.interfaces.SetSize(msg.Width, msg.Height-4)
		m.troubleshoot = m.troubleshoot.SetSize(msg.Width, msg.Height-4)
		m.logs = m.logs.SetSize(msg.Width, msg.Height-4)
		m.picker = m.picker.SetSize(msg.Width, msg.Height-4)
		m.devicePicker = m.devicePicker.SetSize(msg.Width, msg.Height-4)
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
			case ViewSessions:
				m.sessions = m.sessions.SetLoading(true)
			case ViewInterfaces:
				m.interfaces = m.interfaces.SetLoading(true)
			case ViewLogs:
				m.logs = m.logs.SetLoading(true)
			}
			return m, m.refreshCurrentView()
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
		switch msg.View {
		case ViewDashboard:
			return m, m.fetchCurrentDashboardData()
		case ViewPolicies:
			m.policies = m.policies.SetLoading(true)
			return m, m.fetchPolicies()
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
		return m, m.fetchCurrentDashboardData()

	case ShowPickerMsg:
		m.currentView = ViewPicker
		m.picker = m.picker.UpdateConnections(m.session)
		return m, nil

	case RefreshMsg:
		switch m.currentView {
		case ViewPolicies:
			m.policies = m.policies.SetLoading(true)
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

func (m Model) renderHeader() string {
	title := HeaderStyle.Render(" pyre ")

	conn := m.session.GetActiveConnection()
	var status string
	if conn != nil {
		statusText := fmt.Sprintf("● %s (%s)", conn.Name, conn.Config.Host)
		// Show current target for Panorama connections
		if conn.IsPanorama {
			if target := conn.GetTargetDevice(); target != nil {
				hostname := target.Hostname
				if hostname == "" {
					hostname = target.Serial
				}
				statusText += " → " + hostname
			}
		}
		status = ConnectedStyle.Render(statusText)
	} else {
		status = DisconnectedStyle.Render("● disconnected")
	}

	// Show current view name on the right
	viewName := m.currentViewName()
	viewLabel := ActiveTabStyle.Render(viewName)

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", status)
	padding := m.width - lipgloss.Width(left) - lipgloss.Width(viewLabel) - 2
	if padding < 0 {
		padding = 0
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		left,
		strings.Repeat(" ", padding),
		viewLabel,
	)
}

func (m Model) currentViewName() string {
	switch m.currentView {
	case ViewDashboard:
		// Use friendly names without "Dashboard" suffix
		switch m.currentDashboard {
		case views.DashboardMain:
			return "Overview"
		case views.DashboardNetwork:
			return "Network"
		case views.DashboardSecurity:
			return "Security"
		case views.DashboardVPN:
			return "VPN"
		case views.DashboardConfig:
			return "Config"
		default:
			return "Overview"
		}
	case ViewPolicies:
		return "Policies"
	case ViewSessions:
		return "Sessions"
	case ViewInterfaces:
		return "Interfaces"
	case ViewTroubleshoot:
		return "Troubleshoot"
	case ViewLogs:
		return "Logs"
	case ViewPicker:
		return "Connections"
	case ViewDevicePicker:
		return "Devices"
	case ViewCommandPalette:
		return "Commands"
	default:
		return ""
	}
}

func (m Model) renderFooter() string {
	if m.loading {
		return FooterStyle.Render(m.spinner.View() + " Loading...")
	}

	help := HelpKeyStyle.Render("Ctrl+P") + HelpDescStyle.Render(" navigate") +
		HelpKeyStyle.Render("  r") + HelpDescStyle.Render(" refresh") +
		HelpKeyStyle.Render("  ?") + HelpDescStyle.Render(" help") +
		HelpKeyStyle.Render("  q") + HelpDescStyle.Render(" quit")

	return FooterStyle.Render(help)
}

func (m Model) renderHelp() string {
	return m.help.View(m.keys)
}

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
			Label:       "Policies",
			Description: "Security rules",
			Category:    "Analyze",
			Action:      func() tea.Msg { return SwitchViewMsg{ViewPolicies} },
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

func (m Model) doLogin() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := auth.GenerateAPIKey(
			ctx,
			m.login.Host(),
			m.login.Username(),
			m.login.Password(),
			true,
		)
		if err != nil {
			return LoginErrorMsg{Err: err}
		}
		if result.Error != nil {
			return LoginErrorMsg{Err: result.Error}
		}
		return LoginSuccessMsg{
			Name:     m.login.Host(),
			APIKey:   result.APIKey,
			Username: m.login.Username(),
			Password: m.login.Password(),
		}
	}
}

func (m Model) fetchCurrentDashboardData() tea.Cmd {
	switch m.currentDashboard {
	case views.DashboardNetwork:
		return m.fetchNetworkDashboardData()
	case views.DashboardSecurity:
		return m.fetchSecurityDashboardData()
	case views.DashboardVPN:
		return m.fetchVPNDashboardData()
	case views.DashboardConfig:
		return m.fetchConfigDashboardData()
	default:
		return m.fetchDashboardData()
	}
}

func (m Model) fetchDashboardData() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchSystemInfo(conn),
		m.fetchResources(conn),
		m.fetchSessionInfo(conn),
		m.fetchHAStatus(conn),
		m.fetchInterfaces(),
		m.fetchThreatSummary(conn),
		m.fetchGlobalProtect(conn),
		m.fetchLoggedInAdmins(conn),
		m.fetchLicenses(conn),
		m.fetchJobs(conn),
		m.fetchDiskUsage(conn),
		m.fetchEnvironmentals(conn),
		m.fetchCertificates(conn),
	)
}

func (m Model) fetchNetworkDashboardData() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchInterfaces(),
		m.fetchARPTable(conn),
		m.fetchRoutingTable(conn),
	)
}

func (m Model) fetchSecurityDashboardData() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchThreatSummary(conn),
		m.fetchPolicies(),
	)
}

func (m Model) fetchVPNDashboardData() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchIPSecTunnels(conn),
		m.fetchGlobalProtectUsers(conn),
	)
}

func (m Model) fetchConfigDashboardData() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchPolicies(),
		m.fetchPendingChanges(conn),
	)
}

func (m Model) fetchSystemInfo(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		info, err := conn.Client.GetSystemInfo(context.Background())
		return SystemInfoMsg{Info: info, Err: err}
	}
}

func (m Model) fetchManagedDevices(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := conn.RefreshManagedDevices(ctx)
		return ManagedDevicesMsg{Devices: conn.ManagedDevices, Err: err}
	}
}

func (m Model) detectPanorama(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		info, err := conn.Client.GetSystemInfo(ctx)
		if err != nil {
			return PanoramaDetectedMsg{IsPanorama: false}
		}
		isPanorama := api.IsPanoramaModel(info.Model)
		return PanoramaDetectedMsg{IsPanorama: isPanorama, Model: info.Model}
	}
}

func (m Model) fetchResources(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		res, err := conn.Client.GetSystemResources(context.Background())
		return ResourcesMsg{Resources: res, Err: err}
	}
}

func (m Model) fetchSessionInfo(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		info, err := conn.Client.GetSessionInfo(context.Background())
		return SessionInfoMsg{Info: info, Err: err}
	}
}

func (m Model) fetchHAStatus(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		status, err := conn.Client.GetHAStatus(context.Background())
		return HAStatusMsg{Status: status, Err: err}
	}
}

func (m Model) fetchInterfaces() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}
	return func() tea.Msg {
		ifaces, err := conn.Client.GetInterfaces(context.Background())
		return InterfacesMsg{Interfaces: ifaces, Err: err}
	}
}

func (m Model) fetchThreatSummary(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		summary, err := conn.Client.GetThreatSummary(context.Background())
		return ThreatSummaryMsg{Summary: summary, Err: err}
	}
}

func (m Model) fetchGlobalProtect(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		info, err := conn.Client.GetGlobalProtectInfo(context.Background())
		return GlobalProtectMsg{Info: info, Err: err}
	}
}

func (m Model) fetchLoggedInAdmins(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		admins, err := conn.Client.GetLoggedInAdmins(context.Background())
		return LoggedInAdminsMsg{Admins: admins, Err: err}
	}
}

func (m Model) fetchLicenses(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		licenses, err := conn.Client.GetLicenseInfo(context.Background())
		return LicensesMsg{Licenses: licenses, Err: err}
	}
}

func (m Model) fetchJobs(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		jobs, err := conn.Client.GetJobs(context.Background())
		return JobsMsg{Jobs: jobs, Err: err}
	}
}

func (m Model) fetchDiskUsage(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		disks, err := conn.Client.GetDiskUsage(context.Background())
		return DiskUsageMsg{Disks: disks, Err: err}
	}
}

func (m Model) fetchEnvironmentals(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		envs, err := conn.Client.GetEnvironmentals(context.Background())
		return EnvironmentalsMsg{Environmentals: envs, Err: err}
	}
}

func (m Model) fetchCertificates(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		certs, err := conn.Client.GetCertificates(context.Background())
		return CertificatesMsg{Certificates: certs, Err: err}
	}
}

func (m Model) fetchARPTable(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		entries, err := conn.Client.GetARPTable(context.Background())
		return ARPTableMsg{Entries: entries, Err: err}
	}
}

func (m Model) fetchRoutingTable(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		routes, err := conn.Client.GetRoutingTable(context.Background())
		return RoutingTableMsg{Routes: routes, Err: err}
	}
}

func (m Model) fetchIPSecTunnels(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		tunnels, err := conn.Client.GetIPSecTunnels(context.Background())
		return IPSecTunnelsMsg{Tunnels: tunnels, Err: err}
	}
}

func (m Model) fetchGlobalProtectUsers(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		users, err := conn.Client.GetGlobalProtectUsers(context.Background())
		return GlobalProtectUsersMsg{Users: users, Err: err}
	}
}

func (m Model) fetchPendingChanges(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		changes, err := conn.Client.GetPendingChanges(context.Background())
		return PendingChangesMsg{Changes: changes, Err: err}
	}
}

func (m Model) fetchPolicies() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return func() tea.Msg {
		policies, err := conn.Client.GetSecurityPolicies(context.Background())
		return PoliciesMsg{Policies: policies, Err: err}
	}
}

func (m Model) fetchSessions() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return func() tea.Msg {
		sessions, err := conn.Client.GetSessions(context.Background(), "")
		return SessionsMsg{Sessions: sessions, Err: err}
	}
}

func (m Model) fetchLogs() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchSystemLogs(conn),
		m.fetchTrafficLogs(conn),
		m.fetchThreatLogs(conn),
	)
}

func (m Model) fetchSystemLogs(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		logs, err := conn.Client.GetSystemLogs(context.Background(), "", 100)
		return SystemLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) fetchTrafficLogs(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		logs, err := conn.Client.GetTrafficLogs(context.Background(), "", 100)
		return TrafficLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) fetchThreatLogs(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		logs, err := conn.Client.GetThreatLogs(context.Background(), "", 100)
		return ThreatLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) refreshCurrentView() tea.Cmd {
	switch m.currentView {
	case ViewDashboard:
		return m.fetchCurrentDashboardData()
	case ViewPolicies:
		return m.fetchPolicies()
	case ViewSessions:
		return m.fetchSessions()
	case ViewInterfaces:
		return m.fetchInterfaces()
	case ViewLogs:
		return m.fetchLogs()
	}
	return nil
}

func (m *Model) updateTroubleshootSSH() tea.Cmd {
	conn := m.session.GetActiveConnection()
	sshConfigured := conn != nil && conn.HasSSH()
	hasSSH := conn != nil && conn.SSHEnabled && conn.SSHClient != nil
	m.troubleshoot = m.troubleshoot.SetSSHConfigured(sshConfigured)
	m.troubleshoot = m.troubleshoot.SetSSHAvailable(hasSSH)
	return nil
}

func (m Model) connectSSH(conn *auth.Connection) tea.Cmd {
	return func() tea.Msg {
		if !conn.HasSSH() {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := conn.ConnectSSH(ctx); err != nil {
			return SSHErrorMsg{ConnectionName: conn.Name, Err: err}
		}
		return SSHConnectedMsg{ConnectionName: conn.Name}
	}
}

func (m Model) runTroubleshoot(runbook *troubleshoot.Runbook) tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return func() tea.Msg {
			return TroubleshootResultMsg{Err: fmt.Errorf("no active connection")}
		}
	}

	// Create engine with current connections
	engine := troubleshoot.NewEngine(conn.Client, conn.SSHClient, m.tsRegistry)

	return func() tea.Msg {
		ctx := context.Background()
		result, err := engine.RunRunbook(ctx, runbook)
		return TroubleshootResultMsg{Result: result, Err: err}
	}
}

// SetupDemoSSH configures SSH for demo mode with a mock SSH server.
// This is exported for use by pyre-demo.
func (m *Model) SetupDemoSSH(sshClient *ssh.Client) {
	conn := m.session.GetActiveConnection()
	if conn != nil {
		conn.SSHClient = sshClient
		conn.SSHEnabled = true
	}
	m.troubleshoot = m.troubleshoot.SetSSHAvailable(true)
}

// GetSession returns the session for external configuration.
// This is exported for use by pyre-demo.
func (m *Model) GetSession() *auth.Session {
	return m.session
}
