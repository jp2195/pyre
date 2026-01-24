package tui

import (
	"github.com/jp2195/pyre/internal/models"
	"github.com/jp2195/pyre/internal/troubleshoot"
	"github.com/jp2195/pyre/internal/tui/views"
)

// Message types for the TUI application

type SystemInfoMsg struct {
	Info *models.SystemInfo
	Err  error
}

type ResourcesMsg struct {
	Resources *models.Resources
	Err       error
}

type SessionInfoMsg struct {
	Info *models.SessionInfo
	Err  error
}

type SessionsMsg struct {
	Sessions []models.Session
	Err      error
}

type PoliciesMsg struct {
	Policies []models.SecurityRule
	Err      error
}

type NATPoliciesMsg struct {
	Rules []models.NATRule
	Err   error
}

type HAStatusMsg struct {
	Status *models.HAStatus
	Err    error
}

type InterfacesMsg struct {
	Interfaces []models.Interface
	Err        error
}

type ThreatSummaryMsg struct {
	Summary *models.ThreatSummary
	Err     error
}

type GlobalProtectMsg struct {
	Info *models.GlobalProtectInfo
	Err  error
}

type LoggedInAdminsMsg struct {
	Admins []models.LoggedInAdmin
	Err    error
}

type LicensesMsg struct {
	Licenses []models.LicenseInfo
	Err      error
}

type JobsMsg struct {
	Jobs []models.Job
	Err  error
}

type LoginSuccessMsg struct {
	Name     string
	APIKey   string
	Username string
	Password string
}

type LoginErrorMsg struct {
	Err error
}

type RefreshTickMsg struct{}

type ErrorMsg struct {
	Err error
}

type TroubleshootResultMsg struct {
	Result *troubleshoot.RunbookResult
	Err    error
}

type TroubleshootStepMsg struct {
	StepIndex int
	Status    troubleshoot.StepStatus
	Output    string
}

type ManagedDevicesMsg struct {
	Devices []models.ManagedDevice
	Err     error
}

type PanoramaDetectedMsg struct {
	IsPanorama bool
	Model      string
}

type SSHConnectedMsg struct {
	ConnectionName string
}

type SSHErrorMsg struct {
	ConnectionName string
	Err            error
}

type SystemLogsMsg struct {
	Logs []models.SystemLogEntry
	Err  error
}

type TrafficLogsMsg struct {
	Logs []models.TrafficLogEntry
	Err  error
}

type ThreatLogsMsg struct {
	Logs []models.ThreatLogEntry
	Err  error
}

type DashboardSelectedMsg struct {
	Dashboard views.DashboardType
}

// SwitchViewMsg requests switching to a specific view
type SwitchViewMsg struct {
	View ViewState
}

// SwitchDashboardMsg requests switching to a specific dashboard
type SwitchDashboardMsg struct {
	Dashboard views.DashboardType
}

// ShowPickerMsg requests showing the firewall picker
type ShowPickerMsg struct{}

// RefreshMsg requests a refresh of the current view
type RefreshMsg struct{}

// ShowHelpMsg requests showing the help overlay
type ShowHelpMsg struct{}

type DiskUsageMsg struct {
	Disks []models.DiskUsage
	Err   error
}

type EnvironmentalsMsg struct {
	Environmentals []models.Environmental
	Err            error
}

type CertificatesMsg struct {
	Certificates []models.Certificate
	Err          error
}

type ARPTableMsg struct {
	Entries []models.ARPEntry
	Err     error
}

type RoutingTableMsg struct {
	Routes []models.RouteEntry
	Err    error
}

type IPSecTunnelsMsg struct {
	Tunnels []models.IPSecTunnel
	Err     error
}

type GlobalProtectUsersMsg struct {
	Users []models.GlobalProtectUser
	Err   error
}

type PendingChangesMsg struct {
	Changes []models.PendingChange
	Err     error
}

type NATPoolMsg struct {
	Pools []models.NATPoolInfo
	Err   error
}
