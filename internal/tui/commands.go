package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/api"
	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/tui/views"
)

func (m Model) doLogin() tea.Cmd {
	ctx := m.ctx
	host := m.login.Host()
	username := m.login.Username()
	password := m.login.Password()
	insecure := m.login.Insecure()

	return func() tea.Msg {
		result, err := auth.GenerateAPIKey(ctx, host, username, password, insecure)
		if err != nil {
			return LoginErrorMsg{Err: err}
		}
		if result.Error != nil {
			return LoginErrorMsg{Err: result.Error}
		}

		// Password is now out of scope and will be garbage collected
		return LoginSuccessMsg{
			Name:     host,
			APIKey:   result.APIKey,
			Username: username,
			Insecure: insecure,
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
		m.fetchNATPoolInfo(conn),
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
		m.fetchBGPNeighbors(conn),
		m.fetchOSPFNeighbors(conn),
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
	ctx := m.ctx
	return func() tea.Msg {
		info, err := conn.Client.GetSystemInfo(ctx)
		return SystemInfoMsg{Info: info, Err: err}
	}
}

func (m Model) fetchManagedDevices(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		err := conn.RefreshManagedDevices(ctx)
		return ManagedDevicesMsg{Devices: conn.ManagedDevices, Err: err}
	}
}

func (m Model) detectPanorama(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		info, err := conn.Client.GetSystemInfo(ctx)
		if err != nil {
			return PanoramaDetectedMsg{IsPanorama: false}
		}
		isPanorama := api.IsPanoramaModel(info.Model)
		return PanoramaDetectedMsg{IsPanorama: isPanorama, Model: info.Model}
	}
}

func (m Model) fetchResources(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		res, err := conn.Client.GetSystemResources(ctx)
		if err != nil {
			return ResourcesMsg{Resources: res, Err: err}
		}

		// Fetch dataplane CPU separately and merge into resources
		dpCPU, dpErr := conn.Client.GetDataPlaneResources(ctx)
		if dpErr == nil {
			res.DataPlaneCPU = dpCPU
		}
		// Don't fail the whole request if dataplane fetch fails

		return ResourcesMsg{Resources: res, Err: err}
	}
}

func (m Model) fetchSessionInfo(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		info, err := conn.Client.GetSessionInfo(ctx)
		return SessionInfoMsg{Info: info, Err: err}
	}
}

func (m Model) fetchHAStatus(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		status, err := conn.Client.GetHAStatus(ctx)
		return HAStatusMsg{Status: status, Err: err}
	}
}

func (m Model) fetchInterfaces() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}
	ctx := m.ctx
	return func() tea.Msg {
		ifaces, err := conn.Client.GetInterfaces(ctx)
		return InterfacesMsg{Interfaces: ifaces, Err: err}
	}
}

func (m Model) fetchThreatSummary(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		summary, err := conn.Client.GetThreatSummary(ctx)
		return ThreatSummaryMsg{Summary: summary, Err: err}
	}
}

func (m Model) fetchGlobalProtect(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		info, err := conn.Client.GetGlobalProtectInfo(ctx)
		return GlobalProtectMsg{Info: info, Err: err}
	}
}

func (m Model) fetchLoggedInAdmins(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		admins, err := conn.Client.GetLoggedInAdmins(ctx)
		return LoggedInAdminsMsg{Admins: admins, Err: err}
	}
}

func (m Model) fetchLicenses(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		licenses, err := conn.Client.GetLicenseInfo(ctx)
		return LicensesMsg{Licenses: licenses, Err: err}
	}
}

func (m Model) fetchJobs(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		jobs, err := conn.Client.GetJobs(ctx)
		return JobsMsg{Jobs: jobs, Err: err}
	}
}

func (m Model) fetchDiskUsage(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		disks, err := conn.Client.GetDiskUsage(ctx)
		return DiskUsageMsg{Disks: disks, Err: err}
	}
}

//nolint:misspell // "environmentals" is the PAN-OS XML API tag name
func (m Model) fetchEnvironmentals(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		envs, err := conn.Client.GetEnvironmentals(ctx)
		return EnvironmentalsMsg{Environmentals: envs, Err: err}
	}
}

func (m Model) fetchCertificates(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		certs, err := conn.Client.GetCertificates(ctx)
		return CertificatesMsg{Certificates: certs, Err: err}
	}
}

func (m Model) fetchARPTable(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		entries, err := conn.Client.GetARPTable(ctx)
		return ARPTableMsg{Entries: entries, Err: err}
	}
}

func (m Model) fetchRoutingTable(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		routes, err := conn.Client.GetRoutingTable(ctx)
		return RoutingTableMsg{Routes: routes, Err: err}
	}
}

func (m Model) fetchBGPNeighbors(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		neighbors, err := conn.Client.GetBGPNeighbors(ctx)
		return BGPNeighborsMsg{Neighbors: neighbors, Err: err}
	}
}

func (m Model) fetchOSPFNeighbors(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		neighbors, err := conn.Client.GetOSPFNeighbors(ctx)
		return OSPFNeighborsMsg{Neighbors: neighbors, Err: err}
	}
}

func (m Model) fetchIPSecTunnels(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		tunnels, err := conn.Client.GetIPSecTunnels(ctx)
		return IPSecTunnelsMsg{Tunnels: tunnels, Err: err}
	}
}

func (m Model) fetchGlobalProtectUsers(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		users, err := conn.Client.GetGlobalProtectUsers(ctx)
		return GlobalProtectUsersMsg{Users: users, Err: err}
	}
}

func (m Model) fetchPendingChanges(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		changes, err := conn.Client.GetPendingChanges(ctx)
		return PendingChangesMsg{Changes: changes, Err: err}
	}
}

func (m Model) fetchNATPoolInfo(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		pools, err := conn.Client.GetNATPoolInfo(ctx)
		return NATPoolMsg{Pools: pools, Err: err}
	}
}

func (m Model) fetchPolicies() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	return func() tea.Msg {
		policies, err := conn.Client.GetSecurityPolicies(ctx)
		return PoliciesMsg{Policies: policies, Err: err}
	}
}

func (m Model) fetchNATPolicies() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	return func() tea.Msg {
		rules, err := conn.Client.GetNATRules(ctx)
		return NATPoliciesMsg{Rules: rules, Err: err}
	}
}

func (m Model) fetchSessions() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	return func() tea.Msg {
		sessions, err := conn.Client.GetSessions(ctx, "")
		return SessionsMsg{Sessions: sessions, Err: err}
	}
}

func (m Model) fetchSessionDetail(id int64) tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	return func() tea.Msg {
		detail, err := conn.Client.GetSessionByID(ctx, id)
		return SessionDetailMsg{Detail: detail, Err: err}
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
	ctx := m.ctx
	return func() tea.Msg {
		logs, err := conn.Client.GetSystemLogs(ctx, "", 100)
		return SystemLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) fetchTrafficLogs(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		logs, err := conn.Client.GetTrafficLogs(ctx, "", 100)
		return TrafficLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) fetchThreatLogs(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		logs, err := conn.Client.GetThreatLogs(ctx, "", 100)
		return ThreatLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) refreshCurrentView() tea.Cmd {
	switch m.currentView {
	case ViewDashboard:
		return m.fetchCurrentDashboardData()
	case ViewPolicies:
		return m.fetchPolicies()
	case ViewNATPolicies:
		return m.fetchNATPolicies()
	case ViewSessions:
		return m.fetchSessions()
	case ViewInterfaces:
		conn := m.session.GetActiveConnection()
		if conn != nil {
			return tea.Batch(m.fetchInterfaces(), m.fetchARPTable(conn))
		}
		return m.fetchInterfaces()
	case ViewLogs:
		return m.fetchLogs()
	}
	return nil
}
