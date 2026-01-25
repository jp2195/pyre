package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jp2195/pyre/internal/tui/views"
)

func (m Model) renderHeader() string {
	// Left: pyre title + connection status
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
	availableWidth := m.width - leftWidth - tabsWidth - viewWidth - 2

	if availableWidth < 0 {
		availableWidth = 0
	}

	leftPad := availableWidth / 2
	rightPad := availableWidth - leftPad

	row := left + strings.Repeat(" ", leftPad) + tabs + strings.Repeat(" ", rightPad) + viewLabel

	// Build sub-tab row showing items in current group
	subTabs := m.navbar.RenderSubTabs()
	subTabRow := ""
	if subTabs != "" {
		// Center the sub-tabs
		subTabWidth := lipgloss.Width(subTabs)
		subPad := (m.width - subTabWidth) / 2
		if subPad < 0 {
			subPad = 0
		}
		subTabRow = "\n" + strings.Repeat(" ", subPad) + subTabs
	}

	// Add subtle bottom border
	return NavHeaderBorder.Width(m.width).Render(row + subTabRow)
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
	case ViewNATPolicies:
		return "NAT Policies"
	case ViewSessions:
		return "Sessions"
	case ViewInterfaces:
		return "Interfaces"
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

	// Show navigation hint based on current state
	// Get active group key for hint
	group := m.navbar.ActiveGroup()
	var navHint string
	if group != nil {
		navHint = HelpKeyStyle.Render("1-4") + HelpDescStyle.Render(" section") +
			HelpKeyStyle.Render("  "+group.Key) + HelpDescStyle.Render(" cycle")
	} else {
		navHint = HelpKeyStyle.Render("1-4") + HelpDescStyle.Render(" section")
	}

	help := navHint +
		HelpKeyStyle.Render("  Tab") + HelpDescStyle.Render(" next") +
		HelpKeyStyle.Render("  r") + HelpDescStyle.Render(" refresh") +
		HelpKeyStyle.Render("  Ctrl+P") + HelpDescStyle.Render(" search") +
		HelpKeyStyle.Render("  ?") + HelpDescStyle.Render(" help") +
		HelpKeyStyle.Render("  q") + HelpDescStyle.Render(" quit")

	return FooterStyle.Render(help)
}

func (m Model) renderHelp() string {
	return m.help.View(m.keys)
}
