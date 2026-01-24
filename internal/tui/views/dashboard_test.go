package views

import (
	"errors"
	"testing"

	"github.com/jp2195/pyre/internal/models"
)

func TestNewDashboardModel(t *testing.T) {
	m := NewDashboardModel()

	if m.width != 0 {
		t.Errorf("expected width=0, got %d", m.width)
	}
	if m.height != 0 {
		t.Errorf("expected height=0, got %d", m.height)
	}
	if m.systemInfo != nil {
		t.Error("expected systemInfo=nil")
	}
}

func TestDashboardModel_SetSize(t *testing.T) {
	m := NewDashboardModel()
	m = m.SetSize(100, 50)

	if m.width != 100 {
		t.Errorf("expected width=100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height=50, got %d", m.height)
	}
}

func TestDashboardModel_SetSystemInfo(t *testing.T) {
	m := NewDashboardModel()

	info := &models.SystemInfo{
		Hostname: "fw01",
		Model:    "PA-3220",
	}
	m = m.SetSystemInfo(info, nil)

	if m.systemInfo != info {
		t.Error("expected systemInfo to be set")
	}
	if m.sysInfoErr != nil {
		t.Error("expected no error")
	}

	// With error
	err := errors.New("API error")
	m = m.SetSystemInfo(nil, err)
	if m.sysInfoErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetResources(t *testing.T) {
	m := NewDashboardModel()

	res := &models.Resources{
		CPUPercent:    50.0,
		MemoryPercent: 75.0,
	}
	m = m.SetResources(res, nil)

	if m.resources != res {
		t.Error("expected resources to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetResources(nil, err)
	if m.resourceErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetSessionInfo(t *testing.T) {
	m := NewDashboardModel()

	info := &models.SessionInfo{
		ActiveCount: 1000,
		MaxCount:    5000,
	}
	m = m.SetSessionInfo(info, nil)

	if m.sessionInfo != info {
		t.Error("expected sessionInfo to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetSessionInfo(nil, err)
	if m.sessionErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetHAStatus(t *testing.T) {
	m := NewDashboardModel()

	status := &models.HAStatus{
		Enabled:   true,
		State:     "active",
		PeerState: "passive",
	}
	m = m.SetHAStatus(status, nil)

	if m.haStatus != status {
		t.Error("expected haStatus to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetHAStatus(nil, err)
	if m.haErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetInterfaces(t *testing.T) {
	m := NewDashboardModel()

	ifaces := []models.Interface{
		{Name: "ethernet1/1", State: "up"},
		{Name: "ethernet1/2", State: "down"},
	}
	m = m.SetInterfaces(ifaces, nil)

	if len(m.interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(m.interfaces))
	}

	// With error
	err := errors.New("API error")
	m = m.SetInterfaces(nil, err)
	if m.ifaceErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardModel_SetThreatSummary(t *testing.T) {
	m := NewDashboardModel()

	summary := &models.ThreatSummary{
		TotalThreats:  100,
		CriticalCount: 5,
		HighCount:     20,
	}
	m = m.SetThreatSummary(summary, nil)

	if m.threatSummary != summary {
		t.Error("expected threatSummary to be set")
	}

	// With error
	err := errors.New("API error")
	m = m.SetThreatSummary(nil, err)
	if m.threatErr != err {
		t.Error("expected error to be set")
	}
}

func TestDashboardName(t *testing.T) {
	tests := []struct {
		dt   DashboardType
		want string
	}{
		{DashboardMain, "Main"},
		{DashboardNetwork, "Network"},
		{DashboardSecurity, "Security"},
		{DashboardVPN, "VPN"},
		{DashboardConfig, "Config"},
		{DashboardType(99), "Main"}, // Unknown type defaults to Main
	}

	for _, tt := range tests {
		got := DashboardName(tt.dt)
		if got != tt.want {
			t.Errorf("DashboardName(%d) = %q, want %q", tt.dt, got, tt.want)
		}
	}
}

func TestDashboardType_Constants(t *testing.T) {
	if DashboardMain != 0 {
		t.Errorf("expected DashboardMain=0, got %d", DashboardMain)
	}
	if DashboardNetwork != 1 {
		t.Errorf("expected DashboardNetwork=1, got %d", DashboardNetwork)
	}
	if DashboardSecurity != 2 {
		t.Errorf("expected DashboardSecurity=2, got %d", DashboardSecurity)
	}
	if DashboardVPN != 3 {
		t.Errorf("expected DashboardVPN=3, got %d", DashboardVPN)
	}
	if DashboardConfig != 4 {
		t.Errorf("expected DashboardConfig=4, got %d", DashboardConfig)
	}
}
