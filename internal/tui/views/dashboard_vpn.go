package views

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

// VPNDashboardModel represents the VPN-focused dashboard
type VPNDashboardModel struct {
	DashboardBase

	tunnels []models.IPSecTunnel
	gpUsers []models.GlobalProtectUser

	tunnelErr error
	gpErr     error
}

// NewVPNDashboardModel creates a new VPN dashboard model
func NewVPNDashboardModel() VPNDashboardModel {
	return VPNDashboardModel{}
}

// SetSpinnerFrame sets the current spinner animation frame
func (m VPNDashboardModel) SetSpinnerFrame(frame string) VPNDashboardModel {
	m.SpinnerFrame = frame
	return m
}

// SetSize sets the terminal dimensions
func (m VPNDashboardModel) SetSize(width, height int) VPNDashboardModel {
	m.Width = width
	m.Height = height
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

// HasData returns true if the dashboard has already loaded its data
func (m VPNDashboardModel) HasData() bool {
	return m.tunnels != nil
}

// View renders the VPN dashboard
func (m VPNDashboardModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	totalWidth, leftColWidth, rightColWidth := m.ColumnWidths()

	if m.IsNarrow() {
		return m.renderSingleColumn(totalWidth)
	}

	// Left column: IPSec tunnels
	leftPanels := []string{
		m.renderIPSecSummary(leftColWidth),
		m.renderIPSecTunnels(leftColWidth),
	}

	// Right column: GlobalProtect users
	rightPanels := []string{
		m.renderGlobalProtectSummary(rightColWidth),
		m.renderGlobalProtectUsers(rightColWidth),
	}

	return m.RenderTwoColumn(leftPanels, rightPanels)
}

func (m VPNDashboardModel) renderSingleColumn(width int) string {
	return m.RenderSingleColumn([]string{
		m.renderIPSecSummary(width),
		m.renderIPSecTunnels(width),
		m.renderGlobalProtectSummary(width),
		m.renderGlobalProtectUsers(width),
	})
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
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
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
	b.WriteString(highlightStyle().Render(strconv.Itoa(upCount)))
	b.WriteString(dimStyle().Render(" up"))

	if downCount > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(errorStyle().Render(strconv.Itoa(downCount)))
		b.WriteString(dimStyle().Render(" down"))
	}

	if initCount > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(warningStyle().Render(strconv.Itoa(initCount)))
		b.WriteString(dimStyle().Render(" init"))
	}

	// Visual bar
	if len(m.tunnels) > 0 {
		b.WriteString("\n")
		barWidth := max(width-8, 10)
		upPct := float64(upCount) / float64(len(m.tunnels)) * 100
		b.WriteString(renderBar(upPct, barWidth, theme.Colors().Success))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m VPNDashboardModel) renderIPSecTunnels(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("IPSec Tunnels"))
	b.WriteString("\n")

	if m.tunnelErr != nil || m.tunnels == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
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

	maxShow := min(len(m.tunnels), 10)

	for i := range maxShow {
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

	b.WriteString("\n")
	b.WriteString(dimStyle().Render("[Tab to IPSec view for details]"))

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
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.gpUsers) == 0 {
		b.WriteString(dimStyle().Render("No active users"))
		return panelStyle().Width(width).Render(b.String())
	}

	// User count
	b.WriteString(highlightStyle().Render(strconv.Itoa(len(m.gpUsers))))
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
			b.WriteString(valueStyle().Render(strconv.Itoa(count)))
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
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
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

	maxShow := min(len(m.gpUsers), 12)

	for i := range maxShow {
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

	b.WriteString("\n")
	b.WriteString(dimStyle().Render("[Tab to GP Users view for details]"))

	return panelStyle().Width(width).Render(b.String())
}
