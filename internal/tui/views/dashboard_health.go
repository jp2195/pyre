package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jp2195/pyre/internal/tui/theme"
)

func (m DashboardModel) renderSystemInfo(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("System Information"))
	b.WriteString("\n")

	if m.sysInfoErr != nil {
		b.WriteString(errorStyle().Render("Error: " + m.sysInfoErr.Error()))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.systemInfo == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	si := m.systemInfo

	// Compact device info - hostname and model on first line
	b.WriteString(valueStyle().Render(si.Hostname))
	if si.Model != "" {
		b.WriteString(dimStyle().Render(" • "))
		b.WriteString(labelStyle().Render(si.Model))
	}
	b.WriteString("\n")

	// Serial and version on second line
	if si.Serial != "" {
		b.WriteString(dimStyle().Render("S/N: "))
		b.WriteString(labelStyle().Render(si.Serial))
	}
	if si.Version != "" {
		b.WriteString(dimStyle().Render("  PAN-OS: "))
		b.WriteString(valueStyle().Render(si.Version))
	}
	b.WriteString("\n")

	// IP and uptime on third line
	if si.IPAddress != "" {
		b.WriteString(dimStyle().Render("IP: "))
		b.WriteString(valueStyle().Render(si.IPAddress))
	}
	if si.Uptime != "" {
		b.WriteString(dimStyle().Render("  Up: "))
		b.WriteString(labelStyle().Render(si.Uptime))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderResourcesCompact(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Resources"))
	b.WriteString("\n")

	if m.resourceErr != nil {
		b.WriteString(errorStyle().Render("Error"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.resources == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	barWidth := max(width-20, 10)

	c := theme.Colors()

	// Management CPU (from load average, treated as percentage)
	mgmtCPU := m.resources.ManagementCPU
	mgmtColor := c.Success
	if mgmtCPU > 80 {
		mgmtColor = c.Error
	} else if mgmtCPU > 60 {
		mgmtColor = c.Warning
	}
	b.WriteString(labelStyle().Render("Mgmt"))
	b.WriteString(renderBar(mgmtCPU, barWidth, mgmtColor))
	b.WriteString(fmt.Sprintf(" %4.0f%%\n", mgmtCPU))

	// Dataplane CPU (percentage from resource monitor)
	dpCPU := m.resources.DataPlaneCPU
	dpColor := c.Success
	if dpCPU > 80 {
		dpColor = c.Error
	} else if dpCPU > 60 {
		dpColor = c.Warning
	}
	b.WriteString(labelStyle().Render("DP  "))
	b.WriteString(renderBar(dpCPU, barWidth, dpColor))
	b.WriteString(fmt.Sprintf(" %4.0f%%\n", dpCPU))

	// Memory
	memPct := m.resources.MemoryPercent
	memColor := c.Success
	if memPct > 85 {
		memColor = c.Error
	} else if memPct > 70 {
		memColor = c.Warning
	}
	b.WriteString(labelStyle().Render("Mem "))
	b.WriteString(renderBar(memPct, barWidth, memColor))
	b.WriteString(fmt.Sprintf(" %4.0f%%", memPct))

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderSessionsCompact(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Sessions"))
	b.WriteString("\n")

	if m.sessionErr != nil {
		b.WriteString(errorStyle().Render("Error"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.sessionInfo == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	si := m.sessionInfo

	// Active/Max with utilization bar
	if si.MaxCount > 0 {
		c := theme.Colors()
		sessPct := float64(si.ActiveCount) / float64(si.MaxCount) * 100
		sessColor := c.Success
		if sessPct > 80 {
			sessColor = c.Error
		} else if sessPct > 60 {
			sessColor = c.Warning
		}
		barWidth := max(width-22, 8)
		b.WriteString(renderBar(sessPct, barWidth, sessColor))
		b.WriteString(fmt.Sprintf(" %s/%s\n", formatNumber(int64(si.ActiveCount)), formatNumber(int64(si.MaxCount))))
	}

	// CPS and throughput on one line
	b.WriteString(dimStyle().Render("CPS: "))
	b.WriteString(valueStyle().Render(strconv.Itoa(si.CPS)))
	b.WriteString(dimStyle().Render("  Thru: "))
	b.WriteString(valueStyle().Render(formatThroughput(si.ThroughputKbps)))

	// Protocol breakdown inline (if available)
	if si.TCPSessions > 0 || si.UDPSessions > 0 || si.ICMPSessions > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render("TCP:"))
		b.WriteString(valueStyle().Render(formatNumber(int64(si.TCPSessions))))
		b.WriteString(dimStyle().Render(" UDP:"))
		b.WriteString(valueStyle().Render(formatNumber(int64(si.UDPSessions))))
		b.WriteString(dimStyle().Render(" ICMP:"))
		b.WriteString(valueStyle().Render(formatNumber(int64(si.ICMPSessions))))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderHAStatus(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("HA Status"))
	b.WriteString("\n")

	if m.haErr != nil || m.haStatus == nil || !m.haStatus.Enabled {
		b.WriteString(dimStyle().Render("Not enabled"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Local state with colored indicator
	stateStyle := valueStyle()
	switch m.haStatus.State {
	case "active":
		stateStyle = highlightStyle()
	case "passive":
		stateStyle = warningStyle()
	case "suspended", "initial":
		stateStyle = errorStyle()
	}

	b.WriteString(stateStyle.Render(strings.ToUpper(m.haStatus.State)))
	b.WriteString(dimStyle().Render(" / peer: "))
	b.WriteString(valueStyle().Render(m.haStatus.PeerState))

	if m.haStatus.SyncState != "" {
		syncStyle := dimStyle()
		if m.haStatus.SyncState == "synchronized" {
			syncStyle = highlightStyle()
		}
		b.WriteString("\n")
		b.WriteString(syncStyle.Render(m.haStatus.SyncState))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderDiskUsage(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Disk Usage"))
	b.WriteString("\n")

	if m.diskErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.diskUsage == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.diskUsage) == 0 {
		b.WriteString(dimStyle().Render("No disk info"))
		return panelStyle().Width(width).Render(b.String())
	}

	barWidth := max(width-25, 10)

	c := theme.Colors()

	// Show most relevant filesystems (root, var, etc)
	maxShow := 6
	shown := 0
	totalEligible := 0
	for _, disk := range m.diskUsage {
		if disk.MountPoint == "/dev" || disk.MountPoint == "/run" {
			continue
		}
		totalEligible++

		if shown >= maxShow {
			continue
		}

		// Determine color based on usage
		color := c.Success
		if disk.Percent > 90 {
			color = c.Error
		} else if disk.Percent > 80 {
			color = c.Warning
		}

		mountPoint := disk.MountPoint
		if len(mountPoint) > 12 {
			mountPoint = mountPoint[:10] + ".."
		}

		b.WriteString(labelStyle().Render(fmt.Sprintf("%-12s ", mountPoint)))
		b.WriteString(renderBar(disk.Percent, barWidth, color))
		b.WriteString(fmt.Sprintf(" %3.0f%%\n", disk.Percent))
		shown++
	}

	result := strings.TrimSuffix(b.String(), "\n")
	if totalEligible > maxShow {
		result += "\n" + dimStyle().Render(fmt.Sprintf("... and %d more filesystems", totalEligible-maxShow))
	}

	return panelStyle().Width(width).Render(result)
}

func (m DashboardModel) renderEnvironmentals(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Hardware Status"))
	b.WriteString("\n")

	if m.envErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.environmentals == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.environmentals) == 0 {
		b.WriteString(dimStyle().Render("No sensor data"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count status
	alarmCount := 0
	for _, env := range m.environmentals {
		if env.Alarm {
			alarmCount++
		}
	}

	// Summary
	if alarmCount == 0 {
		b.WriteString(highlightStyle().Render("All sensors normal"))
		b.WriteString("\n")
	} else {
		b.WriteString(errorStyle().Render(fmt.Sprintf("%d alarm(s) active", alarmCount)))
		b.WriteString("\n")
	}

	// Show sensors with issues first, then some normal ones
	maxShow := 5
	shown := 0

	// Alarms first
	for _, env := range m.environmentals {
		if shown >= maxShow {
			break
		}
		if !env.Alarm {
			continue
		}

		statusStyle := errorStyle()
		statusIcon := "!"

		component := truncateEllipsis(env.Component, 18)
		b.WriteString(statusStyle.Render(statusIcon))
		b.WriteString(" ")
		b.WriteString(labelStyle().Render(fmt.Sprintf("%-18s ", component)))
		b.WriteString(statusStyle.Render(env.Value))
		b.WriteString("\n")
		shown++
	}

	// Some normal sensors if space allows
	if shown < maxShow {
		for _, env := range m.environmentals {
			if shown >= maxShow {
				break
			}
			if env.Alarm {
				continue
			}

			component := truncateEllipsis(env.Component, 18)
			b.WriteString(highlightStyle().Render("o"))
			b.WriteString(" ")
			b.WriteString(labelStyle().Render(fmt.Sprintf("%-18s ", component)))
			b.WriteString(dimStyle().Render(env.Value))
			b.WriteString("\n")
			shown++
		}
	}

	result := strings.TrimSuffix(b.String(), "\n")
	if len(m.environmentals) > maxShow {
		result += "\n" + dimStyle().Render(fmt.Sprintf("... and %d more sensors", len(m.environmentals)-maxShow))
	}

	return panelStyle().Width(width).Render(result)
}
