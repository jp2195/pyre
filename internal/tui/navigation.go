package tui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/tui/views"
)

// navTarget describes the view state and data fetch for a navigation item.
type navTarget struct {
	view      ViewState
	dashboard views.DashboardType // Only used when view == ViewDashboard
	hasData   func(m *Model) bool
	fetch     func(m *Model) tea.Cmd
}

// navItemDef is one entry in navDefs: a navbar item plus its navigation
// target. Embedding navTarget makes the navTargets derivation lossless by
// construction.
type navItemDef struct {
	id    string
	label string
	navTarget
}

// navGroupDef is one navbar group in navDefs.
type navGroupDef struct {
	id    string
	label string
	items []navItemDef
}

// navDefs is the single source of truth for navigation. The navbar widget
// groups (navbarGroups), navTargets, and viewToNavbar are all derived from
// it. Adding a nav item means adding exactly one entry here.
var navDefs = []navGroupDef{
	{
		id:    "monitor",
		label: "Monitor",
		items: []navItemDef{
			{id: "overview", label: "Overview", navTarget: navTarget{
				view:      ViewDashboard,
				dashboard: views.DashboardMain,
				hasData:   func(m *Model) bool { return m.dashboard.HasData() },
				fetch:     func(m *Model) tea.Cmd { return m.fetchDashboardData() },
			}},
			{id: "network", label: "Network", navTarget: navTarget{
				view:      ViewDashboard,
				dashboard: views.DashboardNetwork,
				hasData:   func(m *Model) bool { return m.networkDashboard.HasData() },
				fetch:     func(m *Model) tea.Cmd { return m.fetchNetworkDashboardData() },
			}},
			{id: "security", label: "Security", navTarget: navTarget{
				view:      ViewDashboard,
				dashboard: views.DashboardSecurity,
				hasData:   func(m *Model) bool { return m.securityDashboard.HasData() },
				fetch:     func(m *Model) tea.Cmd { return m.fetchSecurityDashboardData() },
			}},
			{id: "vpn", label: "VPN", navTarget: navTarget{
				view:      ViewDashboard,
				dashboard: views.DashboardVPN,
				hasData:   func(m *Model) bool { return m.vpnDashboard.HasData() },
				fetch:     func(m *Model) tea.Cmd { return m.fetchVPNDashboardData() },
			}},
		},
	},
	{
		id:    "analyze",
		label: "Analyze",
		items: []navItemDef{
			{id: "policies", label: "Policies", navTarget: navTarget{
				view:    ViewPolicies,
				hasData: func(m *Model) bool { return m.policies.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.policies = m.policies.SetLoading(true)
					return m.fetchPolicies()
				},
			}},
			{id: "nat", label: "NAT", navTarget: navTarget{
				view:    ViewNATPolicies,
				hasData: func(m *Model) bool { return m.natPolicies.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.natPolicies = m.natPolicies.SetLoading(true)
					return m.fetchNATPolicies()
				},
			}},
			{id: "objects", label: "Objects", navTarget: navTarget{
				view:    ViewObjects,
				hasData: func(m *Model) bool { return m.objects.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.objects = m.objects.SetLoading(true)
					return m.fetchObjects()
				},
			}},
			{id: "sessions", label: "Sessions", navTarget: navTarget{
				view:    ViewSessions,
				hasData: func(m *Model) bool { return m.sessions.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.sessions = m.sessions.SetLoading(true)
					return m.fetchSessions()
				},
			}},
			{id: "interfaces", label: "Interfaces", navTarget: navTarget{
				view:    ViewInterfaces,
				hasData: func(m *Model) bool { return m.interfaces.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.interfaces = m.interfaces.SetLoading(true)
					return m.fetchInterfaces()
				},
			}},
			{id: "routes", label: "Routes", navTarget: navTarget{
				view:    ViewRoutes,
				hasData: func(m *Model) bool { return m.routes.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.routes = m.routes.SetLoading(true)
					return m.fetchRoutesData()
				},
			}},
			{id: "ipsec", label: "IPSec", navTarget: navTarget{
				view:    ViewIPSecTunnels,
				hasData: func(m *Model) bool { return m.ipsecTunnels.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.ipsecTunnels = m.ipsecTunnels.SetLoading(true)
					conn := m.session.GetActiveConnection()
					if conn != nil {
						return m.fetchIPSecTunnels(conn)
					}
					return nil
				},
			}},
			{id: "gpusers", label: "GP Users", navTarget: navTarget{
				view:    ViewGPUsers,
				hasData: func(m *Model) bool { return m.gpUsers.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.gpUsers = m.gpUsers.SetLoading(true)
					conn := m.session.GetActiveConnection()
					if conn != nil {
						return m.fetchGlobalProtectUsers(conn)
					}
					return nil
				},
			}},
			{id: "logs", label: "Logs", navTarget: navTarget{
				view:    ViewLogs,
				hasData: func(m *Model) bool { return m.logs.HasData() },
				fetch: func(m *Model) tea.Cmd {
					m.logs = m.logs.SetLoading(true)
					return m.fetchLogs()
				},
			}},
		},
	},
	{
		id:    "tools",
		label: "Tools",
		items: []navItemDef{
			{id: "config", label: "Config", navTarget: navTarget{
				view:      ViewDashboard,
				dashboard: views.DashboardConfig,
				hasData:   func(m *Model) bool { return m.configDashboard.HasData() },
				fetch:     func(m *Model) tea.Cmd { return m.fetchConfigDashboardData() },
			}},
		},
	},
}

// navbarGroups derives the navbar widget's groups from navDefs. Display
// keys are positional for both groups and items, preserving the keys of
// the original views/navbar.go literal.
func navbarGroups() []views.NavGroup {
	groups := make([]views.NavGroup, 0, len(navDefs))
	for gi, g := range navDefs {
		items := make([]views.NavItem, 0, len(g.items))
		for ii, it := range g.items {
			items = append(items, views.NavItem{ID: it.id, Label: it.label, Key: strconv.Itoa(ii + 1)})
		}
		groups = append(groups, views.NavGroup{ID: g.id, Label: g.label, Key: strconv.Itoa(gi + 1), Items: items})
	}
	return groups
}

// navTargets maps navbar item IDs to their navigation targets, derived
// from navDefs at package init.
var navTargets = buildNavTargets()

func buildNavTargets() map[string]navTarget {
	targets := make(map[string]navTarget)
	for _, g := range navDefs {
		for _, it := range g.items {
			targets[it.id] = it.navTarget
		}
	}
	return targets
}

// navbarID identifies a navbar item by its group and item IDs.
type navbarID struct {
	group string
	item  string
}

// navbarEntry is an entry in viewToNavbar. isDashboard marks entries whose
// dashboard field is load-bearing; for all other views it is false and the
// dashboard field must be ignored (it is zero-valued but not meaningful).
type navbarEntry struct {
	isDashboard bool
	dashboard   views.DashboardType // only valid when isDashboard
	id          navbarID
}

// viewToNavbar maps each ViewState to its navbar entries, derived from
// navDefs at package init. Entry order follows table order, preserving
// first-match-wins in syncNavbarToCurrentView. isDashboard is derived:
// exactly the ViewDashboard entries have a load-bearing dashboard field.
var viewToNavbar = buildViewToNavbar()

func buildViewToNavbar() map[ViewState][]navbarEntry {
	vt := make(map[ViewState][]navbarEntry)
	for _, g := range navDefs {
		for _, it := range g.items {
			vt[it.view] = append(vt[it.view], navbarEntry{
				isDashboard: it.view == ViewDashboard,
				dashboard:   it.dashboard,
				id:          navbarID{group: g.id, item: it.id},
			})
		}
	}
	return vt
}

// handleNavGroupKey handles number key presses for navigation
func (m Model) handleNavGroupKey(groupIndex int) (tea.Model, tea.Cmd) {
	currentGroup := m.navbar.ActiveGroupIndex()

	if currentGroup == groupIndex {
		// Already in this group - cycle to next item
		group := m.navbar.ActiveGroup()
		if group != nil {
			nextItem := (m.navbar.ActiveItemIndex() + 1) % len(group.Items)
			m.navbar = m.navbar.SetActiveItem(nextItem)
		}
	} else {
		// Switch to the new group (first item)
		m.navbar = m.navbar.SetActiveGroup(groupIndex)
	}

	// Navigate to the selected item
	return m.navigateToCurrentItem()
}

// navigateToCurrentItem switches to the view corresponding to the current navbar item
func (m Model) navigateToCurrentItem() (tea.Model, tea.Cmd) {
	item := m.navbar.ActiveItem()
	if item == nil {
		return m, nil
	}

	target, ok := navTargets[item.ID]
	if !ok {
		return m, nil
	}

	m.currentView = target.view
	if target.view == ViewDashboard {
		m.currentDashboard = target.dashboard
	}

	var cmd tea.Cmd
	if target.fetch != nil && !target.hasData(&m) {
		cmd = target.fetch(&m)
	}

	return m, cmd
}

// syncNavbarToCurrentView syncs the navbar state to match the current view
func (m *Model) syncNavbarToCurrentView() {
	entries, ok := viewToNavbar[m.currentView]
	if !ok {
		return
	}
	for _, e := range entries {
		if !e.isDashboard || e.dashboard == m.currentDashboard {
			m.navbar = m.navbar.SetActiveByID(e.id.group, e.id.item)
			return
		}
	}
}
