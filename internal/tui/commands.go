package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

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
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) (*models.SystemInfo, error) {
		return conn.Client.GetSystemInfo(ctx, target)
	}, func(info *models.SystemInfo, err error) tea.Msg {
		return SystemInfoMsg{Info: info, Err: err}
	})
}

func (m Model) fetchManagedDevices(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		err := conn.RefreshManagedDevices(ctx)
		return ManagedDevicesMsg{Devices: conn.ManagedDevicesSnapshot(), Err: err}
	}
}

func (m Model) detectPanorama(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	return func() tea.Msg {
		// Detection always queries the appliance itself, never a managed device.
		info, err := conn.Client.GetSystemInfo(ctx, "")
		if err != nil {
			return PanoramaDetectedMsg{IsPanorama: false}
		}
		isPanorama := api.IsPanoramaModel(info.Model)
		return PanoramaDetectedMsg{IsPanorama: isPanorama, Model: info.Model}
	}
}

func (m Model) fetchResources(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		res, err := conn.Client.GetSystemResources(ctx, target)
		if err != nil {
			return ResourcesMsg{Resources: res, Err: err}
		}

		// Fetch dataplane CPU separately and merge into resources
		dpCPU, dpErr := conn.Client.GetDataPlaneResources(ctx, target)
		if dpErr == nil {
			res.DataPlaneCPU = dpCPU
		}
		// Don't fail the whole request if dataplane fetch fails

		return ResourcesMsg{Resources: res, Err: err}
	}
}

func (m Model) fetchSessionInfo(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) (*models.SessionInfo, error) {
		return conn.Client.GetSessionInfo(ctx, target)
	}, func(info *models.SessionInfo, err error) tea.Msg {
		return SessionInfoMsg{Info: info, Err: err}
	})
}

func (m Model) fetchHAStatus(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) (*models.HAStatus, error) {
		return conn.Client.GetHAStatus(ctx, target)
	}, func(status *models.HAStatus, err error) tea.Msg {
		return HAStatusMsg{Status: status, Err: err}
	})
}

func (m Model) fetchInterfaces() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}
	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		ifaces, err := conn.Client.GetInterfaces(ctx, target)
		return InterfacesMsg{Interfaces: ifaces, Err: err}
	}
}

func (m Model) fetchThreatSummary(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) (*models.ThreatSummary, error) {
		return conn.Client.GetThreatSummary(ctx, target)
	}, func(s *models.ThreatSummary, err error) tea.Msg {
		return ThreatSummaryMsg{Summary: s, Err: err}
	})
}

func (m Model) fetchGlobalProtect(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) (*models.GlobalProtectInfo, error) {
		return conn.Client.GetGlobalProtectInfo(ctx, target)
	}, func(info *models.GlobalProtectInfo, err error) tea.Msg {
		return GlobalProtectMsg{Info: info, Err: err}
	})
}

func (m Model) fetchLoggedInAdmins(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.LoggedInAdmin, error) {
		return conn.Client.GetLoggedInAdmins(ctx, target)
	}, func(admins []models.LoggedInAdmin, err error) tea.Msg {
		return LoggedInAdminsMsg{Admins: admins, Err: err}
	})
}

func (m Model) fetchLicenses(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.LicenseInfo, error) {
		return conn.Client.GetLicenseInfo(ctx, target)
	}, func(lics []models.LicenseInfo, err error) tea.Msg {
		return LicensesMsg{Licenses: lics, Err: err}
	})
}

func (m Model) fetchJobs(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.Job, error) {
		return conn.Client.GetJobs(ctx, target)
	}, func(jobs []models.Job, err error) tea.Msg {
		return JobsMsg{Jobs: jobs, Err: err}
	})
}

func (m Model) fetchDiskUsage(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.DiskUsage, error) {
		return conn.Client.GetDiskUsage(ctx, target)
	}, func(disks []models.DiskUsage, err error) tea.Msg {
		return DiskUsageMsg{Disks: disks, Err: err}
	})
}

//nolint:misspell // "environmentals" is the PAN-OS XML API tag name
func (m Model) fetchEnvironmentals(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.Environmental, error) {
		return conn.Client.GetEnvironmentals(ctx, target)
	}, func(envs []models.Environmental, err error) tea.Msg {
		return EnvironmentalsMsg{Environmentals: envs, Err: err}
	})
}

func (m Model) fetchCertificates(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.Certificate, error) {
		return conn.Client.GetCertificates(ctx, target)
	}, func(certs []models.Certificate, err error) tea.Msg {
		return CertificatesMsg{Certificates: certs, Err: err}
	})
}

func (m Model) fetchARPTable(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.ARPEntry, error) {
		return conn.Client.GetARPTable(ctx, target)
	}, func(entries []models.ARPEntry, err error) tea.Msg {
		return ARPTableMsg{Entries: entries, Err: err}
	})
}

func (m Model) fetchRoutingTable(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.RouteEntry, error) {
		return conn.Client.GetRoutingTable(ctx, target)
	}, func(routes []models.RouteEntry, err error) tea.Msg {
		return RoutingTableMsg{Routes: routes, Err: err}
	})
}

func (m Model) fetchBGPNeighbors(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.BGPNeighbor, error) {
		return conn.Client.GetBGPNeighbors(ctx, target)
	}, func(n []models.BGPNeighbor, err error) tea.Msg {
		return BGPNeighborsMsg{Neighbors: n, Err: err}
	})
}

func (m Model) fetchOSPFNeighbors(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.OSPFNeighbor, error) {
		return conn.Client.GetOSPFNeighbors(ctx, target)
	}, func(n []models.OSPFNeighbor, err error) tea.Msg {
		return OSPFNeighborsMsg{Neighbors: n, Err: err}
	})
}

func (m Model) fetchIPSecTunnels(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.IPSecTunnel, error) {
		return conn.Client.GetIPSecTunnels(ctx, target)
	}, func(t []models.IPSecTunnel, err error) tea.Msg {
		return IPSecTunnelsMsg{Tunnels: t, Err: err}
	})
}

func (m Model) fetchGlobalProtectUsers(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.GlobalProtectUser, error) {
		return conn.Client.GetGlobalProtectUsers(ctx, target)
	}, func(u []models.GlobalProtectUser, err error) tea.Msg {
		return GlobalProtectUsersMsg{Users: u, Err: err}
	})
}

func (m Model) fetchPendingChanges(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.PendingChange, error) {
		return conn.Client.GetPendingChanges(ctx, target)
	}, func(c []models.PendingChange, err error) tea.Msg {
		return PendingChangesMsg{Changes: c, Err: err}
	})
}

func (m Model) fetchNATPoolInfo(conn *auth.Connection) tea.Cmd {
	target := conn.Target()
	return fetchCmd(m.ctx, func(ctx context.Context) ([]models.NATPoolInfo, error) {
		return conn.Client.GetNATPoolInfo(ctx, target)
	}, func(p []models.NATPoolInfo, err error) tea.Msg {
		return NATPoolMsg{Pools: p, Err: err}
	})
}

func (m Model) fetchPolicies() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		policies, err := conn.Client.GetSecurityPolicies(ctx, target)
		return PoliciesMsg{Policies: policies, Err: err}
	}
}

func (m Model) fetchNATPolicies() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		rules, err := conn.Client.GetNATRules(ctx, target)
		return NATPoliciesMsg{Rules: rules, Err: err}
	}
}

func (m Model) fetchSessions() tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		sessions, err := conn.Client.GetSessions(ctx, "", target)
		return SessionsMsg{Sessions: sessions, Err: err}
	}
}

func (m Model) fetchSessionDetail(id int64) tea.Cmd {
	conn := m.session.GetActiveConnection()
	if conn == nil {
		return nil
	}

	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		detail, err := conn.Client.GetSessionByID(ctx, id, target)
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
	target := conn.Target()
	return func() tea.Msg {
		logs, err := conn.Client.GetSystemLogs(ctx, "", 100, target)
		return SystemLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) fetchTrafficLogs(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		logs, err := conn.Client.GetTrafficLogs(ctx, "", 100, target)
		return TrafficLogsMsg{Logs: logs, Err: err}
	}
}

func (m Model) fetchThreatLogs(conn *auth.Connection) tea.Cmd {
	ctx := m.ctx
	target := conn.Target()
	return func() tea.Msg {
		logs, err := conn.Client.GetThreatLogs(ctx, "", 100, target)
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
