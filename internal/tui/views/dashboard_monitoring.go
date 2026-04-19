package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/tui/theme"
)

func (m DashboardModel) renderContentVersions(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Content Versions"))
	b.WriteString("\n")

	if m.systemInfo == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	si := m.systemInfo

	// Compact inline format
	type versionInfo struct {
		abbrev  string
		version string
	}

	versions := []versionInfo{
		{"App", si.AppVersion},
		{"Threat", si.ThreatVersion},
		{"AV", si.AntivirusVersion},
		{"WF", si.WildFireVersion},
		{"URL", si.URLFilteringVersion},
	}

	hasContent := false
	for _, v := range versions {
		if v.version != "" {
			hasContent = true
			b.WriteString(labelStyle().Render(v.abbrev + ": "))
			b.WriteString(valueStyle().Render(v.version))
			b.WriteString("\n")
		}
	}

	if !hasContent {
		b.WriteString(dimStyle().Render("No content info"))
	}

	return panelStyle().Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m DashboardModel) renderGlobalProtect(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("GlobalProtect"))
	b.WriteString("\n")

	if m.gpErr != nil || m.gpInfo == nil {
		b.WriteString(dimStyle().Render("Not configured"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Users and gateways on one line
	b.WriteString(dimStyle().Render("Users: "))
	if m.gpInfo.ActiveUsers > 0 {
		b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", m.gpInfo.ActiveUsers)))
	} else {
		b.WriteString(dimStyle().Render("0"))
	}
	b.WriteString(dimStyle().Render("  GW: "))
	b.WriteString(valueStyle().Render(fmt.Sprintf("%d/%d", m.gpInfo.ActiveGateways, m.gpInfo.TotalGateways)))

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderLicenses(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Licenses"))
	b.WriteString("\n")

	if m.licenseErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.licenses == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.licenses) == 0 {
		b.WriteString(dimStyle().Render("No licenses"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count status
	var valid, expiring, expired int
	for _, lic := range m.licenses {
		if lic.Expired {
			expired++
		} else if lic.DaysLeft > 0 && lic.DaysLeft < 60 {
			expiring++
		} else {
			valid++
		}
	}

	// Summary line
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", valid)))
	b.WriteString(dimStyle().Render(" valid"))
	if expiring > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(warningStyle().Render(fmt.Sprintf("%d", expiring)))
		b.WriteString(dimStyle().Render(" expiring"))
	}
	if expired > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(errorStyle().Render(fmt.Sprintf("%d", expired)))
		b.WriteString(dimStyle().Render(" expired"))
	}

	// Show licenses needing attention (expiring or expired), max 3
	var attention []models.LicenseInfo
	for _, lic := range m.licenses {
		if lic.Expired || (lic.DaysLeft > 0 && lic.DaysLeft < 60) {
			attention = append(attention, lic)
		}
	}

	if len(attention) > 0 {
		b.WriteString("\n")
		maxShow := 3
		for i, lic := range attention {
			if i >= maxShow {
				break
			}
			// Truncate feature name
			name := lic.Feature
			if len(name) > 18 {
				name = name[:15] + "..."
			}

			var statusStyle lipgloss.Style
			var days string
			if lic.Expired {
				statusStyle = errorStyle()
				days = "exp"
			} else {
				if lic.DaysLeft < 30 {
					statusStyle = SeverityHighStyle
				} else {
					statusStyle = warningStyle()
				}
				days = fmt.Sprintf("%dd", lic.DaysLeft)
			}
			b.WriteString(statusStyle.Render(days))
			b.WriteString(dimStyle().Render(" "))
			b.WriteString(labelStyle().Render(name))
			if i < len(attention)-1 && i < maxShow-1 {
				b.WriteString("\n")
			}
		}
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderLoggedInAdmins(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Admins Online"))
	b.WriteString("\n")

	if m.adminErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.admins == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.admins) == 0 {
		b.WriteString(dimStyle().Render("None"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Show up to 6 admins, single line each
	maxAdmins := 6
	for i, admin := range m.admins {
		if i >= maxAdmins {
			break
		}

		// Format admin type with style
		typeStyle := dimStyle()
		switch admin.Type {
		case "Web":
			typeStyle = accentStyle()
		case "CLI":
			typeStyle = highlightStyle()
		case "API":
			typeStyle = warningStyle()
		}

		// Single line: username (type) from IP
		b.WriteString(valueStyle().Render(admin.Username))
		b.WriteString(typeStyle.Render(" " + admin.Type))
		b.WriteString(dimStyle().Render(" " + admin.From))

		if i < len(m.admins)-1 && i < maxAdmins-1 {
			b.WriteString("\n")
		}
	}

	if len(m.admins) > maxAdmins {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more admins", len(m.admins)-maxAdmins)))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderJobs(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Recent Jobs"))
	b.WriteString("\n")

	if m.jobErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.jobs == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.jobs) == 0 {
		b.WriteString(dimStyle().Render("None"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Show up to 6 most recent jobs
	maxJobs := 6
	for i, job := range m.jobs {
		if i >= maxJobs {
			break
		}

		// Status indicator
		var statusStyle lipgloss.Style
		var statusIndicator string

		switch job.Status {
		case "FIN":
			if job.Result == "OK" {
				statusStyle = highlightStyle()
				statusIndicator = "✓"
			} else {
				statusStyle = errorStyle()
				statusIndicator = "✗"
			}
		case "ACT":
			statusStyle = accentStyle()
			statusIndicator = "●"
		case "PEND":
			statusStyle = warningStyle()
			statusIndicator = "○"
		default:
			statusStyle = dimStyle()
			statusIndicator = "?"
		}

		// Job type (truncated if needed)
		jobType := job.Type
		if len(jobType) > 10 {
			jobType = jobType[:8] + ".."
		}

		b.WriteString(statusStyle.Render(statusIndicator))
		b.WriteString(valueStyle().Render(" " + jobType))
		b.WriteString(dimStyle().Render(fmt.Sprintf(" %d", job.ID)))

		if i < len(m.jobs)-1 && i < maxJobs-1 {
			b.WriteString("\n")
		}
	}

	if len(m.jobs) > maxJobs {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more jobs", len(m.jobs)-maxJobs)))
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderThreatSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Threats"))
	b.WriteString("\n")

	if m.threatErr != nil || m.threatSummary == nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}

	ts := m.threatSummary

	if ts.TotalThreats == 0 {
		b.WriteString(highlightStyle().Render("None detected"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Color-coded severity on one line
	criticalStyle := SeverityCriticalStyle
	highStyle := SeverityHighStyle
	mediumStyle := SeverityMediumStyle
	lowStyle := SeverityLowStyle

	if ts.CriticalCount > 0 {
		b.WriteString(criticalStyle.Render(fmt.Sprintf("C:%d ", ts.CriticalCount)))
	}
	if ts.HighCount > 0 {
		b.WriteString(highStyle.Render(fmt.Sprintf("H:%d ", ts.HighCount)))
	}
	if ts.MediumCount > 0 {
		b.WriteString(mediumStyle.Render(fmt.Sprintf("M:%d ", ts.MediumCount)))
	}
	if ts.LowCount > 0 {
		b.WriteString(lowStyle.Render(fmt.Sprintf("L:%d", ts.LowCount)))
	}

	// Actions on second line
	b.WriteString("\n")
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", ts.BlockedCount)))
	b.WriteString(dimStyle().Render(" blocked  "))
	b.WriteString(warningStyle().Render(fmt.Sprintf("%d", ts.AlertedCount)))
	b.WriteString(dimStyle().Render(" alerted"))

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderCertificates(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("Certificates"))
	b.WriteString("\n")

	if m.certErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.certificates == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.certificates) == 0 {
		b.WriteString(dimStyle().Render("No certificates"))
		return panelStyle().Width(width).Render(b.String())
	}

	// Count by status
	var expired, expiring, valid int
	for _, cert := range m.certificates {
		switch cert.Status {
		case "expired":
			expired++
		case "expiring":
			expiring++
		default:
			valid++
		}
	}

	// Summary
	b.WriteString(highlightStyle().Render(fmt.Sprintf("%d", valid)))
	b.WriteString(dimStyle().Render(" valid"))
	if expiring > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(warningStyle().Render(fmt.Sprintf("%d", expiring)))
		b.WriteString(dimStyle().Render(" expiring"))
	}
	if expired > 0 {
		b.WriteString(dimStyle().Render("  "))
		b.WriteString(errorStyle().Render(fmt.Sprintf("%d", expired)))
		b.WriteString(dimStyle().Render(" expired"))
	}

	// Show certs needing attention
	var attention []models.Certificate
	for _, cert := range m.certificates {
		if cert.Status == "expired" || cert.Status == "expiring" {
			attention = append(attention, cert)
		}
	}

	if len(attention) > 0 {
		b.WriteString("\n\n")
		maxShow := 6
		for i, cert := range attention {
			if i >= maxShow {
				break
			}

			name := truncateEllipsis(cert.Name, 18)
			var statusStyle lipgloss.Style
			var days string

			if cert.Status == "expired" {
				statusStyle = errorStyle()
				days = "exp"
			} else {
				if cert.DaysLeft < 14 {
					statusStyle = SeverityHighStyle
				} else {
					statusStyle = warningStyle()
				}
				days = fmt.Sprintf("%dd", cert.DaysLeft)
			}

			b.WriteString(statusStyle.Render(fmt.Sprintf("%-4s ", days)))
			b.WriteString(labelStyle().Render(name))
			if i < len(attention)-1 && i < maxShow-1 {
				b.WriteString("\n")
			}
		}

		if len(attention) > maxShow {
			b.WriteString("\n")
			b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more certificates", len(attention)-maxShow)))
		}
	}

	return panelStyle().Width(width).Render(b.String())
}

func (m DashboardModel) renderNATPoolUtilization(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle().Render("NAT Pool Utilization"))
	b.WriteString("\n")

	if m.natPoolErr != nil {
		b.WriteString(dimStyle().Render("Not available"))
		return panelStyle().Width(width).Render(b.String())
	}
	if m.natPools == nil {
		b.WriteString(RenderLoadingInline(m.SpinnerFrame, "Loading..."))
		return panelStyle().Width(width).Render(b.String())
	}

	if len(m.natPools) == 0 {
		b.WriteString(dimStyle().Render("No NAT pools configured"))
		return panelStyle().Width(width).Render(b.String())
	}

	barWidth := max(width-28, 8)

	c := theme.Colors()

	// Show up to 5 pools
	maxShow := 5
	for i, pool := range m.natPools {
		if i >= maxShow {
			break
		}

		// Truncate pool name to 15 chars
		name := pool.RuleName
		if len(name) > 15 {
			name = name[:12] + "..."
		}

		// Color based on utilization
		poolColor := c.Success
		if pool.Percent > 80 {
			poolColor = c.Error
		} else if pool.Percent > 60 {
			poolColor = c.Warning
		}

		b.WriteString(labelStyle().Render(fmt.Sprintf("%-15s ", name)))
		b.WriteString(renderBar(pool.Percent, barWidth, poolColor))
		b.WriteString(fmt.Sprintf(" %3.0f%%", pool.Percent))

		if i < len(m.natPools)-1 && i < maxShow-1 {
			b.WriteString("\n")
		}
	}

	if len(m.natPools) > maxShow {
		b.WriteString("\n")
		b.WriteString(dimStyle().Render(fmt.Sprintf("... and %d more", len(m.natPools)-maxShow)))
	}

	return panelStyle().Width(width).Render(b.String())
}
