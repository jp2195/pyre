package tui

import (
	"github.com/jp2195/pyre/internal/config"
	"github.com/jp2195/pyre/internal/models"
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

type SessionDetailMsg struct {
	Detail *models.SessionDetail
	Err    error
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
	Host     string
	APIKey   string
	Username string
	Insecure bool
	// Password is intentionally not included - credentials should not persist in messages.
}

type LoginErrorMsg struct {
	Err error
}

type RefreshTickMsg struct{}

type ErrorMsg struct {
	Err error
}

// ErrorDismissMsg is sent after a timeout to clear the error from the footer
type ErrorDismissMsg struct{}

type ManagedDevicesMsg struct {
	Devices []models.ManagedDevice
	Err     error
}

type PanoramaDetectedMsg struct {
	IsPanorama bool
	Model      string
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

//nolint:misspell // "environmentals" is the PAN-OS XML API tag name
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

type BGPNeighborsMsg struct {
	Neighbors []models.BGPNeighbor
	Err       error
}

type OSPFNeighborsMsg struct {
	Neighbors []models.OSPFNeighbor
	Err       error
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

// Connection Hub messages

// ConnectionSelectedMsg is sent when a connection is selected from the hub
type ConnectionSelectedMsg struct {
	Host   string
	Config config.ConnectionConfig
}

// ConnectionFormSubmitMsg is sent when the connection form is submitted
type ConnectionFormSubmitMsg struct {
	Host         string
	Config       config.ConnectionConfig
	SaveToConfig bool
	Mode         views.FormMode
}

// ConnectionDeletedMsg is sent when a connection is deleted
type ConnectionDeletedMsg struct {
	Host string
}

// ConfigSavedMsg is sent after config is saved
type ConfigSavedMsg struct {
	Err error
}

// StateSavedMsg is sent after state is saved
type StateSavedMsg struct {
	Err error
}

// ShowConnectionHubMsg requests showing the connection hub
type ShowConnectionHubMsg struct{}

// ShowConnectionFormMsg requests showing the connection form
type ShowConnectionFormMsg struct {
	Mode   views.FormMode
	Host   string                  // For edit mode
	Config config.ConnectionConfig // For edit mode
}
