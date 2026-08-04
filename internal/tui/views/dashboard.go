package views

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jp2195/pyre/internal/models"
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
	DashboardBase

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
	environmentals []models.Environmental //nolint:misspell // "environmentals" is the PAN-OS XML API tag name
	certificates   []models.Certificate
	natPools       []models.NATPoolInfo

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
	natPoolErr  error
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{}
}

// SetSpinnerFrame sets the current spinner animation frame
func (m DashboardModel) SetSpinnerFrame(frame string) DashboardModel {
	m.SpinnerFrame = frame
	return m
}

func (m DashboardModel) SetSize(width, height int) DashboardModel {
	m.Width = width
	m.Height = height
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

func (m DashboardModel) SetNATPoolInfo(pools []models.NATPoolInfo, err error) DashboardModel {
	m.natPools = pools
	m.natPoolErr = err
	return m
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	return m, nil
}

// HasData returns true if the dashboard has already loaded its data
func (m DashboardModel) HasData() bool {
	return m.systemInfo != nil
}

func (m DashboardModel) View() string {
	if m.Width == 0 {
		return RenderLoadingInline(m.SpinnerFrame, "Loading...")
	}

	totalWidth, leftColWidth, rightColWidth := m.ColumnWidths()

	if m.IsNarrow() {
		return m.renderSingleColumn(totalWidth)
	}

	// Build left column - "Device Health"
	// System info, resources, sessions, disk, hardware
	leftPanels := []string{
		m.renderSystemInfo(leftColWidth),
		m.renderResourcesCompact(leftColWidth),
		m.renderSessionsCompact(leftColWidth),
	}

	// Add disk usage panel to left column (health metric)
	if len(m.diskUsage) > 0 {
		leftPanels = append(leftPanels, m.renderDiskUsage(leftColWidth))
	}

	// Add hardware status panel to left column (health metric)
	if len(m.environmentals) > 0 { //nolint:misspell // "environmentals" is the PAN-OS XML API tag name
		leftPanels = append(leftPanels, m.renderEnvironmentals(leftColWidth))
	}

	// Build right column - "Operations & Security"
	// HA at top (critical), NAT pools, content, licenses, etc.
	var rightPanels []string

	// HA Status at top of right column when enabled (high priority)
	if m.haStatus != nil && m.haStatus.Enabled {
		rightPanels = append(rightPanels, m.renderHAStatus(rightColWidth))
	}

	// NAT Pool Utilization
	if len(m.natPools) > 0 {
		rightPanels = append(rightPanels, m.renderNATPoolUtilization(rightColWidth))
	}

	// Content Versions
	rightPanels = append(rightPanels, m.renderContentVersions(rightColWidth))

	// Licenses
	if len(m.licenses) > 0 {
		rightPanels = append(rightPanels, m.renderLicenses(rightColWidth))
	}

	// Threat summary if we have threat data
	if m.threatSummary != nil && m.threatSummary.TotalThreats > 0 {
		rightPanels = append(rightPanels, m.renderThreatSummary(rightColWidth))
	}

	// Admins Online
	if len(m.admins) > 0 {
		rightPanels = append(rightPanels, m.renderLoggedInAdmins(rightColWidth))
	}

	// GlobalProtect if configured
	if m.gpInfo != nil && (m.gpInfo.ActiveUsers > 0 || m.gpInfo.TotalGateways > 0) {
		rightPanels = append(rightPanels, m.renderGlobalProtect(rightColWidth))
	}

	// Recent Jobs
	if len(m.jobs) > 0 {
		rightPanels = append(rightPanels, m.renderJobs(rightColWidth))
	}

	// Certificates (expiring/expired)
	if len(m.certificates) > 0 {
		rightPanels = append(rightPanels, m.renderCertificates(rightColWidth))
	}

	return m.RenderTwoColumn(leftPanels, rightPanels)
}

func (m DashboardModel) renderSingleColumn(width int) string {
	panels := []string{
		m.renderSystemInfo(width),
		m.renderResourcesCompact(width),
		m.renderSessionsCompact(width),
	}

	// Disk usage (health)
	if len(m.diskUsage) > 0 {
		panels = append(panels, m.renderDiskUsage(width))
	}

	// Hardware status (health)
	if len(m.environmentals) > 0 { //nolint:misspell // "environmentals" is the PAN-OS XML API tag name
		panels = append(panels, m.renderEnvironmentals(width))
	}

	// HA Status (critical when enabled)
	if m.haStatus != nil && m.haStatus.Enabled {
		panels = append(panels, m.renderHAStatus(width))
	}

	// NAT Pool Utilization
	if len(m.natPools) > 0 {
		panels = append(panels, m.renderNATPoolUtilization(width))
	}

	if len(m.licenses) > 0 {
		panels = append(panels, m.renderLicenses(width))
	}

	if len(m.jobs) > 0 {
		panels = append(panels, m.renderJobs(width))
	}

	if len(m.certificates) > 0 {
		panels = append(panels, m.renderCertificates(width))
	}

	return m.RenderSingleColumn(panels)
}
