package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jp2195/pyre/internal/models"
)

// VPNDashboardModel represents the VPN-focused dashboard
type VPNDashboardModel struct {
	tunnels []models.IPSecTunnel
	gpUsers []models.GlobalProtectUser

	tunnelErr error
	gpErr     error

	width  int
	height int
}

// NewVPNDashboardModel creates a new VPN dashboard model
func NewVPNDashboardModel() VPNDashboardModel {
	return VPNDashboardModel{}
}

// SetSize sets the terminal dimensions
func (m VPNDashboardModel) SetSize(width, height int) VPNDashboardModel {
	m.width = width
	m.height = height
	return m
}

// SetIPSecTunnels sets the IPSec tunnel data
func (m VPNDashboardModel) SetIPSecTunnels(tunnels []models.IPSecTunnel, err error) VPNDashboardModel {
	m.tunnels = tunnels
	m.tunnelErr = err
	return m
}

// SetGlobalProtectUsers sets the GlobalProtect user data
func (m VPNDashboardModel) SetGlobalProtectUsers(users []models.GlobalProtectUser, err error) VPNDashboardModel {
	m.gpUsers = users
	m.gpErr = err
	return m
}

// Update handles key events
func (m VPNDashboardModel) Update(msg tea.Msg) (VPNDashboardModel, tea.Cmd) {
	return m, nil
}

// View renders the VPN dashboard
func (m VPNDashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	totalWidth := m.width - 4
	leftColWidth := totalWidth / 2
	rightColWidth := totalWidth - leftColWidth - 2

	if leftColWidth < 35 {
		return m.renderSingleColumn(totalWidth)
	}

	// Left column: IPSec tunnels
	leftPanels := []string{
		m.renderIPSecSummary(leftColWidth),
		m.renderIPSecTunnels(leftColWidth),
	}
	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)

	// Right column: GlobalProtect users
	rightPanels := []string{
		m.renderGlobalProtectSummary(rightColWidth),
		m.renderGlobalProtectUsers(rightColWidth),
	}
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
}

func (m VPNDashboardModel) renderSingleColumn(width int) string {
	panels := []string{
		m.renderIPSecSummary(width),
		m.renderIPSecTunnels(width),
		m.renderGlobalProtectSummary(width),
		m.renderGlobalProtectUsers(width),
	}
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

func (m VPNDashboardModel) renderIPSecSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("IPSec VPN Status"))
	b.WriteString("\n")

	if m.tunnelErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.tunnels == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.tunnels) == 0 {
		b.WriteString(dimStyle().Render("No IPSec tunnels configured"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count up/down tunnels
	upCount := 0
	downCount := 0
	initCount := 0
	for _, t := range m.tunnels {
		switch t.State {
		case "up":
			upCount++
		case "init":
			initCount++
		default:
			downCount++
		}
	}

	// Status indicators
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", upCount)))
	b.WriteString(dimStyle().Render(" up"))

	if downCount > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(errorStyle().Render(fmt.Sprintf("%d", downCount)))
		b.WriteString(dimStyle().Render(" down"))
	}

	if initCount > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(warningStyle().Render(fmt.Sprintf("%d", initCount)))
		b.WriteString(dimStyle().Render(" init"))
	}

	// Visual bar
	if len(m.tunnels) > 0 {
		b.WriteString("\n")
		barWidth := width - 8
		if barWidth < 10 {
			barWidth = 10
		}
		upPct := float64(upCount) / float64(len(m.tunnels)) * 100
		b.WriteString(renderBar(upPct, barWidth, "#10B981"))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m VPNDashboardModel) renderIPSecTunnels(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("IPSec Tunnels"))
	b.WriteString("\n")

	if m.tunnelErr != nil || m.tunnels == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.tunnels) == 0 {
		b.WriteString(dimStyle().Render("None"))
		return panelStyle().Width(width).Render(b.String())
	}

	nameWidth := 20
	if width < 60 {
		nameWidth = 15
	}

	maxShow := 10
	if len(m.tunnels) < maxShow {
		maxShow = len(m.tunnels)
	}

	for i := 0; i < maxShow; i++ {
		tunnel := m.tunnels[i]

		// State indicator
		var stateStyle lipgloss.Style
		var stateIcon string
		switch tunnel.State {
		case "up":
			stateStyle = highlightStyle()
			stateIcon = "o"
		case "init":
			stateStyle = warningStyle()
			stateIcon = "~"
		default:
			stateStyle = errorStyle()
			stateIcon = "x"
		}

		name := truncateEllipsis(tunnel.Name, nameWidth)
		b.WriteString(stateStyle.Render(stateIcon))
		b.WriteString(" ")
		b.WriteString(valueStyle().Render(fmt.Sprintf("%-*s ", nameWidth, name)))

		// Gateway
		if tunnel.Gateway != "" {
			gateway := truncateEllipsis(tunnel.Gateway, 15)
			b.WriteString(dimStyle().Render(gateway))
		}

		// Traffic stats if available
		if tunnel.BytesIn > 0 || tunnel.BytesOut > 0 {
			b.WriteString(" ")
			b.WriteString(dimStyle().Render("In:"))
			b.WriteString(labelStyle().Render(formatBytes(tunnel.BytesIn)))
			b.WriteString(dimStyle().Render(" Out:"))
			b.WriteString(labelStyle().Render(formatBytes(tunnel.BytesOut)))
		}

		if i < maxShow-1 {
			b.WriteString("\n")
		}
	}

	if len(m.tunnels) > maxShow {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more", len(m.tunnels)-maxShow)))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m VPNDashboardModel) renderGlobalProtectSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("GlobalProtect Status"))
	b.WriteString("\n")

	if m.gpErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.gpUsers == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.gpUsers) == 0 {
		b.WriteString(dimStyle().Render("No active users"))
		return panelStyle().Width(width).Render(b.String())
	}

	// User count
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", len(m.gpUsers))))
	b.WriteString(dimStyle().Render(" active users"))

	// Count by gateway if available
	gatewayCounts := make(map[string]int)
	for _, user := range m.gpUsers {
		gw := user.Gateway
		if gw == "" {
			gw = "default"
		}
		gatewayCounts[gw]++
	}

	if len(gatewayCounts) > 1 {
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle().Render("By Gateway:"))
		b.WriteString("\n")
		for gw, count := range gatewayCounts {
			gwName := truncateEllipsis(gw, 20)
			b.WriteString(labelStyle().Render(fmt.Sprintf("  %-20s ", gwName)))
			b.WriteString(valueStyle().Render(fmt.Sprintf("%d", count)))
			b.WriteString("\n")
		}
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m VPNDashboardModel) renderGlobalProtectUsers(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("GlobalProtect Users"))
	b.WriteString("\n")

	if m.gpErr != nil || m.gpUsers == nil {
		b.WriteString(dimStyle().Render("Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.gpUsers) == 0 {
		b.WriteString(dimStyle().Render("None"))
		return panelStyle().Width(width).Render(b.String())
	}

	userWidth := 15
	ipWidth := 15
	if width < 60 {
		userWidth = 12
		ipWidth = 12
	}

	maxShow := 12
	if len(m.gpUsers) < maxShow {
		maxShow = len(m.gpUsers)
	}

	for i := 0; i < maxShow; i++ {
		user := m.gpUsers[i]

		username := truncateEllipsis(user.Username, userWidth)
		b.WriteString(valueStyle().Render(fmt.Sprintf("%-*s ", userWidth, username)))

		if user.VirtualIP != "" {
			vip := truncateEllipsis(user.VirtualIP, ipWidth)
			b.WriteString(accentStyle().Render(fmt.Sprintf("%-*s ", ipWidth, vip)))
		}

		if user.Duration != "" {
			b.WriteString(dimStyle().Render(user.Duration))
		} else if !user.LoginTime.IsZero() {
			b.WriteString(dimStyle().Render(formatTimeAgo(user.LoginTime)))
		}

		if i < maxShow-1 {
			b.WriteString("\n")
		}
	}

	if len(m.gpUsers) > maxShow {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more", len(m.gpUsers)-maxShow)))
	}

	return panelStyle().Width(width).Render(b.String())
}
