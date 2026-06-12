package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	ViewObjects
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
	objects           views.ObjectsModel
	picker            views.PickerModel
	devicePicker      views.DevicePickerModel
	commandPalette    views.CommandPaletteModel
	previousView      ViewState // Track previous view for Esc to return

	// selectedConnection stores the connection selected from hub before login
	selectedConnection       string
	selectedConnectionConfig config.ConnectionConfig
}

func NewModel(cfg *config.Config, state *config.State, creds *auth.Credentials, startView ViewState) (Model, error) {
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
		if _, err := session.AddConnection(creds.Host, connConfig, creds.APIKey); err != nil {
			cancel()
			return Model{}, fmt.Errorf("initializing connection to %s: %w", creds.Host, err)
		}
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
	m.objects = views.NewObjectsModel()
	m.picker = views.NewPickerModel(session)
	m.devicePicker = views.NewDevicePickerModel()
	m.commandPalette = views.NewCommandPaletteModel()

	return m, nil
}

// setError returns a copy of the model with the error set, plus a Cmd
// that auto-dismisses the error after errorDismissTimeout.
func (m Model) setError(err error) (Model, tea.Cmd) {
	m.err = err
	return m, tea.Tick(errorDismissTimeout, func(time.Time) tea.Msg {
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
	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	default:
		return m.handleDataMsg(msg)
	}
}

// handleWindowSize propagates resize events to all sub-views via viewSlots.
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.help.SetWidth(msg.Width)

	// Calculate content height (minus header and footer)
	// Header = main row + sub-tab row + border = 3 lines
	// Footer = 1 line
	contentH := msg.Height - 4

	for _, s := range viewSlots() {
		s.resize(&m, msg.Width, msg.Height, contentH)
	}

	return m, nil
}

// handleKeyMsg routes keyboard input to the appropriate view or global handler.
func (m Model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		if conn != nil {
			if conn.PanoramaInfo() {
				m.currentView = ViewDevicePicker
				m.devicePicker = m.devicePicker.SetDevices(
					conn.ManagedDevicesSnapshot(),
					conn.Target(),
					conn.Host,
				)
				return m, nil
			}
		}
		// Not Panorama — fall through to view-level handler so 'd' reaches
		// views (e.g. sessions) that bind it to their own action.

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
		// Objects view uses Tab internally to cycle Address ↔ Service.
		if m.currentView == ViewObjects {
			return m.handleViewKeys(msg)
		}
		group := m.navbar.ActiveGroup()
		if group != nil && len(group.Items) > 0 {
			nextItem := (m.navbar.ActiveItemIndex() + 1) % len(group.Items)
			m.navbar = m.navbar.SetActiveItem(nextItem)
			return m.navigateToCurrentItem()
		}

	// Shift+Tab cycles backward through items in current group
	case key.Matches(msg, m.keys.ShiftTab):
		// Objects view uses Tab internally to cycle Address ↔ Service.
		if m.currentView == ViewObjects {
			return m.handleViewKeys(msg)
		}
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

// handleRefresh sets loading state and refreshes the current view via viewSlots.
func (m Model) handleRefresh() (tea.Model, tea.Cmd) {
	for _, s := range viewSlots() {
		if s.loading != nil && s.refreshFor == m.currentView {
			s.loading(&m, true)
		}
	}
	return m, m.refreshCurrentView()
}

// handleSpinnerTick updates the spinner and shares its frame with all views via viewSlots.
func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	frame := m.spinner.View()
	for _, s := range viewSlots() {
		if s.spinner != nil {
			s.spinner(&m, frame)
		}
	}

	return m, cmd
}

// View is the top-level Bubble Tea v2 view. It composes sub-view strings and
// wraps them in a tea.View so that program-level options (alt-screen, mouse,
// window title, cursor) can be set here rather than on tea.NewProgram.
func (m Model) View() tea.View {
	v := tea.NewView(m.renderContent())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderContent produces the raw string content for the current view state.
// It is kept as a pointer-free string helper so it can be called from tests
// and composed by tea.View.
func (m Model) renderContent() string {
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

	case ViewObjects:
		content = m.objects.View()
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
