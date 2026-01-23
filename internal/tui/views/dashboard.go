package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/joshuamontgomery/pyre/internal/models"
)

// DashboardType represents the type of dashboard to display
type DashboardType int

const (
	DashboardMain DashboardType = iota
	DashboardNetwork
	DashboardSecurity
	DashboardVPN
	DashboardConfig
)

// DashboardName returns the display name for a dashboard type
func DashboardName(dt DashboardType) string {
	names := map[DashboardType]string{
		DashboardMain:     "Main",
		DashboardNetwork:  "Network",
		DashboardSecurity: "Security",
		DashboardVPN:      "VPN",
		DashboardConfig:   "Config",
	}
	if name, ok := names[dt]; ok {
		return name
	}
	return "Main"
}

type DashboardModel struct {
	systemInfo     *models.SystemInfo
	resources      *models.Resources
	sessionInfo    *models.SessionInfo
	haStatus       *models.HAStatus
	interfaces     []models.Interface
	threatSummary  *models.ThreatSummary
	gpInfo         *models.GlobalProtectInfo
	admins         []models.LoggedInAdmin
	licenses       []models.LicenseInfo
	jobs           []models.Job
	diskUsage      []models.DiskUsage
	environmentals []models.Environmental
	certificates   []models.Certificate

	sysInfoErr  error
	resourceErr error
	sessionErr  error
	haErr       error
	ifaceErr    error
	threatErr   error
	gpErr       error
	adminErr    error
	licenseErr  error
	jobErr      error
	diskErr     error
	envErr      error
	certErr     error

	width  int
	height int
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{}
}

func (m DashboardModel) SetSize(width, height int) DashboardModel {
	m.width = width
	m.height = height
	return m
}

func (m DashboardModel) SetSystemInfo(info *models.SystemInfo, err error) DashboardModel {
	m.systemInfo = info
	m.sysInfoErr = err
	return m
}

func (m DashboardModel) SetResources(res *models.Resources, err error) DashboardModel {
	m.resources = res
	m.resourceErr = err
	return m
}

func (m DashboardModel) SetSessionInfo(info *models.SessionInfo, err error) DashboardModel {
	m.sessionInfo = info
	m.sessionErr = err
	return m
}

func (m DashboardModel) SetHAStatus(status *models.HAStatus, err error) DashboardModel {
	m.haStatus = status
	m.haErr = err
	return m
}

func (m DashboardModel) SetInterfaces(ifaces []models.Interface, err error) DashboardModel {
	m.interfaces = ifaces
	m.ifaceErr = err
	return m
}

func (m DashboardModel) SetThreatSummary(summary *models.ThreatSummary, err error) DashboardModel {
	m.threatSummary = summary
	m.threatErr = err
	return m
}

func (m DashboardModel) SetGlobalProtectInfo(info *models.GlobalProtectInfo, err error) DashboardModel {
	m.gpInfo = info
	m.gpErr = err
	return m
}

func (m DashboardModel) SetLoggedInAdmins(admins []models.LoggedInAdmin, err error) DashboardModel {
	m.admins = admins
	m.adminErr = err
	return m
}

func (m DashboardModel) SetLicenses(licenses []models.LicenseInfo, err error) DashboardModel {
	m.licenses = licenses
	m.licenseErr = err
	return m
}

func (m DashboardModel) SetJobs(jobs []models.Job, err error) DashboardModel {
	m.jobs = jobs
	m.jobErr = err
	return m
}

func (m DashboardModel) SetDiskUsage(disks []models.DiskUsage, err error) DashboardModel {
	m.diskUsage = disks
	m.diskErr = err
	return m
}

func (m DashboardModel) SetEnvironmentals(envs []models.Environmental, err error) DashboardModel {
	m.environmentals = envs
	m.envErr = err
	return m
}

func (m DashboardModel) SetCertificates(certs []models.Certificate, err error) DashboardModel {
	m.certificates = certs
	m.certErr = err
	return m
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	return m, nil
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate column widths - 50/50 split
	totalWidth := m.width - 4
	leftColWidth := totalWidth / 2
	rightColWidth := totalWidth - leftColWidth - 2

	if leftColWidth < 35 {
		// Single column layout for narrow terminals
		return m.renderSingleColumn(totalWidth)
	}

	// Build left column - system info and related
	leftPanels := []string{
		m.renderSystemInfo(leftColWidth),
		m.renderResourcesCompact(leftColWidth),
		m.renderSessionsCompact(leftColWidth),
	}

	// Conditionally add HA panel if HA is enabled
	if m.haStatus != nil && m.haStatus.Enabled {
		leftPanels = append(leftPanels, m.renderHAStatus(leftColWidth))
	}

	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftPanels...)

	// Build right column - versions, licenses, jobs, etc
	rightPanels := []string{
		m.renderContentVersions(rightColWidth),
	}

	// Conditionally add licenses panel if we have data
	if m.licenses != nil && len(m.licenses) > 0 {
		rightPanels = append(rightPanels, m.renderLicenses(rightColWidth))
	}

	// Conditionally add admins panel if we have data
	if m.admins != nil && len(m.admins) > 0 {
		rightPanels = append(rightPanels, m.renderLoggedInAdmins(rightColWidth))
	}

	// Conditionally add GlobalProtect panel if configured
	if m.gpInfo != nil && (m.gpInfo.ActiveUsers > 0 || m.gpInfo.TotalGateways > 0) {
		rightPanels = append(rightPanels, m.renderGlobalProtect(rightColWidth))
	}

	// Conditionally add jobs panel if we have jobs
	if m.jobs != nil && len(m.jobs) > 0 {
		rightPanels = append(rightPanels, m.renderJobs(rightColWidth))
	}

	// Conditionally add threat summary if we have threat data
	if m.threatSummary != nil && m.threatSummary.TotalThreats > 0 {
		rightPanels = append(rightPanels, m.renderThreatSummary(rightColWidth))
	}

	// Conditionally add disk usage panel
	if m.diskUsage != nil && len(m.diskUsage) > 0 {
		rightPanels = append(rightPanels, m.renderDiskUsage(rightColWidth))
	}

	// Conditionally add environmentals panel if we have data with issues
	if m.environmentals != nil && len(m.environmentals) > 0 {
		rightPanels = append(rightPanels, m.renderEnvironmentals(rightColWidth))
	}

	// Conditionally add certificates panel if we have expiring certs
	if m.certificates != nil && len(m.certificates) > 0 {
		rightPanels = append(rightPanels, m.renderCertificates(rightColWidth))
	}

	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightPanels...)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
}

func (m DashboardModel) renderSingleColumn(width int) string {
	panels := []string{
		m.renderSystemInfo(width),
		m.renderResourcesCompact(width),
		m.renderSessionsCompact(width),
	}

	if m.haStatus != nil && m.haStatus.Enabled {
		panels = append(panels, m.renderHAStatus(width))
	}

	if m.licenses != nil && len(m.licenses) > 0 {
		panels = append(panels, m.renderLicenses(width))
	}

	if m.jobs != nil && len(m.jobs) > 0 {
		panels = append(panels, m.renderJobs(width))
	}

	if m.diskUsage != nil && len(m.diskUsage) > 0 {
		panels = append(panels, m.renderDiskUsage(width))
	}

	if m.environmentals != nil && len(m.environmentals) > 0 {
		panels = append(panels, m.renderEnvironmentals(width))
	}

	if m.certificates != nil && len(m.certificates) > 0 {
		panels = append(panels, m.renderCertificates(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

// Styles
var (
	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A78BFA"))

	subtitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	valueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB"))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	highlightStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	accentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA"))
)

func (m DashboardModel) renderSystemInfo(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("System Information"))
	b.WriteString("\n")

	if m.sysInfoErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.sysInfoErr.Error()))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.systemInfo == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	si := m.systemInfo

	// Compact device info - hostname and model on first line
	b.WriteString(valueStyle.Render(si.Hostname))
	if si.Model != "" {
		b.WriteString(dimStyle.Render(" • "))
		b.WriteString(labelStyle.Render(si.Model))
	}
	b.WriteString("\n")

	// Serial and version on second line
	if si.Serial != "" {
		b.WriteString(dimStyle.Render("S/N: "))
		b.WriteString(labelStyle.Render(si.Serial))
	}
	if si.Version != "" {
		b.WriteString(dimStyle.Render("  PAN-OS: "))
		b.WriteString(valueStyle.Render(si.Version))
	}
	b.WriteString("\n")

	// IP and uptime on third line
	if si.IPAddress != "" {
		b.WriteString(dimStyle.Render("IP: "))
		b.WriteString(valueStyle.Render(si.IPAddress))
	}
	if si.Uptime != "" {
		b.WriteString(dimStyle.Render("  Up: "))
		b.WriteString(labelStyle.Render(si.Uptime))
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderContentVersions(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Content Versions"))
	b.WriteString("\n")

	if m.systemInfo == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
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
			b.WriteString(labelStyle.Render(v.abbrev + ": "))
			b.WriteString(valueStyle.Render(v.version))
			b.WriteString("\n")
		}
	}

	if !hasContent {
		b.WriteString(dimStyle.Render("No content info"))
	}

	return panelStyle.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m DashboardModel) renderResourcesCompact(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Resources"))
	b.WriteString("\n")

	if m.resourceErr != nil {
		b.WriteString(errorStyle.Render("Error"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.resources == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	barWidth := width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	// CPU
	cpuPct := m.resources.CPUPercent
	cpuColor := "#10B981"
	if cpuPct > 80 {
		cpuColor = "#EF4444"
	} else if cpuPct > 60 {
		cpuColor = "#F59E0B"
	}
	b.WriteString(labelStyle.Render("CPU "))
	b.WriteString(renderBar(cpuPct, barWidth, cpuColor))
	b.WriteString(fmt.Sprintf(" %4.0f%%\n", cpuPct))

	// Memory
	memPct := m.resources.MemoryPercent
	memColor := "#10B981"
	if memPct > 85 {
		memColor = "#EF4444"
	} else if memPct > 70 {
		memColor = "#F59E0B"
	}
	b.WriteString(labelStyle.Render("Mem "))
	b.WriteString(renderBar(memPct, barWidth, memColor))
	b.WriteString(fmt.Sprintf(" %4.0f%%", memPct))

	// Load average on same line as session bar if available
	if m.resources.Load1 > 0 || m.resources.Load5 > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Load: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%.1f %.1f %.1f", m.resources.Load1, m.resources.Load5, m.resources.Load15)))
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderSessionsCompact(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sessions"))
	b.WriteString("\n")

	if m.sessionErr != nil {
		b.WriteString(errorStyle.Render("Error"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.sessionInfo == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	si := m.sessionInfo

	// Active/Max with utilization bar
	if si.MaxCount > 0 {
		sessPct := float64(si.ActiveCount) / float64(si.MaxCount) * 100
		sessColor := "#10B981"
		if sessPct > 80 {
			sessColor = "#EF4444"
		} else if sessPct > 60 {
			sessColor = "#F59E0B"
		}
		barWidth := width - 22
		if barWidth < 8 {
			barWidth = 8
		}
		b.WriteString(renderBar(sessPct, barWidth, sessColor))
		b.WriteString(fmt.Sprintf(" %s/%s\n", formatNumber(int64(si.ActiveCount)), formatNumber(int64(si.MaxCount))))
	}

	// CPS and throughput on one line
	b.WriteString(dimStyle.Render("CPS: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", si.CPS)))
	b.WriteString(dimStyle.Render("  Thru: "))
	b.WriteString(valueStyle.Render(formatThroughput(si.ThroughputKbps)))

	// Protocol breakdown inline (if available)
	if si.TCPSessions > 0 || si.UDPSessions > 0 || si.ICMPSessions > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("TCP:"))
		b.WriteString(valueStyle.Render(formatNumber(int64(si.TCPSessions))))
		b.WriteString(dimStyle.Render(" UDP:"))
		b.WriteString(valueStyle.Render(formatNumber(int64(si.UDPSessions))))
		b.WriteString(dimStyle.Render(" ICMP:"))
		b.WriteString(valueStyle.Render(formatNumber(int64(si.ICMPSessions))))
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderHAStatus(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("HA Status"))
	b.WriteString("\n")

	if m.haErr != nil || m.haStatus == nil || !m.haStatus.Enabled {
		b.WriteString(dimStyle.Render("Not enabled"))
		return panelStyle.Width(width).Render(b.String())
	}

	// Local state with colored indicator
	stateStyle := valueStyle
	switch m.haStatus.State {
	case "active":
		stateStyle = highlightStyle
	case "passive":
		stateStyle = warningStyle
	case "suspended", "initial":
		stateStyle = errorStyle
	}

	b.WriteString(stateStyle.Render(strings.ToUpper(m.haStatus.State)))
	b.WriteString(dimStyle.Render(" / peer: "))
	b.WriteString(valueStyle.Render(m.haStatus.PeerState))

	if m.haStatus.SyncState != "" {
		syncStyle := dimStyle
		if m.haStatus.SyncState == "synchronized" {
			syncStyle = highlightStyle
		}
		b.WriteString("\n")
		b.WriteString(syncStyle.Render(m.haStatus.SyncState))
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderGlobalProtect(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("GlobalProtect"))
	b.WriteString("\n")

	if m.gpErr != nil || m.gpInfo == nil {
		b.WriteString(dimStyle.Render("Not configured"))
		return panelStyle.Width(width).Render(b.String())
	}

	// Users and gateways on one line
	b.WriteString(dimStyle.Render("Users: "))
	if m.gpInfo.ActiveUsers > 0 {
		b.WriteString(highlightStyle.Render(fmt.Sprintf("%d", m.gpInfo.ActiveUsers)))
	} else {
		b.WriteString(dimStyle.Render("0"))
	}
	b.WriteString(dimStyle.Render("  GW: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d/%d", m.gpInfo.ActiveGateways, m.gpInfo.TotalGateways)))

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderLicenses(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Licenses"))
	b.WriteString("\n")

	if m.licenseErr != nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.licenses == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.licenses) == 0 {
		b.WriteString(dimStyle.Render("No licenses"))
		return panelStyle.Width(width).Render(b.String())
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
	b.WriteString(highlightStyle.Render(fmt.Sprintf("%d", valid)))
	b.WriteString(dimStyle.Render(" valid"))
	if expiring > 0 {
		b.WriteString(dimStyle.Render("  "))
		b.WriteString(warningStyle.Render(fmt.Sprintf("%d", expiring)))
		b.WriteString(dimStyle.Render(" expiring"))
	}
	if expired > 0 {
		b.WriteString(dimStyle.Render("  "))
		b.WriteString(errorStyle.Render(fmt.Sprintf("%d", expired)))
		b.WriteString(dimStyle.Render(" expired"))
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
				statusStyle = errorStyle
				days = "exp"
			} else {
				if lic.DaysLeft < 30 {
					statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
				} else {
					statusStyle = warningStyle
				}
				days = fmt.Sprintf("%dd", lic.DaysLeft)
			}
			b.WriteString(statusStyle.Render(days))
			b.WriteString(dimStyle.Render(" "))
			b.WriteString(labelStyle.Render(name))
			if i < len(attention)-1 && i < maxShow-1 {
				b.WriteString("\n")
			}
		}
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderLoggedInAdmins(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Admins Online"))
	b.WriteString("\n")

	if m.adminErr != nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.admins == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.admins) == 0 {
		b.WriteString(dimStyle.Render("None"))
		return panelStyle.Width(width).Render(b.String())
	}

	// Show up to 4 admins, single line each
	maxAdmins := 4
	for i, admin := range m.admins {
		if i >= maxAdmins {
			break
		}

		// Format admin type with style
		typeStyle := dimStyle
		switch admin.Type {
		case "Web":
			typeStyle = accentStyle
		case "CLI":
			typeStyle = highlightStyle
		case "API":
			typeStyle = warningStyle
		}

		// Single line: username (type) from IP
		b.WriteString(valueStyle.Render(admin.Username))
		b.WriteString(typeStyle.Render(" " + admin.Type))
		b.WriteString(dimStyle.Render(" " + admin.From))

		if i < len(m.admins)-1 && i < maxAdmins-1 {
			b.WriteString("\n")
		}
	}

	if len(m.admins) > maxAdmins {
		b.WriteString(dimStyle.Render(fmt.Sprintf(" +%d", len(m.admins)-maxAdmins)))
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderJobs(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Recent Jobs"))
	b.WriteString("\n")

	if m.jobErr != nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.jobs == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.jobs) == 0 {
		b.WriteString(dimStyle.Render("None"))
		return panelStyle.Width(width).Render(b.String())
	}

	// Show up to 4 most recent jobs
	maxJobs := 4
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
				statusStyle = highlightStyle
				statusIndicator = "✓"
			} else {
				statusStyle = errorStyle
				statusIndicator = "✗"
			}
		case "ACT":
			statusStyle = accentStyle
			statusIndicator = "●"
		case "PEND":
			statusStyle = warningStyle
			statusIndicator = "○"
		default:
			statusStyle = dimStyle
			statusIndicator = "?"
		}

		// Job type (truncated if needed)
		jobType := job.Type
		if len(jobType) > 10 {
			jobType = jobType[:8] + ".."
		}

		b.WriteString(statusStyle.Render(statusIndicator))
		b.WriteString(valueStyle.Render(" " + jobType))
		b.WriteString(dimStyle.Render(fmt.Sprintf(" %d", job.ID)))

		if i < len(m.jobs)-1 && i < maxJobs-1 {
			b.WriteString("\n")
		}
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderThreatSummary(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Threats"))
	b.WriteString("\n")

	if m.threatErr != nil || m.threatSummary == nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}

	ts := m.threatSummary

	if ts.TotalThreats == 0 {
		b.WriteString(highlightStyle.Render("None detected"))
		return panelStyle.Width(width).Render(b.String())
	}

	// Color-coded severity on one line
	criticalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	highStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
	mediumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308"))
	lowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

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
	b.WriteString(highlightStyle.Render(fmt.Sprintf("%d", ts.BlockedCount)))
	b.WriteString(dimStyle.Render(" blocked  "))
	b.WriteString(warningStyle.Render(fmt.Sprintf("%d", ts.AlertedCount)))
	b.WriteString(dimStyle.Render(" alerted"))

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderDiskUsage(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Disk Usage"))
	b.WriteString("\n")

	if m.diskErr != nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.diskUsage == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.diskUsage) == 0 {
		b.WriteString(dimStyle.Render("No disk info"))
		return panelStyle.Width(width).Render(b.String())
	}

	barWidth := width - 25
	if barWidth < 10 {
		barWidth = 10
	}

	// Show most relevant filesystems (root, var, etc)
	maxShow := 4
	shown := 0
	for _, disk := range m.diskUsage {
		if shown >= maxShow {
			break
		}

		// Skip some system filesystems
		if disk.MountPoint == "/dev" || disk.MountPoint == "/run" {
			continue
		}

		// Determine color based on usage
		color := "#10B981"
		if disk.Percent > 90 {
			color = "#EF4444"
		} else if disk.Percent > 80 {
			color = "#F59E0B"
		}

		mountPoint := disk.MountPoint
		if len(mountPoint) > 12 {
			mountPoint = mountPoint[:10] + ".."
		}

		b.WriteString(labelStyle.Render(fmt.Sprintf("%-12s ", mountPoint)))
		b.WriteString(renderBar(disk.Percent, barWidth, color))
		b.WriteString(fmt.Sprintf(" %3.0f%%\n", disk.Percent))
		shown++
	}

	return panelStyle.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m DashboardModel) renderEnvironmentals(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Hardware Status"))
	b.WriteString("\n")

	if m.envErr != nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.environmentals == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.environmentals) == 0 {
		b.WriteString(dimStyle.Render("No sensor data"))
		return panelStyle.Width(width).Render(b.String())
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
		b.WriteString(highlightStyle.Render("All sensors normal"))
		b.WriteString("\n")
	} else {
		b.WriteString(errorStyle.Render(fmt.Sprintf("%d alarm(s) active", alarmCount)))
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

		statusStyle := errorStyle
		statusIcon := "!"

		component := truncateEllipsis(env.Component, 18)
		b.WriteString(statusStyle.Render(statusIcon))
		b.WriteString(" ")
		b.WriteString(labelStyle.Render(fmt.Sprintf("%-18s ", component)))
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
			b.WriteString(highlightStyle.Render("o"))
			b.WriteString(" ")
			b.WriteString(labelStyle.Render(fmt.Sprintf("%-18s ", component)))
			b.WriteString(dimStyle.Render(env.Value))
			b.WriteString("\n")
			shown++
		}
	}

	return panelStyle.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m DashboardModel) renderCertificates(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Certificates"))
	b.WriteString("\n")

	if m.certErr != nil {
		b.WriteString(dimStyle.Render("Not available"))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.certificates == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.certificates) == 0 {
		b.WriteString(dimStyle.Render("No certificates"))
		return panelStyle.Width(width).Render(b.String())
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
	b.WriteString(highlightStyle.Render(fmt.Sprintf("%d", valid)))
	b.WriteString(dimStyle.Render(" valid"))
	if expiring > 0 {
		b.WriteString(dimStyle.Render("  "))
		b.WriteString(warningStyle.Render(fmt.Sprintf("%d", expiring)))
		b.WriteString(dimStyle.Render(" expiring"))
	}
	if expired > 0 {
		b.WriteString(dimStyle.Render("  "))
		b.WriteString(errorStyle.Render(fmt.Sprintf("%d", expired)))
		b.WriteString(dimStyle.Render(" expired"))
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
		maxShow := 4
		for i, cert := range attention {
			if i >= maxShow {
				break
			}

			name := truncateEllipsis(cert.Name, 18)
			var statusStyle lipgloss.Style
			var days string

			if cert.Status == "expired" {
				statusStyle = errorStyle
				days = "exp"
			} else {
				if cert.DaysLeft < 14 {
					statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
				} else {
					statusStyle = warningStyle
				}
				days = fmt.Sprintf("%dd", cert.DaysLeft)
			}

			b.WriteString(statusStyle.Render(fmt.Sprintf("%-4s ", days)))
			b.WriteString(labelStyle.Render(name))
			if i < len(attention)-1 && i < maxShow-1 {
				b.WriteString("\n")
			}
		}
	}

	return panelStyle.Width(width).Render(b.String())
}

func (m DashboardModel) renderInterfaces(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Network Interfaces"))
	b.WriteString("\n\n")

	if m.ifaceErr != nil {
		b.WriteString(errorStyle.Render("Error: " + m.ifaceErr.Error()))
		return panelStyle.Width(width).Render(b.String())
	}
	if m.interfaces == nil {
		b.WriteString(dimStyle.Render("Loading..."))
		return panelStyle.Width(width).Render(b.String())
	}

	if len(m.interfaces) == 0 {
		b.WriteString(dimStyle.Render("No interfaces configured"))
		return panelStyle.Width(width).Render(b.String())
	}

	// Calculate column widths based on available space
	availWidth := width - 8
	nameW := 16
	stateW := 6
	zoneW := 12
	ipW := availWidth - nameW - stateW - zoneW - 6

	if ipW < 10 {
		ipW = 15
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Bold(true)

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s",
		nameW, "Interface",
		stateW, "State",
		zoneW, "Zone",
		ipW, "IP Address")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", minInt(availWidth, len(header)))))
	b.WriteString("\n")

	// Show interfaces, prioritizing those with IPs
	maxRows := 8
	shown := 0
	upWithIP := []models.Interface{}
	upNoIP := []models.Interface{}
	downIfaces := []models.Interface{}

	for _, iface := range m.interfaces {
		if iface.State == "up" {
			if iface.IP != "" {
				upWithIP = append(upWithIP, iface)
			} else {
				upNoIP = append(upNoIP, iface)
			}
		} else {
			downIfaces = append(downIfaces, iface)
		}
	}

	// Display order: up with IP, up without IP, down
	displayOrder := append(upWithIP, upNoIP...)
	displayOrder = append(displayOrder, downIfaces...)

	for _, iface := range displayOrder {
		if shown >= maxRows {
			break
		}

		stateStr := "up"
		stStyle := highlightStyle
		if iface.State != "up" {
			stateStr = "down"
			stStyle = dimStyle
		}

		zone := iface.Zone
		if zone == "" {
			zone = "-"
		}

		ip := iface.IP
		if ip == "" {
			ip = "-"
		}

		row := fmt.Sprintf("%-*s %s %-*s %-*s",
			nameW, truncateDash(iface.Name, nameW),
			stStyle.Render(fmt.Sprintf("%-*s", stateW, stateStr)),
			zoneW, truncateDash(zone, zoneW),
			ipW, truncateDash(ip, ipW))
		b.WriteString(row)
		b.WriteString("\n")
		shown++
	}

	if len(m.interfaces) > maxRows {
		remaining := len(m.interfaces) - maxRows
		b.WriteString(dimStyle.Render(fmt.Sprintf("... and %d more", remaining)))
	}

	return panelStyle.Width(width).Render(b.String())
}

// Helper functions

func formatRow(label, value string, labelWidth int) string {
	if value == "" {
		return ""
	}
	return labelStyle.Width(labelWidth).Render(label+":") + " " + valueStyle.Render(value) + "\n"
}

func renderBar(percent float64, width int, color string) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))

	bar := strings.Builder{}
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString(filledStyle.Render("█"))
		} else {
			bar.WriteString(emptyStyle.Render("░"))
		}
	}
	return bar.String()
}

func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

func formatThroughput(kbps int64) string {
	if kbps == 0 {
		return "0 Kbps"
	}
	if kbps >= 1_000_000 {
		return fmt.Sprintf("%.1f Gbps", float64(kbps)/1_000_000)
	}
	if kbps >= 1_000 {
		return fmt.Sprintf("%.1f Mbps", float64(kbps)/1_000)
	}
	return fmt.Sprintf("%d Kbps", kbps)
}

func truncateDash(s string, maxLen int) string {
	return truncateEllipsis(s, maxLen)
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}
