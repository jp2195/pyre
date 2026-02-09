package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jp2195/pyre/internal/api"
	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/views"
)

// fetchCmd creates a tea.Cmd that calls fn with the given context and wraps the result in a message.
func fetchCmd[T any](ctx context.Context, fn func(context.Context) (T, error), wrap func(T, error) tea.Msg) tea.Cmd {
	return func() tea.Msg {
		result, err := fn(ctx)
		return wrap(result, err)
	}
}

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
			Host:     host,
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
	return fetchCmd(m.ctx, conn.Client.GetSystemInfo, func(info *models.SystemInfo, err error) tea.Msg {
		return SystemInfoMsg{Info: info, Err: err}
	})
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
	return fetchCmd(m.ctx, conn.Client.GetSessionInfo, func(info *models.SessionInfo, err error) tea.Msg {
		return SessionInfoMsg{Info: info, Err: err}
	})
}

func (m Model) fetchHAStatus(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetHAStatus, func(status *models.HAStatus, err error) tea.Msg {
		return HAStatusMsg{Status: status, Err: err}
	})
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
	return fetchCmd(m.ctx, conn.Client.GetThreatSummary, func(s *models.ThreatSummary, err error) tea.Msg {
		return ThreatSummaryMsg{Summary: s, Err: err}
	})
}

func (m Model) fetchGlobalProtect(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetGlobalProtectInfo, func(info *models.GlobalProtectInfo, err error) tea.Msg {
		return GlobalProtectMsg{Info: info, Err: err}
	})
}

func (m Model) fetchLoggedInAdmins(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetLoggedInAdmins, func(admins []models.LoggedInAdmin, err error) tea.Msg {
		return LoggedInAdminsMsg{Admins: admins, Err: err}
	})
}

func (m Model) fetchLicenses(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetLicenseInfo, func(lics []models.LicenseInfo, err error) tea.Msg {
		return LicensesMsg{Licenses: lics, Err: err}
	})
}

func (m Model) fetchJobs(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetJobs, func(jobs []models.Job, err error) tea.Msg {
		return JobsMsg{Jobs: jobs, Err: err}
	})
}

func (m Model) fetchDiskUsage(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetDiskUsage, func(disks []models.DiskUsage, err error) tea.Msg {
		return DiskUsageMsg{Disks: disks, Err: err}
	})
}

//nolint:misspell // "environmentals" is the PAN-OS XML API tag name
func (m Model) fetchEnvironmentals(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetEnvironmentals, func(envs []models.Environmental, err error) tea.Msg {
		return EnvironmentalsMsg{Environmentals: envs, Err: err}
	})
}

func (m Model) fetchCertificates(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetCertificates, func(certs []models.Certificate, err error) tea.Msg {
		return CertificatesMsg{Certificates: certs, Err: err}
	})
}

func (m Model) fetchARPTable(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetARPTable, func(entries []models.ARPEntry, err error) tea.Msg {
		return ARPTableMsg{Entries: entries, Err: err}
	})
}

func (m Model) fetchRoutingTable(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetRoutingTable, func(routes []models.RouteEntry, err error) tea.Msg {
		return RoutingTableMsg{Routes: routes, Err: err}
	})
}

func (m Model) fetchBGPNeighbors(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetBGPNeighbors, func(n []models.BGPNeighbor, err error) tea.Msg {
		return BGPNeighborsMsg{Neighbors: n, Err: err}
	})
}

func (m Model) fetchOSPFNeighbors(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetOSPFNeighbors, func(n []models.OSPFNeighbor, err error) tea.Msg {
		return OSPFNeighborsMsg{Neighbors: n, Err: err}
	})
}

func (m Model) fetchIPSecTunnels(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetIPSecTunnels, func(t []models.IPSecTunnel, err error) tea.Msg {
		return IPSecTunnelsMsg{Tunnels: t, Err: err}
	})
}

func (m Model) fetchGlobalProtectUsers(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetGlobalProtectUsers, func(u []models.GlobalProtectUser, err error) tea.Msg {
		return GlobalProtectUsersMsg{Users: u, Err: err}
	})
}

func (m Model) fetchPendingChanges(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetPendingChanges, func(c []models.PendingChange, err error) tea.Msg {
		return PendingChangesMsg{Changes: c, Err: err}
	})
}

func (m Model) fetchNATPoolInfo(conn *auth.Connection) tea.Cmd {
	return fetchCmd(m.ctx, conn.Client.GetNATPoolInfo, func(p []models.NATPoolInfo, err error) tea.Msg {
		return NATPoolMsg{Pools: p, Err: err}
	})
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
	case ViewRoutes:
		return m.fetchRoutesData()
	case ViewIPSecTunnels:
		conn := m.session.GetActiveConnection()
		if conn != nil {
			return m.fetchIPSecTunnels(conn)
		}
	case ViewGPUsers:
		conn := m.session.GetActiveConnection()
		if conn != nil {
			return m.fetchGlobalProtectUsers(conn)
		}
	case ViewLogs:
		return m.fetchLogs()
	}
	return nil
}

func (m Model) saveConfig() tea.Cmd {
	cfg := m.config
	return func() tea.Msg {
		err := cfg.Save()
		return ConfigSavedMsg{Err: err}
	}
}

func (m Model) saveState() tea.Cmd {
	state := m.state
	return func() tea.Msg {
		err := state.Save()
		return StateSavedMsg{Err: err}
	}
}

func (m Model) fetchRoutesData() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	return tea.Batch(
		m.fetchRoutingTable(conn),
		m.fetchBGPNeighbors(conn),
		m.fetchOSPFNeighbors(conn),
	)
}
