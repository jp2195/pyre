package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/tui/views"
)

func (m Model) renderHeader() string {
	// Left: pyre title + connection status
	title := HeaderStyle.Render(" pyre ")

	conn := m.session.GetActiveConnection()
	var status string
	if conn != nil {
		statusText := fmt.Sprintf("● %s", conn.Host)
		// Show current target for Panorama connections
		if conn.PanoramaInfo() {
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

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", status)

	// Center: Navigation tabs
	tabs := m.navbar.RenderTabs()

	// Right: Current view name
	viewName := m.currentViewName()
	viewLabel := NavViewLabel.Render(viewName)

	// Calculate spacing to distribute evenly
	leftWidth := lipgloss.Width(left)
	tabsWidth := lipgloss.Width(tabs)
	viewWidth := lipgloss.Width(viewLabel)
	availableWidth := max(m.width-leftWidth-tabsWidth-viewWidth-2, 0)

	leftPad := availableWidth / 2
	rightPad := availableWidth - leftPad

	row := left + strings.Repeat(" ", leftPad) + tabs + strings.Repeat(" ", rightPad) + viewLabel

	// Build sub-tab row showing items in current group
	subTabs := m.navbar.RenderSubTabs()
	subTabRow := ""
	if subTabs != "" {
		// Center the sub-tabs
		subTabWidth := lipgloss.Width(subTabs)
		subPad := max((m.width-subTabWidth)/2, 0)
		subTabRow = "\n" + strings.Repeat(" ", subPad) + subTabs
	}

	// Add subtle bottom border
	return NavHeaderBorder.Width(m.width).Render(row + subTabRow)
}

func (m Model) currentViewName() string {
	switch m.currentView {
	case ViewDashboard:
		// Use breadcrumb format: Monitor/SubView
		switch m.currentDashboard {
		case views.DashboardMain:
			return "Monitor/Overview"
		case views.DashboardNetwork:
			return "Monitor/Network"
		case views.DashboardSecurity:
			return "Monitor/Security"
		case views.DashboardVPN:
			return "Monitor/VPN"
		case views.DashboardConfig:
			return "Tools/Config"
		default:
			return "Monitor/Overview"
		}
	case ViewPolicies:
		return "Analyze/Policies"
	case ViewNATPolicies:
		return "Analyze/NAT"
	case ViewSessions:
		return "Analyze/Sessions"
	case ViewInterfaces:
		return "Analyze/Interfaces"
	case ViewRoutes:
		return "Analyze/Routes"
	case ViewIPSecTunnels:
		return "Analyze/IPSec"
	case ViewGPUsers:
		return "Analyze/GP Users"
	case ViewLogs:
		return "Analyze/Logs"
	case ViewObjects:
		return "Analyze/Objects"
	case ViewPicker:
		return "Connections"
	case ViewDevicePicker:
		return "Connections/Devices"
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

	var sections []string

	// Show error if set
	if m.err != nil {
		errLine := ErrorStyle.Render("Error: " + m.err.Error())
		sections = append(sections, errLine)
	}

	// Show navigation hint based on current state
	// Get active group key for hint
	group := m.navbar.ActiveGroup()
	var navHint string
	if group != nil {
		navHint = views.HelpKeyStyle.Render("1-3") + views.HelpDescStyle.Render(" section") +
			views.HelpKeyStyle.Render("  "+group.Key) + views.HelpDescStyle.Render(" cycle")
	} else {
		navHint = views.HelpKeyStyle.Render("1-3") + views.HelpDescStyle.Render(" section")
	}

	// Show devices hint for Panorama connections
	var devicesHint string
	conn := m.session.GetActiveConnection()
	if conn != nil {
		if conn.PanoramaInfo() {
			devicesHint = views.HelpKeyStyle.Render("  d") + views.HelpDescStyle.Render(" devices")
		}
	}

	// Hints in decreasing order of usefulness, kept only while they fit. The
	// footer used to be a fixed ~95-cell string, and because the screen is
	// joined vertically that one over-long line padded every other line to
	// its width, pushing the whole interface past the right edge of any
	// narrower terminal.
	hints := []struct{ key, desc string }{
		{"  Tab/S-Tab", " next/prev"},
		{"  r", " refresh"},
		{"  :", " conn"},
		{"  Ctrl+P", " commands"},
		{"  ?", " help"},
		{"  q", " quit"},
	}
	help := navHint + devicesHint
	used := lipgloss.Width(help)
	for _, h := range hints {
		w := lipgloss.Width(h.key) + lipgloss.Width(h.desc)
		// FooterStyle adds a cell of padding either side.
		if m.width > 0 && used+w+2 > m.width {
			break
		}
		used += w
		help += views.HelpKeyStyle.Render(h.key) + views.HelpDescStyle.Render(h.desc)
	}

	sections = append(sections, FooterStyle.Render(help))

	return strings.Join(sections, "\n")
}

func (m Model) renderHelp() string {
	return m.help.View(m.keys)
}
