package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/tui/views"
)

// scrollJump is larger than any plausible dashboard, so passing it to
// ScrollBy lands on the first or last page (ScrollBy clamps).
const scrollJump = 1 << 20

// scrollActiveDashboard moves the active dashboard's scroll offset by delta
// lines. Dashboards render a fixed panel stack that is routinely taller than
// the terminal — see views.DashboardBase.ClampToHeight.
//
// Scrolling is handled here rather than in each dashboard's Update because
// handleViewKeys routes ViewDashboard keys to m.dashboard (the Overview model)
// regardless of which dashboard is actually on screen.
func (m Model) scrollActiveDashboard(delta int) Model {
	switch m.currentDashboard {
	case views.DashboardNetwork:
		m.networkDashboard.DashboardBase =
			m.networkDashboard.ScrollBy(delta, m.networkDashboard.ContentHeight())
	case views.DashboardSecurity:
		m.securityDashboard.DashboardBase =
			m.securityDashboard.ScrollBy(delta, m.securityDashboard.ContentHeight())
	case views.DashboardVPN:
		m.vpnDashboard.DashboardBase =
			m.vpnDashboard.ScrollBy(delta, m.vpnDashboard.ContentHeight())
	case views.DashboardConfig:
		m.configDashboard.DashboardBase =
			m.configDashboard.ScrollBy(delta, m.configDashboard.ContentHeight())
	default:
		m.dashboard.DashboardBase =
			m.dashboard.ScrollBy(delta, m.dashboard.ContentHeight())
	}
	return m
}

// resetDashboardScroll returns every dashboard to the top. Called when the
// active dashboard changes so a new one never opens mid-scroll.
func (m Model) resetDashboardScroll() Model {
	m.dashboard.Offset = 0
	m.networkDashboard.Offset = 0
	m.securityDashboard.Offset = 0
	m.vpnDashboard.Offset = 0
	m.configDashboard.Offset = 0
	return m
}

// dashboardScrollDelta maps a key press to a scroll distance in lines,
// reporting false when the key is not a scroll key.
func (m Model) dashboardScrollDelta(msg tea.KeyPressMsg) (int, bool) {
	// One screenful minus a line of overlap for context.
	page := max(m.height-5, 1)

	switch msg.String() {
	case "j", "down":
		return 1, true
	case "k", "up":
		return -1, true
	case "pgdown", "ctrl+f":
		return page, true
	case "pgup", "ctrl+b":
		return -page, true
	case "G", "end":
		return scrollJump, true
	case "home":
		return -scrollJump, true
	}
	return 0, false
}
