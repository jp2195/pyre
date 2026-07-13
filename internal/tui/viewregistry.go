package tui

import (
	tea "charm.land/bubbletea/v2"
)

// viewEntry is the single source of truth for a detail view's dispatch
// behavior. Every per-ViewState switch (resize, spinner, refresh, key
// forwarding, rendering, breadcrumb naming, navbar fetch) derives from
// this table; adding a view means adding one entry here plus a message
// case in dispatch.go.
type viewEntry struct {
	name       string // header breadcrumb, e.g. "Analyze/Policies"
	hasData    func(m *Model) bool
	isLoading  func(m *Model) bool
	setLoading func(m *Model)
	setSize    func(m *Model, w, h int)
	setSpinner func(m *Model, frame string)
	update     func(m *Model, msg tea.Msg) tea.Cmd
	render     func(m *Model) string
	refresh    func(m *Model) tea.Cmd // pure fetch; callers pair it with setLoading
}

// detailViewOrder gives deterministic iteration for loops (resize/spinner).
var detailViewOrder = []ViewState{
	ViewPolicies, ViewNATPolicies, ViewSessions, ViewInterfaces,
	ViewRoutes, ViewIPSecTunnels, ViewGPUsers, ViewLogs,
}

var detailViews = map[ViewState]viewEntry{
	ViewPolicies: {
		name:       "Analyze/Policies",
		hasData:    func(m *Model) bool { return m.policies.HasData() },
		isLoading:  func(m *Model) bool { return m.policies.IsLoading() },
		setLoading: func(m *Model) { m.policies = m.policies.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.policies = m.policies.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.policies = m.policies.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.policies, cmd = m.policies.Update(msg)
			return cmd
		},
		render:  func(m *Model) string { return m.policies.View() },
		refresh: func(m *Model) tea.Cmd { return m.fetchPolicies() },
	},
	ViewNATPolicies: {
		name:       "Analyze/NAT",
		hasData:    func(m *Model) bool { return m.natPolicies.HasData() },
		isLoading:  func(m *Model) bool { return m.natPolicies.IsLoading() },
		setLoading: func(m *Model) { m.natPolicies = m.natPolicies.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.natPolicies = m.natPolicies.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.natPolicies = m.natPolicies.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.natPolicies, cmd = m.natPolicies.Update(msg)
			return cmd
		},
		render:  func(m *Model) string { return m.natPolicies.View() },
		refresh: func(m *Model) tea.Cmd { return m.fetchNATPolicies() },
	},
	ViewSessions: {
		name:       "Analyze/Sessions",
		hasData:    func(m *Model) bool { return m.sessions.HasData() },
		isLoading:  func(m *Model) bool { return m.sessions.Loading },
		setLoading: func(m *Model) { m.sessions = m.sessions.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.sessions = m.sessions.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.sessions = m.sessions.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.sessions, cmd = m.sessions.Update(msg)
			return cmd
		},
		render:  func(m *Model) string { return m.sessions.View() },
		refresh: func(m *Model) tea.Cmd { return m.fetchSessions() },
	},
	ViewInterfaces: {
		name:       "Analyze/Interfaces",
		hasData:    func(m *Model) bool { return m.interfaces.HasData() },
		isLoading:  func(m *Model) bool { return m.interfaces.Loading },
		setLoading: func(m *Model) { m.interfaces = m.interfaces.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.interfaces = m.interfaces.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.interfaces = m.interfaces.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.interfaces, cmd = m.interfaces.Update(msg)
			return cmd
		},
		render: func(m *Model) string { return m.interfaces.View() },
		refresh: func(m *Model) tea.Cmd {
			// Interfaces always fetch the ARP table alongside; this is the
			// single definition (the pre-registry copies had drifted).
			if conn := m.session.GetActiveConnection(); conn != nil {
				return tea.Batch(m.fetchInterfaces(), m.fetchARPTable(conn))
			}
			return m.fetchInterfaces()
		},
	},
	ViewRoutes: {
		name:       "Analyze/Routes",
		hasData:    func(m *Model) bool { return m.routes.HasData() },
		isLoading:  func(m *Model) bool { return m.routes.Loading },
		setLoading: func(m *Model) { m.routes = m.routes.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.routes = m.routes.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.routes = m.routes.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.routes, cmd = m.routes.Update(msg)
			return cmd
		},
		render:  func(m *Model) string { return m.routes.View() },
		refresh: func(m *Model) tea.Cmd { return m.fetchRoutesData() },
	},
	ViewIPSecTunnels: {
		name:       "Analyze/IPSec",
		hasData:    func(m *Model) bool { return m.ipsecTunnels.HasData() },
		isLoading:  func(m *Model) bool { return m.ipsecTunnels.IsLoading() },
		setLoading: func(m *Model) { m.ipsecTunnels = m.ipsecTunnels.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.ipsecTunnels = m.ipsecTunnels.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.ipsecTunnels = m.ipsecTunnels.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.ipsecTunnels, cmd = m.ipsecTunnels.Update(msg)
			return cmd
		},
		render: func(m *Model) string { return m.ipsecTunnels.View() },
		refresh: func(m *Model) tea.Cmd {
			if conn := m.session.GetActiveConnection(); conn != nil {
				return m.fetchIPSecTunnels(conn)
			}
			return nil
		},
	},
	ViewGPUsers: {
		name:       "Analyze/GP Users",
		hasData:    func(m *Model) bool { return m.gpUsers.HasData() },
		isLoading:  func(m *Model) bool { return m.gpUsers.IsLoading() },
		setLoading: func(m *Model) { m.gpUsers = m.gpUsers.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.gpUsers = m.gpUsers.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.gpUsers = m.gpUsers.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.gpUsers, cmd = m.gpUsers.Update(msg)
			return cmd
		},
		render: func(m *Model) string { return m.gpUsers.View() },
		refresh: func(m *Model) tea.Cmd {
			if conn := m.session.GetActiveConnection(); conn != nil {
				return m.fetchGlobalProtectUsers(conn)
			}
			return nil
		},
	},
	ViewLogs: {
		name:       "Analyze/Logs",
		hasData:    func(m *Model) bool { return m.logs.HasData() },
		isLoading:  func(m *Model) bool { return m.logs.Loading },
		setLoading: func(m *Model) { m.logs = m.logs.SetLoading(true) },
		setSize:    func(m *Model, w, h int) { m.logs = m.logs.SetSize(w, h) },
		setSpinner: func(m *Model, f string) { m.logs = m.logs.SetSpinnerFrame(f) },
		update: func(m *Model, msg tea.Msg) tea.Cmd {
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			return cmd
		},
		render:  func(m *Model) string { return m.logs.View() },
		refresh: func(m *Model) tea.Cmd { return m.fetchLogs() },
	},
}

// detailNavTarget builds a navTarget that delegates to the registry, so
// navbar navigation can never drift from refresh/switch-view behavior.
func detailNavTarget(v ViewState) navTarget {
	return navTarget{
		view:    v,
		hasData: func(m *Model) bool { return detailViews[v].hasData(m) },
		fetch: func(m *Model) tea.Cmd {
			e := detailViews[v]
			e.setLoading(m)
			return e.refresh(m)
		},
	}
}
